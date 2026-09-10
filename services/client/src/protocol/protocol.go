package protocol

import (
	"encoding/binary"
	"fmt"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/lottery"
)

const (
	TYPE_BET byte = 1
	TYPE_END byte = 2

	HEADER_LEN     = 4
	MIN_PACKET_LEN = 20
)

func is_valid_type(t byte) bool {
	return t == TYPE_BET || t == TYPE_END
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

func IsEndPacket(data []byte) bool {
	return len(data) == 1 && data[0] == TYPE_END
}

func SerializeBet(bet lottery.Bet) []byte {
	betData := make([]byte, 0, 100)

	firstNameLen := len(bet.FirstName)
	lastNameLen := len(bet.LastName)

	payloadLen := MIN_PACKET_LEN + firstNameLen + lastNameLen
	betData = binary.BigEndian.AppendUint16(betData, uint16(payloadLen)) // 2B

	betData = append(betData, TYPE_BET)            // 1B
	betData = append(betData, bet.AgencyId)        // 1B
	betData = append(betData, uint8(firstNameLen)) // 1B
	betData = append(betData, uint8(lastNameLen))  // 1B

	betData = append(betData, []byte(bet.FirstName)...)            // Variable length
	betData = append(betData, []byte(bet.LastName)...)             // Variable length
	betData = append(betData, []byte(bet.Birthdate)...)            // 10B
	betData = binary.BigEndian.AppendUint32(betData, bet.Document) // 4B
	betData = binary.BigEndian.AppendUint16(betData, bet.Number)   // 2B

	return betData
}

func DeserializeBet(betData []byte) (lottery.Bet, error) {
	if len(betData) < HEADER_LEN {
		return lottery.Bet{}, fmt.Errorf("packet too short")
	}

	messageType, err := ParseMessageType(betData)
	if err != nil {
		return lottery.Bet{}, err
	}

	if messageType != TYPE_BET {
		return lottery.Bet{}, fmt.Errorf("invalid bet type: %d", betData[0])
	}

	agencyId := betData[1]
	firstNameLen := int(betData[2])
	lastNameLen := int(betData[3])

	if len(betData) != MIN_PACKET_LEN+firstNameLen+lastNameLen {
		return lottery.Bet{}, fmt.Errorf("invalid bet data: wrong size")
	}

	const firstNameStart = 4
	const birthdateLen = 10
	const documentLen = 4

	lastNameStart := firstNameStart + firstNameLen
	birthdateStart := lastNameStart + lastNameLen
	documentStart := birthdateStart + birthdateLen
	numberStart := documentStart + documentLen

	firstName := string(betData[firstNameStart:lastNameStart])
	lastName := string(betData[lastNameStart:birthdateStart])
	birthdate := string(betData[birthdateStart:documentStart])
	document := binary.BigEndian.Uint32(betData[documentStart:numberStart])
	number := binary.BigEndian.Uint16(betData[numberStart:])

	return lottery.Bet{
		AgencyId:  agencyId,
		FirstName: firstName,
		LastName:  lastName,
		Document:  document,
		Birthdate: birthdate,
		Number:    number,
	}, nil
}
