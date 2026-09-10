from .bet import Bet, deserialize_bet, serialize_bet
from .message import TYPE_BETS, BATCH_HEADER_LEN, parse_message_type


def serialize_batch(bets: list[Bet], agency_id: int) -> bytes:
    bet_data = bytearray((TYPE_BETS, agency_id))
    for bet in bets:
        bet_data.extend(serialize_bet(bet))
    bet_data = len(bet_data).to_bytes(2, byteorder="big") + bet_data
    return bytes(bet_data)


def deserialize_batch(batch_data: bytes) -> list[Bet]:
    if len(batch_data) < BATCH_HEADER_LEN:
        raise ValueError("packet too short")

    message_type = parse_message_type(batch_data)
    if message_type != TYPE_BETS:
        raise ValueError(f"invalid bet type: {batch_data[0]}")

    agency_id = batch_data[1]
    bets = []
    offset = BATCH_HEADER_LEN

    while offset < len(batch_data):
        bet, size = deserialize_bet(batch_data[offset:], agency_id)
        bets.append(bet)
        offset += size
    return bets
