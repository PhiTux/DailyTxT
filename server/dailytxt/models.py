from werkzeug.security import generate_password_hash, check_password_hash
from .file_handling import *
from .encryption import *


def register_user(username, password):
    data = read_users()

    # check if user already existing
    if isinstance(data, dict):
        for user in data['users']:
            if user['username'].lower() == username.lower():
                return {'success': False, 'message': 'Username already existing'}

    # else:
    # that's the 'real' encryption key
    enc_key = create_new_key()

    # that's the one used to secure the encryption key (based on user password)
    pass_key = create_new_key_from_password(password)
    # 'encrypted encryption key'
    enc_enc_key = encrypt_by_key(enc_key, pass_key['key'])

    new_user = {'username': username,
                'password': generate_password_hash(password, method='sha256'),
                'salt': pass_key['salt'].decode(),
                'enc_enc_key': enc_enc_key.decode()}

    if data == '':
        data = {'id_counter': 0, 'users': [new_user]}
    else:
        data['users'].append(new_user)

    data['id_counter'] = data['id_counter'] + 1
    for user in data['users']:
        if user['username'] == username:
            user['user_id'] = data['id_counter']

    if write_users(data):
        return {'success': True}
    else:
        return {'success': False, 'message': 'Internal Error on Registration'}


def login_user(username, password):
    data = read_users()

    if isinstance(data, dict):
        for user in data['users']:
            if user['username'].lower() == username.lower():
                if check_password_hash(user['password'], password):
                    return {'user_id': user['user_id'], 'password_key': get_password_key(password.encode(), user['salt'].encode())}

    return {'user_id': 0, 'password_key': ''}


def changePassword(user_id, old_password_key, p):

    enc_key = get_enc_key(user_id, old_password_key)
    new_password_key = create_new_key_from_password(p['new_password'])
    enc_enc_key = encrypt_by_key(enc_key, new_password_key['key'])

    file_content = read_users()

    if isinstance(file_content, dict):
        for user in file_content['users']:
            if user['user_id'] == user_id:
                if check_password_hash(user['password'], p['old_password']):
                    user['password'] = generate_password_hash(
                        p['new_password'], method='sha256')
                    user['salt'] = new_password_key['salt'].decode()
                    user['enc_enc_key'] = enc_enc_key.decode()
                else:
                    return {'success': False, 'message': 'Password not changed - Old Password was wrong!'}

    if write_users(file_content):
        return {'success': True, 'user_id': user_id, 'password_key': new_password_key['key']}
    else:
        return {'success': False, 'message': 'Internal Error on Registration'}
