from .message import BET_HEADER_LEN, MIN_BET_LEN


class Bet:
    def __init__(self, agency_id: int, first_name: str, last_name: str, birthdate: str, document: int, number: int):
        self.agency_id = agency_id
        self.first_name = first_name
        self.last_name = last_name
        self.birthdate = birthdate
        self.document = document
        self.number = number


def serialize_bet(bet: Bet) -> bytes:
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
    bet_data.extend(bet.number.to_bytes(4, byteorder="big"))

    return bytes(bet_data)


def deserialize_bet(bet_data: bytes, agency_id: int):
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
