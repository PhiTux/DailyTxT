import json
import secrets
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from ..utils import fileHandling
from ..utils import security
import logging

logger = logging.getLogger("dailytxtLogger")

router = APIRouter()

class Login(BaseModel):
    username: str
    password: str

@router.post("/users/login")
async def login(login: Login):
    print(login.username, login.password)
    return {"message": "Login"}


class Register(BaseModel):
    username: str
    password: str

@router.post("/users/register")
async def register(register: Register):
    content = fileHandling.getUsers()
    if isinstance(content, Exception):
        raise HTTPException(status_code=500, detail="Internal Server Error when trying to open users.json") 
    

    # check if username already exists
    if len(content) > 0:
        content: dict = json.loads(content)
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
    enc_enc_key = security.create_new_enc_enc_key(register.password, salt.encode()).decode()
    

    if len(content) == 0:
        content = {"id_counter": 1, "users": [
            {
                "user_id": 1,
                "dailytxt-version": 2,
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
                "dailytxt-version": 2,
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