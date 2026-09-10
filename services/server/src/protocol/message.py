TYPE_BETS = 1
TYPE_END = 2

BATCH_HEADER_LEN = 2
BET_HEADER_LEN = 2
MIN_BET_LEN = 20


def parse_message_type(packet: bytes) -> int:
    message_type = packet[0]
    if message_type not in (TYPE_BETS, TYPE_END):
        raise ValueError(f"invalid message type: {message_type}")

    return message_type


def serialize_end() -> bytes:
    payload = bytearray([TYPE_END])
    return len(payload).to_bytes(2, byteorder="big") + bytes(payload)


def is_end_packet(packet: bytes) -> bool:
    return len(packet) == 2 and packet[0] == TYPE_END
