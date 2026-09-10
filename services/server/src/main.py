import os
import sys

import logger
import server

SERVER_HOST = os.environ["SERVER_HOST"]
SERVER_PORT = int(os.environ["SERVER_PORT"])
LOTTERY_STORAGE_PATH = os.getenv("LOTTERY_STORAGE_PATH", "/tmp/lottery_bets.csv")


def main():
    logger.init()
    s = server.Server(SERVER_HOST, SERVER_PORT, LOTTERY_STORAGE_PATH)
    try:
        s.run()
    except Exception as e:
        logger.error("server-run", logger.LogResult.fail, "err", e)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
