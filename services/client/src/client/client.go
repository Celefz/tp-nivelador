package client

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/lottery"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/protocol"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const CONNECTION_ATTEMPTS_MAX = 3
const CONNECTION_ATTEMPS_DELAY_MS = 1000

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile  string
	OutputFile string
	BatchSize  string
}

type Client struct {
	conn   net.Conn
	config ClientConfig
}

func recvPacket(conn net.Conn) ([]byte, error) {
	header, err := safe_socket.RecvAll(conn, 2)
	if err != nil {
		return nil, err
	}
	if len(header) != 2 {
		return nil, io.ErrUnexpectedEOF
	}

	packetLen := int(binary.BigEndian.Uint16(header))
	if packetLen == 0 {
		return nil, fmt.Errorf("invalid empty packet")
	}

	packet, err := safe_socket.RecvAll(conn, packetLen)
	if err != nil {
		return nil, err
	}
	if len(packet) != packetLen {
		return nil, io.ErrUnexpectedEOF
	}

	return packet, nil
}

func NewClient(config ClientConfig) (*Client, error) {
	conn, err := connectToServer(config.ServerHost, config.ServerPort)
	if err != nil {
		logger.Warn("connect-to-server", logger.Fail)
		return nil, err
	}

	return &Client{conn: conn, config: config}, nil
}

func connectToServer(host, port string) (net.Conn, error) {
	const action = "connect-to-server"
	var err error
	var conn net.Conn

	logger.Info(action, logger.InProgress)
	for i := range CONNECTION_ATTEMPTS_MAX {
		conn, err = net.Dial("tcp", host+":"+port)
		if err != nil {
			logger.Warn(action, logger.Fail, "attempt", i)
			time.Sleep(CONNECTION_ATTEMPS_DELAY_MS * time.Millisecond)
			continue
		}

		logger.Info(action, logger.Success)
		break
	}

	return conn, err
}

func (client *Client) closeConn() {
	if client.conn != nil {
		_ = client.conn.Close()
	}
}

func shouldIgnoreShutdownError(err error, ctx context.Context) bool {
	if ctx != nil && ctx.Err() != nil {
		return true
	}

	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed)
}

func (client *Client) Run() error {
	return client.RunWithContext(context.Background())
}

func (client *Client) RunWithContext(ctx context.Context) error {
	const mainAction = "send-bets"
	defer client.closeConn()

	if ctx == nil {
		ctx = context.Background()
	}

	go func() {
		<-ctx.Done()
		client.closeConn()
	}()

	clientArgs := []any{"agency-id", client.config.AgencyId}

	agencyID, err := client.parseAgencyID()
	if err != nil {
		return err
	}

	inputFile, err := os.Open(client.config.InputFile)
	if err != nil {
		logger.Error("open-input-file", logger.Fail, clientArgs...)
		return err
	}
	defer inputFile.Close()

	batchSize, err := client.parseBatchSize()

	if err != nil {
		return err
	}

	if err := client.sendBets(inputFile, agencyID, batchSize, clientArgs, ctx); err != nil {
		if shouldIgnoreShutdownError(err, ctx) {
			return nil
		}
		return err
	}

	outputFile, err := os.Create(client.config.OutputFile)

	if err != nil {
		logger.Error("create-output-file", logger.Fail, clientArgs...)
		return err
	}

	defer outputFile.Close()

	if err := client.recvWinners(outputFile, clientArgs, ctx); err != nil {
		if shouldIgnoreShutdownError(err, ctx) {
			return nil
		}
		return err
	}

	logger.Info(mainAction, logger.Success, clientArgs...)
	return nil
}

func (client *Client) parseAgencyID() (uint8, error) {
	agencyID, err := strconv.ParseUint(client.config.AgencyId, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid agency id %q: %w", client.config.AgencyId, err)
	}

	return uint8(agencyID), nil
}

func (client *Client) parseBatchSize() (int, error) {
	batchSize, err := strconv.ParseInt(client.config.BatchSize, 10, 16)
	if err != nil || batchSize == 0 {
		return 0, fmt.Errorf("invalid batch size %q", client.config.BatchSize)
	}

	return int(batchSize), nil
}

func (client *Client) sendBets(inputFile io.Reader, agencyID uint8, batchSize int, clientArgs []any, ctx context.Context) error {
	batch := make([]lottery.Bet, 0, batchSize)
	scanner := bufio.NewScanner(inputFile)

	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}

		logger.Info("send-bets", logger.InProgress, clientArgs...)

		bet, err := lottery.ParseBetFromCsv(scanner.Text(), agencyID)
		if err != nil {
			logger.Error("parse-bet", logger.Fail, clientArgs...)
			return err
		}

		batch = append(batch, bet)
		if len(batch) == batchSize {
			if err := client.sendBatch(batch, agencyID, clientArgs, ctx); err != nil {
				if shouldIgnoreShutdownError(err, ctx) {
					return nil
				}
				return err
			}
			batch = batch[:0]
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("scan-input-file", logger.Fail, clientArgs...)
		return err
	}

	if len(batch) > 0 {
		if err := client.sendBatch(batch, agencyID, clientArgs, ctx); err != nil {
			if shouldIgnoreShutdownError(err, ctx) {
				return nil
			}
			return err
		}
	}

	if err := client.sendEnd(agencyID, clientArgs, ctx); err != nil {
		if shouldIgnoreShutdownError(err, ctx) {
			return nil
		}
		return err
	}

	return nil
}

func (client *Client) sendBatch(batch []lottery.Bet, agencyID uint8, clientArgs []any, ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

	packet := protocol.SerializeBatch(batch, agencyID)
	if err := safe_socket.SendAll(client.conn, packet); err != nil {
		if shouldIgnoreShutdownError(err, ctx) {
			return nil
		}
		logger.Error("send-batch", logger.Fail, clientArgs...)
		return err
	}

	if err := client.recvAck(agencyID, clientArgs, ctx); err != nil {
		if shouldIgnoreShutdownError(err, ctx) {
			return nil
		}
		logger.Error("recv-ack", logger.Fail, clientArgs...)
		return err
	}

	return nil
}

func (client *Client) recvAck(agencyID uint8, clientArgs []any, ctx context.Context) error {
	packet, err := recvPacket(client.conn)
	if err != nil {
		if shouldIgnoreShutdownError(err, ctx) {
			return nil
		}
		logger.Error("recv-ack", logger.Fail, clientArgs...)
		return err
	}

	if !protocol.IsAckPacket(packet) {
		return fmt.Errorf("unexpected packet while waiting for ack: type=%d len=%d", packet[0], len(packet))
	}

	if len(packet) != 2 || packet[1] != agencyID {
		return fmt.Errorf("unexpected ack agency id: got=%d expected=%d", packet[1], agencyID)
	}

	return nil
}

func (client *Client) sendEnd(agencyID uint8, clientArgs []any, ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

	if err := safe_socket.SendAll(client.conn, protocol.SerializeEnd(agencyID)); err != nil {
		if shouldIgnoreShutdownError(err, ctx) {
			return nil
		}
		logger.Error("send-end", logger.Fail, clientArgs...)
		return err
	}

	return nil
}

func (client *Client) recvWinners(outputFile io.Writer, clientArgs []any, ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		packet, err := recvPacket(client.conn)
		if err != nil {
			if shouldIgnoreShutdownError(err, ctx) {
				return nil
			}
			logger.Error("recv-winner", logger.Fail, clientArgs...)
			return err
		}

		if protocol.IsEndPacket(packet) {
			return nil
		}

		winners, err := protocol.DeserializeBatch(packet)
		if err != nil {
			if shouldIgnoreShutdownError(err, ctx) {
				return nil
			}
			logger.Error("deserialize-winner", logger.Fail, clientArgs...)
			return err
		}

		for _, winner := range winners {
			if ctx.Err() != nil {
				return nil
			}

			if _, err := fmt.Fprintln(outputFile, lottery.ParseBetToCsv(winner)); err != nil {
				if shouldIgnoreShutdownError(err, ctx) {
					return nil
				}
				logger.Error("write-output-file", logger.Fail, clientArgs...)
				return err
			}
		}
	}
}
