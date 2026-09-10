package lottery

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	FIELD_AMOUNT = 5
	BASE_10      = 10
	BITS_32      = 32
	BITS_16      = 16
)

type Bet struct {
	AgencyId  uint8
	FirstName string
	LastName  string
	Document  uint32
	Birthdate string
	Number    uint16
}

func parseBetFromCsv(line string, agencyId uint8) (Bet, error) {
	fields := strings.Split(line, ",")

	if len(fields) != FIELD_AMOUNT {
		return Bet{}, fmt.Errorf("invalid bet line: %s", line)
	}

	document, err := strconv.ParseUint(fields[2], BASE_10, BITS_32)
	if err != nil {
		return Bet{}, fmt.Errorf("invalid document number: %s", fields[2])
	}

	number, err := strconv.ParseUint(fields[4], BASE_10, BITS_16)
	if err != nil {
		return Bet{}, fmt.Errorf("invalid bet number: %s", fields[4])
	}

	bet := Bet{
		AgencyId:  agencyId,
		FirstName: fields[0],
		LastName:  fields[1],
		Document:  uint32(document),
		Birthdate: fields[3],
		Number:    uint16(number),
	}

	return bet, nil
}

func ParseBetFromCSV(line string, agencyId uint8) (Bet, error) {
	return parseBetFromCsv(line, agencyId)
}

func parseBetToCsv(bet Bet) string {
	return fmt.Sprintf(
		"%s,%s,%d,%s,%d",
		bet.FirstName,
		bet.LastName,
		bet.Document,
		bet.Birthdate,
		bet.Number,
	)
}

func ParseBetToCSV(bet Bet) string {
	return parseBetToCsv(bet)
}
