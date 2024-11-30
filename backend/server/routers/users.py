from typing import Annotated
from fastapi import APIRouter
from fastapi import Form
from pydantic import BaseModel

router = APIRouter()

class Login(BaseModel):
    username: str
    password: str

@router.post("/users/login")
async def login(login: Login):
    print(login.username, login.password)
    return {"message": "Login"}


@router.post("/users/register")
async def register(username: Annotated[str, Form()], password: Annotated[str, Form()]):
    return {"message": "Register"}