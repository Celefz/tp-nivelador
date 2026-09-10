package protocol

import (
	"encoding/binary"
	"fmt"
)

const (
	TYPE_BETS byte = 1
	TYPE_END  byte = 2
	TYPE_ACK  byte = 3

	BATCH_HEADER_LEN = 2
	BET_HEADER_LEN   = 2
	MIN_BET_LEN      = 20
)

func is_valid_type(t byte) bool {
	return t == TYPE_BETS || t == TYPE_END || t == TYPE_ACK
}

func ParseMessageType(packet []byte) (byte, error) {
	messageType := packet[0]
	if !is_valid_type(messageType) {
		return 0, fmt.Errorf("invalid message type: %d", packet[0])
	}

	return messageType, nil
}

func SerializeEnd(agencyID uint8) []byte {
	packet := make([]byte, 0, 4)
	packet = binary.BigEndian.AppendUint16(packet, 2)
	packet = append(packet, TYPE_END, agencyID)
	return packet
}

func SerializeAck(agencyID uint8) []byte {
	packet := make([]byte, 0, 4)
	packet = binary.BigEndian.AppendUint16(packet, 2)
	packet = append(packet, TYPE_ACK, agencyID)
	return packet
}

func IsEndPacket(data []byte) bool {
	return len(data) == 1 && data[0] == TYPE_END
}

func IsAckPacket(data []byte) bool {
	return len(data) == 2 && data[0] == TYPE_ACK
}
