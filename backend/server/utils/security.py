from fastapi import HTTPException
from passlib.hash import argon2
from argon2.low_level import hash_secret_raw, Type
from cryptography.fernet import Fernet
import base64
from . import fileHandling

def hash_password(password: str) -> str:
    return argon2.hash(password)

def verify_password(password: str, hash: str) -> bool:
    return argon2.verify(password, hash)

def derive_key_from_password(password: str, salt: str) -> bytes:
    return hash_secret_raw(secret=password.encode(), salt=salt.encode(), time_cost=2, memory_cost=2**15, parallelism=1, hash_len=32, type=Type.ID)

def create_new_enc_enc_key(password: str, salt: str) -> bytes:
    derived_key = derive_key_from_password(password, salt) # password derived key only to encrypt the actual encryption key
    key = Fernet.generate_key() # actual encryption key
    f = Fernet(base64.urlsafe_b64encode(derived_key))
    return f.encrypt(key)

def get_enc_key(user_id: int, derived_key: str) -> bytes:
    content = fileHandling.getUsers()
    
    if not "users" in content.keys():
        raise HTTPException(status_code=500, detail="users.json is not in the correct format. Key 'users' is missing.")
    
    for user in content["users"]:
        if user["user_id"] == user_id:
            key = user["enc_enc_key"]
    
            f = Fernet(base64.urlsafe_b64encode(base64.b64decode(derived_key)))
            return base64.urlsafe_b64encode(base64.urlsafe_b64decode(f.decrypt(key)))

def encrypt_text(text: str, key: str) -> str:
    f = Fernet(key)
    return f.encrypt(text.encode()).decode()

def decrypt_text(text: str, key: str) -> str:
    f = Fernet(key)
    return f.decrypt(text.encode()).decode()