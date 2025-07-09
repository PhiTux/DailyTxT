import secrets
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
  data_path: str = "/data"
  development: bool = False
  secret_token: str = secrets.token_urlsafe(32) 
  logout_after_days: int = 30
  allowed_hosts: list[str] = ["http://localhost:5173","http://127.0.0.1:5173"]
  indent: int | None = None

settings = Settings()