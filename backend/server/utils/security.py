from passlib.hash import argon2
from argon2.low_level import hash_secret_raw, Type
from cryptography.fernet import Fernet
import base64

def hash_password(password: str) -> str:
    return argon2.hash(password)

def verify_password(password: str, hash: str) -> bool:
    return argon2.verify(password, hash)

def derive_key_from_password(password: str, salt: bytes) -> bytes:
    return hash_secret_raw(secret=password.encode(), salt=salt, time_cost=2, memory_cost=2**15, parallelism=1, hash_len=32, type=Type.ID)

def create_new_enc_enc_key(password: str, salt: bytes) -> bytes:
    derived_key = derive_key_from_password(password, salt) # password derived key only to encrypt the actual encryption key
    key = Fernet.generate_key() # actual encryption key
    f = Fernet(base64.urlsafe_b64encode(derived_key))
    return f.encrypt(key)

    