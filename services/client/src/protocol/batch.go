package protocol

import (
	"encoding/binary"
	"fmt"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/lottery"
)

func SerializeBatch(bets []lottery.Bet, agencyId uint8) []byte {
	batchData := make([]byte, 0, MIN_BET_LEN)
	batchData = append(batchData, TYPE_BETS) // 1B
	batchData = append(batchData, agencyId)  // 1B

	for _, bet := range bets {
		batchData = append(batchData, SerializeBet(bet)...)
	}

	packet := make([]byte, 0, len(batchData)+2)
	packet = binary.BigEndian.AppendUint16(packet, uint16(len(batchData))) // 2B
	packet = append(packet, batchData...)

	return packet
}

func DeserializeBatch(batchData []byte) ([]lottery.Bet, error) {
	if len(batchData) < BATCH_HEADER_LEN {
		return nil, fmt.Errorf("packet too short")
	}

	messageType, err := ParseMessageType(batchData)
	if err != nil {
		return nil, err
	}

	if messageType != TYPE_BETS {
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
