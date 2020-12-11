import os


class BaseConfig(object):
    DEBUG = False

    DATA_PATH = 'data/'
    USERS_FILE = 'users.json'

    # used for encryption and session management
    SECRET_KEY = os.urandom(24)
    if 'SECRET_KEY' in os.environ:
        SECRET_KEY = os.environ.get('SECRET_KEY')
