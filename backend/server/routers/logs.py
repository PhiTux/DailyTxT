import datetime
import logging
from fastapi import APIRouter, Cookie
from pydantic import BaseModel
from fastapi import Depends
from . import users
from ..utils import fileHandling
from ..utils import security


logger = logging.getLogger("dailytxtLogger")

router = APIRouter()


class Log(BaseModel):
    date: str
    text: str
    date_written: str

@router.post("/saveLog")
async def saveLog(log: Log, cookie = Depends(users.isLoggedIn)):
    print(datetime.datetime.fromisoformat(log.date))
    year = datetime.datetime.fromisoformat(log.date).year
    month = datetime.datetime.fromisoformat(log.date).month
    day = datetime.datetime.fromisoformat(log.date).day

    content:dict = fileHandling.getDay(cookie["user_id"], year, month)
    
    # move old log to history
    if "days" in content.keys():
        for dayLog in content["days"]:
            if dayLog["day"] == day:
                historyVersion = 0
                if "history" not in dayLog.keys():
                    dayLog["history"] = []
                else:
                    for historyLog in dayLog["history"]:
                        if historyLog["version"] > historyVersion:
                            historyVersion = historyLog["version"]
                historyVersion += 1
                dayLog["history"].append({"version": historyVersion, "text": dayLog["text"], "date_written": dayLog["date_written"]})
                break
        
    # save new log
    encrypted_text = security.encrypt_text(log.text, cookie["derived_key"])
    encrypted_date_written = security.encrypt_text(log.date_written, cookie["derived_key"])
    
    if "days" not in content.keys():
        content["days"] = []
        content["days"].append({"day": day, "text": encrypted_text, "date_written": encrypted_date_written})
    else:
        found = False
        for dayLog in content["days"]:
            if dayLog["day"] == day:
                dayLog["text"] = encrypted_text
                dayLog["date_written"] = encrypted_date_written
                found = True
                break
        if not found:
            content["days"].append({"day": day, "text": encrypted_text, "date_written": encrypted_date_written})

    if not fileHandling.writeDay(cookie["user_id"], year, month, content):
        logger.error(f"Failed to save log for {cookie['user_id']} on {year}-{month:02d}-{day:02d}")
        return {"success": False}

    return {"success": True}


@router.get("/getLog")
async def getLog(date: str, cookie = Depends(users.isLoggedIn)):
    
    year = datetime.datetime.fromisoformat(date).year
    month = datetime.datetime.fromisoformat(date).month
    day = datetime.datetime.fromisoformat(date).day

    content:dict = fileHandling.getDay(cookie["user_id"], year, month)
    
    if "days" not in content.keys():
        return {"text": "", "date_written": ""}
    
    for dayLog in content["days"]:
        if dayLog["day"] == day:
            text = security.decrypt_text(dayLog["text"], cookie["derived_key"])
            date_written = security.decrypt_text(dayLog["date_written"], cookie["derived_key"])
            return {"text": text, "date_written": date_written}

    return {"text": "", "date_written": ""}