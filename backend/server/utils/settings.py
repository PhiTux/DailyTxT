import secrets
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
  data_path: str = "/data"
  development: bool = False
  secret_token: str = secrets.token_urlsafe(32) 
  logout_after_days: int = 30

settings = Settings()