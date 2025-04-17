import json
import os
import logging

from fastapi import HTTPException
from .settings import settings

logger = logging.getLogger("dailytxtLogger")

def getUsers():
    try:
        f = open(os.path.join(settings.data_path, "users.json"), "r")
    except FileNotFoundError:
        logger.info("users.json - File not found")
        return {}
    except Exception as e:
        logger.exception(e)
        raise HTTPException(status_code=500, detail="Internal Server Error when trying to open users.json")
    else:
        with f:
            s = f.read()
            if s == "":
                return {}
            return json.loads(s)

def getMonth(user_id, year, month):
    try:
        f = open(os.path.join(settings.data_path, f"{user_id}/{year}/{month:02d}.json"), "r")
    except FileNotFoundError:
        logger.info(f"{user_id}/{year}/{month:02d}.json - File not found")
        return {}
    except Exception as e:
        logger.exception(e)
        raise HTTPException(status_code=500, detail="Internal Server Error when trying to open {year}-{month}.json")
    else:
        with f:
            s = f.read()
            if s == "":
                return {}
            return json.loads(s)

def writeUsers(content):
    # print working directory
    try:
        f = open(os.path.join(settings.data_path, "users.json"), "w")
    except Exception as e:
        logger.exception(e)
        return e
    else:
        with f:
            f.write(json.dumps(content, indent=settings.indent))
            return True
        
def writeMonth(user_id, year, month, content):
    try:
        os.makedirs(os.path.join(settings.data_path, f"{user_id}/{year}"), exist_ok=True)
        f = open(os.path.join(settings.data_path, f"{user_id}/{year}/{month:02d}.json"), "w")
    except Exception as e:
        logger.exception(e)
        return False
    else:
        with f:
            f.write(json.dumps(content, indent=settings.indent))
            return True
        
def get_years(user_id):
    for entry in os.scandir(os.path.join(settings.data_path, str(user_id))):
        if entry.is_dir() and entry.name.isnumeric() and len(entry.name) == 4:
            yield entry.name

def get_months(user_id, year):
    for entry in os.scandir(os.path.join(settings.data_path, str(user_id), year)):
        if entry.is_file() and entry.name.endswith(".json"):
            yield entry.name.split(".")[0]

def writeFile(file, user_id, uuid):
    try:
        os.makedirs(os.path.join(settings.data_path, str(user_id), 'files'), exist_ok=True)
        f = open(os.path.join(settings.data_path, str(user_id), 'files', uuid), "w")
    except Exception as e:
        logger.exception(e)
        return False
    else:
        with f:
            f.write(file)
            return True
        
def readFile(user_id, uuid):
    try:
        f = open(os.path.join(settings.data_path, str(user_id), 'files', uuid), "r")
    except FileNotFoundError:
        logger.info(f"{user_id}/files/{uuid} - File not found")
        raise HTTPException(status_code=404, detail="File not found")
    except Exception as e:
        logger.exception(e)
        raise HTTPException(status_code=500, detail="Internal Server Error when trying to open file {uuid}")
    else:
        with f:
            return f.read()
        
def removeFile(user_id, uuid):
    try:
        os.remove(os.path.join(settings.data_path, str(user_id), 'files', uuid))
    except Exception as e:
        logger.exception(e)
        return False
    else:
        return True
    
def getTags(user_id):
    try:
        f = open(os.path.join(settings.data_path, str(user_id), "tags.json"), "r")
    except FileNotFoundError:
        logger.info(f"{user_id}/tags.json - File not found")
        return {}
    except Exception as e:
        logger.exception(e)
        raise HTTPException(status_code=500, detail="Internal Server Error when trying to open tags.json")
    else:
        with f:
            s = f.read()
            if s == "":
                return {}
            return json.loads(s)
        
def writeTags(user_id, content):
    try:
        f = open(os.path.join(settings.data_path, str(user_id), "tags.json"), "w")
    except Exception as e:
        logger.exception(e)
        return False
    else:
        with f:
            f.write(json.dumps(content, indent=settings.indent))
            return True
        
def getUserSettings(user_id):
    try:
        f = open(os.path.join(settings.data_path, str(user_id), "settings.encrypted"), "r")
    except FileNotFoundError:
        logger.info(f"{user_id}/settings.encrypted - File not found")
        return {}
    except Exception as e:
        logger.exception(e)
        raise HTTPException(status_code=500, detail="Internal Server Error when trying to open settings.json")
    else:
        with f:
            s = f.read()
            return s
        
def writeUserSettings(user_id, content):
    try:
        f = open(os.path.join(settings.data_path, str(user_id), "settings.encrypted"), "w")
    except Exception as e:
        logger.exception(e)
        return False
    else:
        with f:
            f.write(content)
            return True