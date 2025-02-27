import base64
import datetime
import io
import logging
import re
from fastapi import APIRouter, Cookie, Depends, Form, UploadFile, File, HTTPException
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
from . import users
from ..utils import fileHandling
from ..utils import security
import html
from typing import Annotated
import time


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
            if dayLog["day"] == day and "text" in dayLog.keys():
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
    
    ''' IMPORTANT: 
    Escaping html characters here is NOT possible, since it would break the appearance
    of html-code in the EDITOR. Code inside a markdown code-quote (`...`) will be auto-escaped.
    Not a perfect solution, but actually any user can only load its own logs...
    '''
    encrypted_text = security.encrypt_text(log.text, enc_key)
    encrypted_date_written = security.encrypt_text(html.escape(log.date_written), enc_key)
    
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
            text = ""
            date_written = ""
            if "text" in dayLog.keys():
                text = security.decrypt_text(dayLog["text"], enc_key)
                date_written = security.decrypt_text(dayLog["date_written"], enc_key)
            if "files" in dayLog.keys():
                for file in dayLog["files"]:
                    file["filename"] = security.decrypt_text(file["enc_filename"], enc_key)
                    file["type"] = security.decrypt_text(file["enc_type"], enc_key)
            return {"text": text, "date_written": date_written, "files": dayLog.get("files", [])}

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
    results.sort(key=lambda x: (int(x["year"]), int(x["month"]), int(x["day"])), reverse=False)
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

'''
Data ist sent as FormData, not as JSON
'''
@router.post("/uploadFile")
async def uploadFile(day: Annotated[int, Form()], month: Annotated[int, Form()], year: Annotated[int, Form()], uuid: Annotated[str, Form()], file: Annotated[UploadFile, File()], cookie = Depends(users.isLoggedIn)):
    
    # encrypt file
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    encrypted_file = security.encrypt_file(file.file.read(), enc_key)
    if not fileHandling.writeFile(encrypted_file, cookie["user_id"], uuid):
        return {"success": False}
    
    # save file in log
    content:dict = fileHandling.getDay(cookie["user_id"], year, month)

    enc_filename = security.encrypt_text(file.filename, enc_key)
    new_file = {"enc_filename": enc_filename, "uuid_filename": uuid, "size": file.size}

    if "days" not in content.keys():
        content["days"] = []
        content["days"].append({"day": day, "files": [new_file]})

    else:
        found = False
        for dayLog in content["days"]:
            if dayLog["day"] == day:
                if "files" not in dayLog.keys():
                    dayLog["files"] = []
                dayLog["files"].append(new_file)
                found = True
                break
        if not found:
            content["days"].append({"day": day, "files": [new_file]})
    
    if not fileHandling.writeDay(cookie["user_id"], year, month, content):
        fileHandling.removeFile(cookie["user_id"], uuid)
        return {"success": False}

    return {"success": True}

"""
@router.get("/getFiles")
async def getFiles(day: int, month: int, year: int, cookie = Depends(users.isLoggedIn)):
    content:dict = fileHandling.getDay(cookie["user_id"], year, month)
    if "days" not in content.keys():
        return []
    
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    for dayLog in content["days"]:
        if "day" in dayLog.keys() and dayLog["day"] == day:
            if "files" in dayLog.keys():
                for file in dayLog["files"]:
                    file["filename"] = security.decrypt_text(file["enc_filename"], enc_key)
                    file["type"] = security.decrypt_text(file["enc_type"], enc_key)
    
                return dayLog["files"]

    return []
"""

@router.get("/deleteFile")
async def deleteFile(uuid: str, day: int, month: int, year: int, cookie = Depends(users.isLoggedIn)):
    content:dict = fileHandling.getDay(cookie["user_id"], year, month)
    if "days" not in content.keys():
        raise HTTPException(status_code=500, detail="Day not found - json error")
    
    for dayLog in content["days"]:
        if dayLog["day"] != day:
            continue
        if not "files" in dayLog.keys():
            break
        for file in dayLog["files"]:
            if file["uuid_filename"] == uuid:
                if not fileHandling.removeFile(cookie["user_id"], uuid):
                    raise HTTPException(status_code=500, detail="Failed to delete file")
                dayLog["files"].remove(file)
                if not fileHandling.writeDay(cookie["user_id"], year, month, content):
                    raise HTTPException(status_code=500, detail="Failed to write changes of deleted file!")
                return {"success": True}

    raise HTTPException(status_code=500, detail="Failed to delete file - not found in log")

@router.get("/downloadFile")
async def downloadFile(uuid: str, cookie = Depends(users.isLoggedIn)):
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    file = fileHandling.readFile(cookie["user_id"], uuid)
    if file is None:
        raise HTTPException(status_code=500, detail="Failed to read file")
    content = security.decrypt_file(file, enc_key)
    return StreamingResponse(iter([content]))