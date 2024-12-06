import json
import os
import logging
from .settings import settings

logger = logging.getLogger("dailytxtLogger")

def getUsers():
    try:
        f = open(os.path.join(settings.data_path, "users.json"), "r")
    except FileNotFoundError:
        logger.info("users.json - File not found")
        return ""
    except Exception as e:
        logger.exception(e)
        return e
    else:
        with f:
            return f.read()

def writeUsers(content):
    # print working directory
    try:
        f = open(os.path.join(settings.data_path, "users.json"), "w")
    except Exception as e:
        logger.exception(e)
        return e
    else:
        with f:
            f.write(json.dumps(content, indent=4))
            return True