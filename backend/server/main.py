from fastapi import FastAPI
from .routers import users, logs
from fastapi.middleware.cors import CORSMiddleware
import logging
from sys import stdout

logger = logging.getLogger("dailytxtLogger")
consoleHandler = logging.StreamHandler(stdout)
consoleHandler.setFormatter(logging.Formatter("%(asctime)s - %(levelname)s - %(message)s"))
logger.addHandler(consoleHandler)
logger.setLevel(logging.DEBUG)

app = FastAPI()

origins = [
    "http://localhost:5173",
    "localhost:5173",
]

app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(users.router, prefix="/users")
app.include_router(logs.router, prefix="/logs")

@app.get("/")
async def root():
    return {"message": "Hello World"}

logger.info("Server started")