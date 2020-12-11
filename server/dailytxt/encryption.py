from cryptography.fernet import Fernet
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend
from os import urandom
from flask import current_app
from .file_handling import *
import base64


def create_new_key():
    return Fernet.generate_key()


def create_new_key_from_password(password_provided):
    password = password_provided.encode()
    salt = urandom(16)
    kdf = PBKDF2HMAC(
        algorithm=hashes.SHA256(),
        length=32,
        salt=salt,
        iterations=100000,
        backend=default_backend()
    )

    key = base64.urlsafe_b64encode(kdf.derive(password))
    return {'salt': base64.urlsafe_b64encode(salt), 'key': key}


def encrypt_by_key(message, key):
    f = Fernet(key)
    return f.encrypt(message)


def decrypt_by_key(message, key):
    f = Fernet(key)
    return f.decrypt(message)


def get_enc_key(user_id, key):
    file_content = read_users()

    for user in file_content['users']:
        if user['user_id'] == user_id:
            return decrypt_by_key(user['enc_enc_key'].encode(), key)

    return ''


def encrypt_file_by_userid(message, user_id, key):
    enc_key = get_enc_key(user_id, key)

    if enc_key == '':
        return {'success': False, 'text': ''}

    return {'success': True, 'text': encrypt_by_key(message, enc_key).decode()}


def decrypt_file_by_userid(message, user_id, key):
    enc_key = get_enc_key(user_id, key)

    if enc_key == '':
        return {'success': False, 'text': ''}

    return {'success': True, 'text': decrypt_by_key(message.encode(), enc_key)}


def encrypt_by_userid(message, user_id, key):
    enc_key = get_enc_key(user_id, key)

    if enc_key == '':
        return {'success': False, 'text': ''}

    return {'success': True, 'text': encrypt_by_key(message.encode(), enc_key).decode()}


def decrypt_by_userid(message, user_id, key):
    enc_key = get_enc_key(user_id, key)

    if enc_key == '':
        return {'success': False, 'text': ''}

    return {'success': True, 'text': decrypt_by_key(message.encode(), enc_key).decode()}


def get_password_key(password, salt):
    kdf = PBKDF2HMAC(
        algorithm=hashes.SHA256(),
        length=32,
        salt=base64.urlsafe_b64decode(salt),
        iterations=100000,
        backend=default_backend()
    )

    return base64.urlsafe_b64encode(kdf.derive(password))
