from pydantic_settings import BaseSettings

class Settings(BaseSettings):
  data_path: str = "/data"

settings = Settings()