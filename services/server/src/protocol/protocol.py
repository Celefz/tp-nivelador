from lottery import Bet

TYPE_BET = 1
TYPE_END = 2

HEADER_LEN = 4
MIN_PACKET_LEN = 20

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


def serialize_bet(bet) -> bytes:
    first_name = bet.first_name.encode("utf-8")
    last_name = bet.last_name.encode("utf-8")
    birthdate = bet.birthdate.encode("utf-8")

    first_name_len = len(first_name)
    last_name_len = len(last_name)

    payload_len = MIN_PACKET_LEN + first_name_len + last_name_len

    bet_data = bytearray()

    bet_data.extend(payload_len.to_bytes(2, byteorder="big"))

    bet_data.append(TYPE_BET)
    bet_data.append(bet.agency_id)
    bet_data.append(first_name_len)
    bet_data.append(last_name_len)

    bet_data.extend(first_name)
    bet_data.extend(last_name)
    bet_data.extend(birthdate)
    bet_data.extend(bet.document.to_bytes(4, byteorder="big"))
    bet_data.extend(bet.number.to_bytes(2, byteorder="big"))

    return bytes(bet_data)


def deserialize_bet(bet_data: bytes):
    if len(bet_data) < HEADER_LEN:
        raise ValueError("packet too short")

    message_type = parse_message_type(bet_data)
    if message_type != TYPE_BET:
        raise ValueError(f"invalid bet type: {bet_data[0]}")

    agency_id = bet_data[1]
    first_name_len = bet_data[2]
    last_name_len = bet_data[3]

    if len(bet_data) != MIN_PACKET_LEN + first_name_len + last_name_len:
        raise ValueError("invalid bet data: wrong size")

    first_name_start = 4
    birthdate_len = 10
    document_len = 4

    last_name_start = first_name_start + first_name_len
    birthdate_start = last_name_start + last_name_len
    document_start = birthdate_start + birthdate_len
    number_start = document_start + document_len

    first_name = bet_data[first_name_start:last_name_start].decode("utf-8")
    last_name = bet_data[last_name_start:birthdate_start].decode("utf-8")
    birthdate = bet_data[birthdate_start:document_start].decode("utf-8")

    document = int.from_bytes(
        bet_data[document_start:number_start],
        byteorder="big"
    )

    number = int.from_bytes(
        bet_data[number_start:],
        byteorder="big"
    )

    return Bet(
        agency_id=agency_id,
        first_name=first_name,
        last_name=last_name,
        document=document,
        birthdate=birthdate,
        number=number,
    )
