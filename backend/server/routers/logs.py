import datetime
import logging
import re
from fastapi import APIRouter, Depends, Form, UploadFile, File, HTTPException
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
from . import users
from ..utils import fileHandling
from ..utils import security
import html
from typing import Annotated


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

    content:dict = fileHandling.getMonth(cookie["user_id"], year, month)
    
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

    if not fileHandling.writeMonth(cookie["user_id"], year, month, content):
        logger.error(f"Failed to save log for {cookie['user_id']} on {year}-{month:02d}-{day:02d}")
        return {"success": False}

    return {"success": True}


@router.get("/getLog")
async def getLog(date: str, cookie = Depends(users.isLoggedIn)):
    
    year = datetime.datetime.fromisoformat(date).year
    month = datetime.datetime.fromisoformat(date).month
    day = datetime.datetime.fromisoformat(date).day

    content:dict = fileHandling.getMonth(cookie["user_id"], year, month)
    
    dummy = {"text": "", "date_written": "", "files": [], "tags": []}

    if "days" not in content.keys():
        return dummy
    
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
            return {"text": text, "date_written": date_written, "files": dayLog.get("files", []), "tags": dayLog.get("tags", [])}

    return dummy

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


def get_begin(text):
    # get first 5 words
    words = text.split()
    if len(words) < 5:
        return text
    return " ".join(words[:5])

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


@router.get("/searchString")
async def searchString(searchString: str, cookie = Depends(users.isLoggedIn)):
    results = []
    
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    
    # search in all years and months (dirs)
    for year in fileHandling.get_years(cookie["user_id"]):
        for month in fileHandling.get_months(cookie["user_id"], year):
            content:dict = fileHandling.getMonth(cookie["user_id"], year, int(month))
            if "days" not in content.keys():
                continue
            for dayLog in content["days"]:
                if "text" not in dayLog.keys():
                    if "files" in dayLog.keys():
                        for file in dayLog["files"]:
                            filename = security.decrypt_text(file["enc_filename"], enc_key)
                            if searchString.lower() in filename.lower():
                                context = "📎 " + filename
                                results.append({"year": year, "month": month, "day": dayLog["day"], "text": context})
                                break
                    continue
                
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
                    
                    elif "files" in dayLog.keys():
                        for file in dayLog["files"]:
                            filename = security.decrypt_text(file["enc_filename"], enc_key)
                            if searchString.lower() in filename.lower():
                                context = "📎 " + filename
                                results.append({"year": year, "month": month, "day": dayLog["day"], "text": context})
                                break
                        
        
    # sort by year and month and day
    results.sort(key=lambda x: (int(x["year"]), int(x["month"]), int(x["day"])), reverse=False)
    return results

@router.get("/searchTag")
async def searchTag(tag_id: int, cookie = Depends(users.isLoggedIn)):
    results = []
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])

    # search in all years and months (dirs)
    for year in fileHandling.get_years(cookie["user_id"]):
        for month in fileHandling.get_months(cookie["user_id"], year):
            content:dict = fileHandling.getMonth(cookie["user_id"], year, int(month))
            if "days" not in content.keys():
                continue
            for dayLog in content["days"]:
                if "tags" not in dayLog.keys():
                    continue
                if tag_id in dayLog["tags"]:
                    context = ''
                    if "text" in dayLog.keys():
                        text = security.decrypt_text(dayLog["text"], enc_key)
                        context = get_begin(text)
                    results.append({"year": year, "month": month, "day": dayLog["day"], "text": context})
    
    # sort by year and month and day
    results.sort(key=lambda x: (int(x["year"]), int(x["month"]), int(x["day"])), reverse=False)
    return results

@router.get("/getMarkedDays")
async def getMarkedDays(month: str, year: str, cookie = Depends(users.isLoggedIn)):
    days_with_logs = []
    days_with_files = []

    content:dict = fileHandling.getMonth(cookie["user_id"], year, int(month))
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
    content:dict = fileHandling.getMonth(cookie["user_id"], year, month)
    if "days" not in content.keys():
        return []
    
    days = []
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    for dayLog in content["days"]:
        day = {"day": dayLog["day"]}
        if "text" in dayLog.keys():
            day["text"] = security.decrypt_text(dayLog["text"], enc_key)
            day["date_written"] = security.decrypt_text(dayLog["date_written"], enc_key)
        if "tags" in dayLog.keys():
            day["tags"] = dayLog["tags"]
        if "files" in dayLog.keys():
            day["files"] = []
            for file in dayLog["files"]:
                file["filename"] = security.decrypt_text(file["enc_filename"], enc_key)
                day["files"].append(file)

        # if one of the keys is in day: 
        if "text" in day or "files" in day or "tags" in day:
            days.append(day)
    
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
    content:dict = fileHandling.getMonth(cookie["user_id"], year, month)

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
    
    if not fileHandling.writeMonth(cookie["user_id"], year, month, content):
        fileHandling.removeFile(cookie["user_id"], uuid)
        return {"success": False}

    return {"success": True}


@router.get("/deleteFile")
async def deleteFile(uuid: str, day: int, month: int, year: int, cookie = Depends(users.isLoggedIn)):
    content:dict = fileHandling.getMonth(cookie["user_id"], year, month)
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
                if not fileHandling.writeMonth(cookie["user_id"], year, month, content):
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

@router.get("/getTags")
async def getTags(cookie = Depends(users.isLoggedIn)):
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    
    content:dict = fileHandling.getTags(cookie["user_id"])

    if not 'tags' in content:
        return []
    
    else:
        for tag in content['tags']:
            tag['icon'] = security.decrypt_text(tag['icon'], enc_key)
            tag['name'] = security.decrypt_text(tag['name'], enc_key)
            tag['color'] = security.decrypt_text(tag['color'], enc_key)
        return content['tags']


class NewTag(BaseModel):
    icon: str
    name: str
    color: str

@router.post("/saveNewTag")
async def saveNewTag(tag: NewTag, cookie = Depends(users.isLoggedIn)):
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    
    content:dict = fileHandling.getTags(cookie["user_id"])
    
    if not 'tags' in content:
        content['tags'] = []
        content['next_id'] = 1
    
    enc_icon = security.encrypt_text(tag.icon, enc_key)
    enc_name = security.encrypt_text(tag.name, enc_key)
    enc_color = security.encrypt_text(tag.color, enc_key)

    new_tag = {"id": content['next_id'], "icon": enc_icon, "name": enc_name, "color": enc_color}
    content['next_id'] += 1
    content['tags'].append(new_tag)

    if not fileHandling.writeTags(cookie["user_id"], content):
        return {"success": False}
    else:
        return {"success": True}


class EditTag(BaseModel):
    id: int
    icon: str
    name: str
    color: str

@router.post("/editTag")
async def editTag(editTag: EditTag, cookie = Depends(users.isLoggedIn)):
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    
    content:dict = fileHandling.getTags(cookie["user_id"])
    
    if not 'tags' in content:
        raise HTTPException(status_code=500, detail="Tag not found - json error")
    
    for tag in content['tags']:
        if tag['id'] == editTag.id:
            tag['icon'] = security.encrypt_text(editTag.icon, enc_key)
            tag['name'] = security.encrypt_text(editTag.name, enc_key)
            tag['color'] = security.encrypt_text(editTag.color, enc_key)
            if not fileHandling.writeTags(cookie["user_id"], content):
                raise HTTPException(status_code=500, detail="Failed to write tag - error writing tags")
            else:
                return {"success": True}
    
    raise HTTPException(status_code=500, detail="Tag not found - not in tags")

@router.get("/deleteTag")
async def deleteTag(id: int, cookie = Depends(users.isLoggedIn)):
    # remove from every log if present
    for year in fileHandling.get_years(cookie["user_id"]):
        for month in fileHandling.get_months(cookie["user_id"], year):
            content:dict = fileHandling.getMonth(cookie["user_id"], year, int(month))
            if "days" not in content.keys():
                continue
            for dayLog in content["days"]:
                if "tags" in dayLog.keys() and id in dayLog["tags"]:
                    dayLog["tags"].remove(id)
            if not fileHandling.writeMonth(cookie["user_id"], year, int(month), content):
                raise HTTPException(status_code=500, detail="Failed to delete tag - error writing log")
    
    # remove from tags
    content:dict = fileHandling.getTags(cookie["user_id"])
    if not 'tags' in content:
        raise HTTPException(status_code=500, detail="Tag not found - json error")
    
    for tag in content['tags']:
        if tag['id'] == id:
            content['tags'].remove(tag)
            if not fileHandling.writeTags(cookie["user_id"], content):
                raise HTTPException(status_code=500, detail="Failed to delete tag - error writing tags")
            else:
                return {"success": True}
    
    raise HTTPException(status_code=500, detail="Tag not found - not in tags")

class AddTagToLog(BaseModel):
    day: int
    month: int
    year: int
    tag_id: int

@router.post("/addTagToLog")
async def addTagToLog(data: AddTagToLog, cookie = Depends(users.isLoggedIn)):
    content:dict = fileHandling.getMonth(cookie["user_id"], data.year, data.month)
    if "days" not in content.keys():
        content["days"] = []
    
    dayFound = False
    for dayLog in content["days"]:
        if dayLog["day"] != data.day:
            continue
        dayFound = True
        if not "tags" in dayLog.keys():
            dayLog["tags"] = []
        if data.tag_id in dayLog["tags"]:
            # fail silently
            return {"success": True}
        dayLog["tags"].append(data.tag_id)
        break
    
    if not dayFound:
        content["days"].append({"day": data.day, "tags": [data.tag_id]})
    
    if not fileHandling.writeMonth(cookie["user_id"], data.year, data.month, content):
        raise HTTPException(status_code=500, detail="Failed to write tag - error writing log")
    return {"success": True}

@router.post("/removeTagFromLog")
async def removeTagFromLog(data: AddTagToLog, cookie = Depends(users.isLoggedIn)):
    content:dict = fileHandling.getMonth(cookie["user_id"], data.year, data.month)
    if "days" not in content.keys():
        raise HTTPException(status_code=500, detail="Day not found - json error")
    
    for dayLog in content["days"]:
        if dayLog["day"] != data.day:
            continue
        if not "tags" in dayLog.keys():
            raise HTTPException(status_code=500, detail="Failed to remove tag - not found in log")
        if not data.tag_id in dayLog["tags"]:
            raise HTTPException(status_code=500, detail="Failed to remove tag - not found in log")
        dayLog["tags"].remove(data.tag_id)
        if not fileHandling.writeMonth(cookie["user_id"], data.year, data.month, content):
            raise HTTPException(status_code=500, detail="Failed to remove tag - error writing log")
        return {"success": True}
    
@router.get("/getTemplates")
async def getTemplates(cookie = Depends(users.isLoggedIn)):
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    
    content:dict = fileHandling.getTemplates(cookie["user_id"])

    if not 'templates' in content:
        return []
    
    else:
        for template in content['templates']:
            template['name'] = security.decrypt_text(template['name'], enc_key)
            template['text'] = security.decrypt_text(template['text'], enc_key)
        return content['templates']

class Templates(BaseModel):
    templates: list[dict]

@router.post("/saveTemplates")
async def saveTemplates(templates: Templates, cookie = Depends(users.isLoggedIn)):
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    
    content = {'templates': []}
    
    for template in templates.templates:
        enc_name = security.encrypt_text(template["name"], enc_key)
        enc_text = security.encrypt_text(template["text"], enc_key)

        new_template = {"name": enc_name, "text": enc_text}
        content['templates'].append(new_template)

    if not fileHandling.writeTemplates(cookie["user_id"], content):
        raise HTTPException(status_code=500, detail="Failed to write templates - error writing templates")
    else:
        return {"success": True}