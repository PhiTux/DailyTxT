from werkzeug.security import generate_password_hash, check_password_hash
from .file_handling import *
from .encryption import *
from shortuuid import ShortUUID
from threading import Lock

lock = Lock()


def register_user(username, password):
    with lock:
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


# Checks, if the given password is either the original password or one of the backup codes.
# If it is a backup code, then it decrypts the password-content of this backup code, which results in
# the original password (!) and it returns this original password
def check_for_password_and_backup_codes(user_id, password):
    lock2 = Lock()
    with lock2:
        data = read_users()

        if isinstance(data, dict):
            for user in data['users']:
                if user['user_id'] == user_id:
                    if check_password_hash(user['password'], password):
                        return {'success': True, 'password': password, 'remaining_backup_codes': 0 if not 'backup_codes' in user.keys() else len(user['backup_codes']), 'used_backup_code': False}
                    else:
                        if 'backup_codes' in user.keys():
                            for b in user['backup_codes'][:]:
                                if check_password_hash(b['password'], password):
                                    # decrypt and return password
                                    password_key = get_password_key(
                                        password.encode(), b['salt'].encode())
                                    orig_password = decrypt_by_key(
                                        b['enc_orig_password'].encode(), password_key).decode()

                                    # delete backup-code
                                    user['backup_codes'].remove(b)

                                    if write_users(data):
                                        return {'success': True, 'password': orig_password, 'remaining_backup_codes': len(user['backup_codes']), 'used_backup_code': True}
                                    else:
                                        return {'success': False}
                        else:
                            return {'success': False}

        return {'success': False}


def login_user(username, password):
    data = read_users()

    if isinstance(data, dict):
        for user in data['users']:
            if user['username'].lower() == username.lower():
                pwd_check = check_for_password_and_backup_codes(
                    user['user_id'], password)
                if pwd_check['success']:
                    return {'user_id': user['user_id'], 'password_key': get_password_key(pwd_check['password'].encode(), user['salt'].encode()), 'remaining_backup_codes': pwd_check['remaining_backup_codes'], 'used_backup_code': pwd_check['used_backup_code']}

    return {'user_id': 0, 'password_key': ''}


def createBackupCodes(user_id, key, p):
    with lock:
        pwd_check = check_for_password_and_backup_codes(user_id, p['password'])
        if not pwd_check['success']:
            return {'success': False, 'message': 'Wrong password!'}

        file_content = read_users()
        new_backup_codes = []

        if isinstance(file_content, dict):
            for user in file_content['users']:
                if user['user_id'] == user_id:
                    user['backup_codes'] = []
                    for x in range(6):
                        backup_code = ShortUUID().random(length=10)
                        backup_code_key = create_new_key_from_password(
                            backup_code)
                        enc_orig_password = encrypt_by_key(
                            pwd_check['password'].encode(), backup_code_key['key'])

                        user['backup_codes'].append({'password': generate_password_hash(backup_code, method='sha256'), 'salt': backup_code_key['salt'].decode(
                        ), 'enc_orig_password': enc_orig_password.decode()})
                        new_backup_codes.append(backup_code)
        else:
            return {'success': False}

        if write_users(file_content):
            return {'success': True, 'backupCodes': new_backup_codes}
        else:
            return {'success': False, 'message': 'Internal Error on generating new backup codes!'}


def changePassword(user_id, old_password_key, p):

    with lock:
        enc_key = get_enc_key(user_id, old_password_key)
        new_password_key = create_new_key_from_password(p['new_password'])
        enc_enc_key = encrypt_by_key(enc_key, new_password_key['key'])

        pwd_check = check_for_password_and_backup_codes(
            user_id, p['old_password'])
        if not pwd_check['success']:
            return {'success': False, 'message': 'Password not changed - Old Password was wrong!'}

        file_content = read_users()
        backup_codes_deleted = False

        if isinstance(file_content, dict):
            for user in file_content['users']:
                if user['user_id'] == user_id:
                    if check_password_hash(user['password'], pwd_check['password']):
                        user['password'] = generate_password_hash(
                            p['new_password'], method='sha256')
                        user['salt'] = new_password_key['salt'].decode()
                        user['enc_enc_key'] = enc_enc_key.decode()

                        if 'backup_codes' in user.keys():
                            if len(user['backup_codes']) > 0:
                                backup_codes_deleted = True

                        user['backup_codes'] = []
                    else:
                        return {'success': False, 'message': 'Password not changed - Old Password was wrong!'}

        if write_users(file_content):
            return {'success': True, 'user_id': user_id, 'password_key': new_password_key['key'], 'backup_codes_deleted': backup_codes_deleted}
        else:
            return {'success': False, 'message': 'Internal Error on Registration'}
