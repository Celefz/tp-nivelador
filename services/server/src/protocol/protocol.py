from lottery import Bet

TYPE_BET = 1
TYPE_END = 2

BATCH_HEADER_LEN = 2
BET_HEADER_LEN = 2
MIN_BET_LEN = 18

def parse_message_type(packet: bytes) -> int:
    message_type = packet[0]
    if message_type not in (TYPE_BET, TYPE_END):
        raise ValueError(f"invalid message type: {message_type}")

    return message_type


def serialize_end() -> bytes:
    payload = bytearray([TYPE_END])
    return len(payload).to_bytes(2, byteorder="big") + bytes(payload)


def is_end_packet(packet: bytes) -> bool:
    return len(packet) == 2 and packet[0] == TYPE_END


def serialize_batch(bets: list[Bet], agency_id: int) -> bytes:
    bet_data = bytearray((TYPE_BET, agency_id))
    for bet in bets:
        bet_data.extend(serialize_bet(bet))
    bet_data = len(bet_data).to_bytes(2, byteorder="big") + bet_data
    return bytes(bet_data)


def serialize_bet(bet) -> bytes:
    first_name = bet.first_name.encode("utf-8")
    last_name = bet.last_name.encode("utf-8")
    birthdate = bet.birthdate.encode("utf-8")

    first_name_len = len(first_name)
    last_name_len = len(last_name)

    bet_data = bytearray()
    bet_data.append(first_name_len)
    bet_data.append(last_name_len)

    bet_data.extend(first_name)
    bet_data.extend(last_name)
    bet_data.extend(birthdate)
    bet_data.extend(bet.document.to_bytes(4, byteorder="big"))
    bet_data.extend(bet.number.to_bytes(2, byteorder="big"))

    return bytes(bet_data)


def deserialize_batch(batch_data: bytes) -> list[Bet]:
    if len(batch_data) < BATCH_HEADER_LEN:
        raise ValueError("packet too short")

    message_type = parse_message_type(batch_data)
    if message_type != TYPE_BET:
        raise ValueError(f"invalid bet type: {batch_data[0]}")

    agency_id = batch_data[1]
    bets = []
    offset = BATCH_HEADER_LEN

    while offset < len(batch_data):
        bet, size = deserialize_bet(batch_data[offset:], agency_id)
        bets.append(bet)
        offset += size
    return bets


def deserialize_bet(bet_data: bytes, agency_id: int) -> tuple[Bet, int]:
    if len(bet_data) < BET_HEADER_LEN:
        raise ValueError("packet too short")

    first_name_len = bet_data[0]
    last_name_len = bet_data[1]
    bet_size = MIN_BET_LEN + first_name_len + last_name_len

    if len(bet_data) < bet_size:
        raise ValueError("invalid bet data: wrong size")

    first_name_start = 2
    birthdate_len = 10
    document_len = 4

    last_name_start = first_name_start + first_name_len
    birthdate_start = last_name_start + last_name_len
    document_start = birthdate_start + birthdate_len
    number_start = document_start + document_len

    return Bet(
        agency_id=agency_id,
        first_name=bet_data[first_name_start:last_name_start].decode("utf-8"),
        last_name=bet_data[last_name_start:birthdate_start].decode("utf-8"),
        birthdate=bet_data[birthdate_start:document_start].decode("utf-8"),
        document=int.from_bytes(bet_data[document_start:number_start], byteorder="big"),
        number=int.from_bytes(bet_data[number_start:bet_size], byteorder="big"),
    ), bet_size
