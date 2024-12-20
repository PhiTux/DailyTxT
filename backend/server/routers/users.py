import asyncio
import datetime
import json
import secrets
from fastapi import APIRouter, HTTPException, Response
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

@router.post("/users/login")
async def login(login: Login, respose: Response):

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
    jwt = create_jwt(user["user_id"], user["username"], derived_key)
    respose.set_cookie(key="jwt", value=jwt, httponly=True)
    return {"username": user["username"]}

def create_jwt(user_id, username, derived_key):
    return jwt.encode({"iat": datetime.datetime.now() + datetime.timedelta(days=settings.logout_after_days), "user_id": user_id, "name": username, "derived_key": derived_key}, settings.secret_token, algorithm="HS256")


@router.get("/users/logout")
def logout(response: Response):
    response.delete_cookie("jwt")
    return {"success": True}


class Register(BaseModel):
    username: str
    password: str

@router.post("/users/register")
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


"""
{
      "user_id": 1,
      "dailytxt-version": 2,
      "username": "Marco",
      "password": "...",
      "salt": "...",
      "enc_enc_key": "...",
      "LOGIN_ON_EACH_LOAD": false,
      "backup_codes": [
        {
          "password": "...",
          "salt": "...",
          "enc_orig_password": "..."
        }
      ]
    }"""