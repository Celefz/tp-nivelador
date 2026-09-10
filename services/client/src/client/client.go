package client

import (
	"bufio"
	"encoding/binary"
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
const CONNECTION_ATTEMPS_DELAY_MS = 200

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

	client := &Client{conn: conn, config: config}
	return client, nil
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

func (client *Client) Run() error {
	const mainAction = "send-bets"
	defer client.conn.Close()

	clientArgs := []any{"agency-id", client.config.AgencyId}
	agencyID, err := strconv.ParseUint(client.config.AgencyId, 10, 8)
	if err != nil {
		return fmt.Errorf("invalid agency id %q: %w", client.config.AgencyId, err)
	}

	inputFileName := client.config.InputFile
	inputFile, err := os.Open(inputFileName)

	if err != nil {
		logger.Error("open-input-file", logger.Fail, clientArgs...)
		return err
	}
	defer inputFile.Close()

	batchSize, err := strconv.ParseInt(client.config.BatchSize, 10, 16)

	if err != nil || batchSize == 0 {
		return fmt.Errorf("invalid batch size %q", client.config.BatchSize)
	}

	batch := make([]lottery.Bet, 0, int(batchSize))

	scanner := bufio.NewScanner(inputFile)

	for scanner.Scan() {
		logger.Info(mainAction, logger.InProgress, clientArgs...)
		bet, err := lottery.ParseBetFromCsv(scanner.Text(), uint8(agencyID))

		if err != nil {
			logger.Error("parse-bet", logger.Fail, clientArgs...)
			return err
		}

		batch = append(batch, bet)

		if len(batch) == int(batchSize) {
			packet := protocol.SerializeBatch(batch, uint8(agencyID))
			if err := safe_socket.SendAll(client.conn, packet); err != nil {
				logger.Error("send-batch", logger.Fail, clientArgs...)
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
		packet := protocol.SerializeBatch(batch, uint8(agencyID))
		if err := safe_socket.SendAll(client.conn, packet); err != nil {
			logger.Error("send-batch", logger.Fail, clientArgs...)
			return err
		}
	}

	if err := safe_socket.SendAll(client.conn, protocol.SerializeEnd(uint8(agencyID))); err != nil {
		logger.Error("send-end", logger.Fail, clientArgs...)
		return err
	}

	outputFileName := client.config.OutputFile
	outputFile, err := os.Create(outputFileName)

	if err != nil {
		logger.Error("create-output-file", logger.Fail, clientArgs...)
		return err
	}
	defer outputFile.Close()

	for {
		packet, err := recvPacket(client.conn)
		if err != nil {
			logger.Error("recv-winner", logger.Fail, clientArgs...)
			return err
		}

		if protocol.IsEndPacket(packet) {
			break
		}

		winners, err := protocol.DeserializeBatch(packet)

		if err != nil {
			logger.Error("deserialize-winner", logger.Fail, clientArgs...)
			return err
		}

		for _, winner := range winners {
			if _, err := fmt.Fprintln(outputFile, lottery.ParseBetToCsv(winner)); err != nil {
				logger.Error("write-output-file", logger.Fail, clientArgs...)
				return err
			}
		}
	}

	logger.Info(mainAction, logger.Success, clientArgs...)

	return nil
}
