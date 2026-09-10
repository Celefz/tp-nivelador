package protocol

import (
	"encoding/binary"
	"fmt"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/lottery"
)

const (
	TYPE_BETS byte = 1
	TYPE_END  byte = 2

	BATCH_HEADER_LEN = 2
	BET_HEADER_LEN   = 2
	MIN_BET_LEN      = 18
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

func SerializeBatch(bets []lottery.Bet, agencyId uint8) []byte {
	batchData := make([]byte, 0, 100)
	batchData = append(batchData, TYPE_BET) // 1B
	batchData = append(batchData, agencyId) // 1B

	for _, bet := range bets {
		batchData = append(batchData, SerializeBet(bet)...)
	}

	packet := make([]byte, 0, len(batchData)+2)
	packet = binary.BigEndian.AppendUint16(packet, uint16(len(batchData))) // 2B
	packet = append(packet, batchData...)

	return packet
}

func SerializeBet(bet lottery.Bet) []byte {
	betData := make([]byte, 0, MIN_BET_LEN)

	firstNameLen := len(bet.FirstName)
	lastNameLen := len(bet.LastName)

	betData = append(betData, uint8(firstNameLen)) // 1B
	betData = append(betData, uint8(lastNameLen))  // 1B

	betData = append(betData, []byte(bet.FirstName)...)            // Variable length
	betData = append(betData, []byte(bet.LastName)...)             // Variable length
	betData = append(betData, []byte(bet.Birthdate)...)            // 10B
	betData = binary.BigEndian.AppendUint32(betData, bet.Document) // 4B
	betData = binary.BigEndian.AppendUint16(betData, bet.Number)   // 2B

	return betData
}

func DeserializeBatch(batchData []byte) ([]lottery.Bet, error) {
	if len(batchData) < BATCH_HEADER_LEN {
		return nil, fmt.Errorf("packet too short")
	}

	messageType, err := ParseMessageType(batchData)

	if err != nil {
		return nil, err
	}

	if messageType != TYPE_BET {
		return nil, fmt.Errorf("invalid bet type: %d", batchData[0])
	}

	agencyId := batchData[1]
	bets := []lottery.Bet{}

	for offset := BATCH_HEADER_LEN; offset < len(batchData); {
		bet, size, err := DeserializeBet(batchData[offset:], agencyId)
		if err != nil {
			return nil, err
		}
		bets = append(bets, bet)
		offset += size
	}
	return bets, nil
}

func DeserializeBet(betData []byte, agencyId uint8) (lottery.Bet, int, error) {
	if len(betData) < BET_HEADER_LEN {
		return lottery.Bet{}, 0, fmt.Errorf("packet too short")
	}

	firstNameLen := int(betData[0])
	lastNameLen := int(betData[1])

	betLen := MIN_BET_LEN + firstNameLen + lastNameLen

	if len(betData) < betLen {
		return lottery.Bet{}, 0, fmt.Errorf("invalid bet data: wrong size")
	}

	const firstNameStart = 2
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
	number := binary.BigEndian.Uint16(betData[numberStart:betLen])

	return lottery.Bet{
		AgencyId:  agencyId,
		FirstName: firstName,
		LastName:  lastName,
		Document:  document,
		Birthdate: birthdate,
		Number:    number,
	}, betLen, nil
}
