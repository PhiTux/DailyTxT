import asyncio
import datetime
import json
import secrets
from fastapi import APIRouter, Cookie, Depends, HTTPException, Response
from pydantic import BaseModel
from ..utils import fileHandling
from ..utils import security
from ..utils.settings import settings
import logging
import base64
import jwt

logger = logging.getLogger("dailytxtLogger")

router = APIRouter()

class Login(BaseModel):
    username: str
    password: str

@router.post("/login")
async def login(login: Login, response: Response):

    # check if user exists
    content:dict = fileHandling.getUsers()
    if len(content) == 0 or "users" not in content.keys() or len(content["users"]) == 0 or not any(user["username"] == login.username for user in content["users"]):
        logger.error(f"Login failed. User '{login.username}' not found")
        raise HTTPException(status_code=404, detail="User/Password combination not found")
    
    # get user data
    user = next(user for user in content["users"] if user["username"] == login.username)
    if not security.verify_password(login.password, user["password"]):
        logger.error(f"Login failed. Password for user '{login.username}' is incorrect")
        raise HTTPException(status_code=404, detail="User/Password combination not found")
    
    # get intermediate key
    derived_key = base64.b64encode(security.derive_key_from_password(login.password, user["salt"])).decode()
    
    # build jwt
    token = create_jwt(user["user_id"], user["username"], derived_key)
    response.set_cookie(key="token", value=token, httponly=True, samesite="lax")
    return {"username": user["username"]}

def create_jwt(user_id, username, derived_key):
    return jwt.encode({"exp": datetime.datetime.now(tz=datetime.timezone.utc) + datetime.timedelta(days=settings.logout_after_days), "user_id": user_id, "name": username, "derived_key": derived_key}, settings.secret_token, algorithm="HS256")

def decode_jwt(token):
    return jwt.decode(token, settings.secret_token, algorithms="HS256")

def isLoggedIn(token: str = Cookie()) -> int:
    try:
        decoded = decode_jwt(token)
        return decoded
    except jwt.ExpiredSignatureError:
        raise HTTPException(status_code=440, detail="Token expired")
    except:
        raise HTTPException(status_code=401, detail="Not logged in")


@router.get("/logout")
def logout(response: Response):
    response.delete_cookie("token", httponly=True)
    return {"success": True}


class Register(BaseModel):
    username: str
    password: str

@router.post("/register")
async def register(register: Register):
    content:dict = fileHandling.getUsers()

    # check if username already exists
    if len(content) > 0:
        if ("users" not in content.keys()):
            logger.error("users.json is not in the correct format. Key 'users' is missing.")
            raise HTTPException(status_code=500, detail="users.json is not in the correct format")
        for user in content["users"]:
            if user["username"] == register.username:
                logger.error(f"Registration failed. Username '{register.username}' already exists")
                raise HTTPException(status_code=400, detail="Username already exists")

    # create new user-data
    username = register.username
    password = security.hash_password(register.password)
    salt = secrets.token_urlsafe(16)
    enc_enc_key = security.create_new_enc_enc_key(register.password, salt).decode()
    

    if len(content) == 0:
        content = {"id_counter": 1, "users": [
            {
                "user_id": 1,
                "dailytxt_version": 2,
                "username": username,
                "password": password,
                "salt": salt, 
                "enc_enc_key": enc_enc_key
            }
        ]}


    else:
        content["id_counter"] += 1
        content["users"].append(
            {
                "user_id": content["id_counter"],
                "dailytxt_version": 2,
                "username": username,
                "password": password,
                "salt": salt, 
                "enc_enc_key": enc_enc_key
            }
        )

    try:
        fileHandling.writeUsers(content)
    except Exception as e:
        raise HTTPException(status_code=500, detail="Internal Server Error when trying to write users.json") from e
    else:
        return {"success": True}

def get_default_user_settings():
    return {
        "autoloadImagesByDefault": False,
        "setAutoloadImagesPerDevice": True,
        "useOnThisDay": True,
        "onThisDayYears": [1,5,10],
        "useBrowserTimezone": True,
        "timezone": "UTC"
    }

@router.get("/getUserSettings")
async def get_user_settings(cookie = Depends(isLoggedIn)):
    user_id = cookie["user_id"]
    content_enc = fileHandling.getUserSettings(user_id)   

    if len(content_enc) > 0:
        # decrypt settings
        enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
        content = json.loads(security.decrypt_text(content_enc, enc_key))
    else:
        content = {}

    default = get_default_user_settings()

    for key in default.keys():
        if key not in content.keys():
            content[key] = default[key]
    
    return content


@router.post("/saveUserSettings")
async def save_user_settings(settings: dict, cookie = Depends(isLoggedIn)):
    user_id = cookie["user_id"]
    content = fileHandling.getUserSettings(user_id)   
    
    enc_key = security.get_enc_key(cookie["user_id"], cookie["derived_key"])
    if len(content) > 0:
        # decrypt settings
        content = json.loads(security.decrypt_text(content, enc_key))
    else:
        content = {}

    # if content is empty dict
    if content is None or len(content) == 0:
        content = get_default_user_settings()
    
    # update settings
    for key in settings.keys():
        content[key] = settings[key]

    # encrypt settings
    content_enc = security.encrypt_text(json.dumps(content), enc_key)

    try:
        fileHandling.writeUserSettings(user_id, content_enc)
    except Exception as e:
        raise HTTPException(status_code=500, detail="Internal Server Error when trying to write users.json") from e
    else:
        return {"success": True}
