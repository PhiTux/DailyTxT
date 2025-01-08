import datetime
import logging
import re
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
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    encrypted_text = security.encrypt_text(log.text, enc_key)
    encrypted_date_written = security.encrypt_text(log.date_written, enc_key)
    
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
            enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
            text = security.decrypt_text(dayLog["text"], enc_key)
            date_written = security.decrypt_text(dayLog["date_written"], enc_key)
            return {"text": text, "date_written": date_written}

    return {"text": "", "date_written": ""}

def get_start_index(text, index):
    # find a whitespace two places before the index
    
    if index == 0:
        return 0
    
    for i in range(3):
        startIndex = text.rfind(" ", 0, index-1)
        index = startIndex
        if startIndex == -1:
            return 0

    return startIndex + 1

def get_end_index(text, index):
    # find a whitespace two places after the index
    
    if index == len(text) - 1:
        return len(text)
    
    for i in range(3):
        endIndex = text.find(" ", index+1)
        index = endIndex
        if endIndex == -1:
            return len(text)

    return endIndex


def get_context(text: str, searchString: str, exact: bool):
    # replace whitespace with non-breaking space
    text = re.sub(r'\s+', " ", text)

    if exact:
        pos = text.find(searchString)
    else:
        pos = text.lower().find(searchString.lower())
    if pos == -1:
        return "<em>Dailytxt: Error formatting...</em>"

    start = get_start_index(text, pos)
    end = get_end_index(text, pos + len(searchString) - 1)
    return text[start:pos] + "<b>" + text[pos:pos+len(searchString)] + "</b>" + text[pos+len(searchString):end]


@router.get("/search")
async def search(searchString: str, cookie = Depends(users.isLoggedIn)):
    results = []
    
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    
    # search in all years and months (dirs)
    for year in fileHandling.get_years(cookie["user_id"]):
        for month in fileHandling.get_months(cookie["user_id"], year):
            content:dict = fileHandling.getDay(cookie["user_id"], year, int(month))
            if "days" not in content.keys():
                continue
            for dayLog in content["days"]:
                text = security.decrypt_text(dayLog["text"], enc_key)
                
                # "..." -> exact
                # ... | ... -> or
                # ...  ... -> and

                if searchString.startswith('"') and searchString.endswith('"'):
                    if searchString[1:-1] in text:
                        context = get_context(text, searchString[1:-1], True)
                        results.append({"year": year, "month": month, "day": dayLog["day"], "text": context})
                        
                
                elif "|" in searchString:
                    words = searchString.split("|")
                    for word in words:
                        if word.strip().lower() in text.lower():
                            context = get_context(text, word.strip(), False)
                            results.append({"year": year, "month": month, "day": dayLog["day"], "text": context})
                            break
                            

                elif " " in searchString:
                    if all([word.strip().lower() in text.lower() for word in searchString.split()]):
                        context = get_context(text, searchString.split()[0].strip(), False)
                        results.append({"year": year, "month": month, "day": dayLog["day"], "text": context})
                        
                
                else:
                    if searchString.lower() in text.lower():
                        context = get_context(text, searchString, False)
                        results.append({"year": year, "month": month, "day": dayLog["day"], "text": context})
                        
        
    # sort by year and month and day
    results.sort(key=lambda x: (int(x["year"]), int(x["month"]), int(x["day"])), reverse=True)
    print(results)
    return results

@router.get("/getMarkedDays")
async def getMarkedDays(month: str, year: str, cookie = Depends(users.isLoggedIn)):
    days_with_logs = []
    days_with_files = []

    content:dict = fileHandling.getDay(cookie["user_id"], year, int(month))
    if "days" not in content.keys():
        return {"days_with_logs": [], "days_with_files": []}

    for dayLog in content["days"]:
        if "text" in dayLog.keys():
            days_with_logs.append(dayLog["day"])
        if "files" in dayLog.keys() and len(dayLog["files"]) > 0:
            days_with_files.append(dayLog["day"])
    
    return {"days_with_logs": days_with_logs, "days_with_files": days_with_files}


@router.get("/loadMonthForReading")
async def loadMonthForReading(month: int, year: int, cookie = Depends(users.isLoggedIn)):
    content:dict = fileHandling.getDay(cookie["user_id"], year, month)
    if "days" not in content.keys():
        return []
    
    days = []
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    for dayLog in content["days"]:
        if "text" in dayLog.keys():
            days.append({"day": dayLog["day"], 
                         "text": security.decrypt_text(dayLog["text"], enc_key), 
                         "date_written": security.decrypt_text(dayLog["date_written"], enc_key)})
    
    days.sort(key=lambda x: x["day"])

    return days