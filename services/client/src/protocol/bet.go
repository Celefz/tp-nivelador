package protocol

import (
	"encoding/binary"
	"fmt"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/lottery"
)

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
	betData = binary.BigEndian.AppendUint32(betData, bet.Number)   // 4B

	return betData
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
	number := binary.BigEndian.Uint32(betData[numberStart:betLen])

	return lottery.Bet{
		AgencyId:  agencyId,
		FirstName: firstName,
		LastName:  lastName,
		Document:  document,
		Birthdate: birthdate,
		Number:    number,
	}, betLen, nil
}
