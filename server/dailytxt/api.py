from flask import jsonify, request, Blueprint, current_app, send_file
from datetime import datetime, timedelta
import jwt
import io
from os import environ
from .models import *
from .logs import *
from functools import wraps


api = Blueprint('api', __name__)


def token_required(f):
    @wraps(f)
    def _verify(*args, **kwargs):
        auth_headers = request.headers.get('Authorization', '').split()

        invalid_msg = {
            'message': 'Invalid token. Registration and / or authentication required',
            'authenticated': False
        }
        expired_msg = {
            'message': 'Expired token. Reauthentication required.',
            'authenticated': False
        }
        internal_error = {
            'message': 'An unknown internal server error occured.',
            'authenticated': True
        }

        if len(auth_headers) != 2:
            return jsonify(invalid_msg), 401

        try:
            token = auth_headers[1]
            data = jwt.decode(
                token, current_app.config['SECRET_KEY'], algorithms="HS256")
            if data['sub'] != 0:
                return f(data['sub'], data['key'], *args, **kwargs)

            raise RuntimeError('User not found')

        except jwt.ExpiredSignatureError:
            # 401 is Unauthorized HTTP status code
            return jsonify(expired_msg), 401
        except (jwt.InvalidTokenError) as e:
            print(e)
            return jsonify(invalid_msg), 401
        except (Exception) as e:
            print(e)
            return jsonify(internal_error), 500

    return _verify


@api.route('/register', methods=['POST'])
def register():
    allow_registration = False
    if 'ALLOW_REGISTRATION' in environ:
        if environ.get('ALLOW_REGISTRATION').lower() == 'true':
            allow_registration = True
    if not allow_registration:
        return jsonify({'message': 'Registration is not allowed'}), 500

    data = request.get_json()
    res = register_user(**data)

    if res['success'] == True:
        return jsonify(res), 201
    else:
        return jsonify({'message': res['message']}), 500


@api.route('/login', methods=['POST'])
def login():
    data = request.get_json()
    user = login_user(**data)

    if user['user_id'] == 0:
        return jsonify({'message': 'Invalid credentials', 'authenticated': False}), 401

    token = jwt.encode({
        'sub': user['user_id'],
        'key': user['password_key'].decode(),
        'iat': datetime.now(),
        'exp': datetime.now() + timedelta(days=current_app.config['JWT_EXP_DAYS'])},
        current_app.config['SECRET_KEY']
    )
    return jsonify({'token': token, 'remaining_backup_codes': user['remaining_backup_codes'], 'used_backup_code': user['used_backup_code']})


@api.route('/saveLog', methods=['POST'])
@token_required
def route_saveLog(user_id, key):
    res = saveLog(user_id, key, request.get_json())
    if res['success']:
        return jsonify(res)

    return jsonify(res), 500


@api.route('/loadDay', methods=['POST'])
@token_required
def route_loadDay(user_id, key):
    res = loadDay(user_id, key, request.get_json())
    return jsonify(res)


@api.route('/getDaysWithLogs', methods=['POST'])
@token_required
def route_getDaysWithLogs(user_id, key):
    res = getDaysWithLogs(user_id, key, request.get_json())
    return jsonify(res)


@api.route('search', methods=['POST'])
@token_required
def route_search(user_id, key):
    res = search(user_id, key, request.get_json())
    return jsonify(res)


@api.route('uploadFile', methods=['POST'])
@token_required
def route_uploadFile(user_id, key):
    res = uploadFile(user_id, key, request)
    return jsonify(res)


@api.route('importData', methods=['POST'])
@token_required
def route_importData(user_id, key):
    res = importData(user_id, key, request)
    return jsonify(res)


@api.route('downloadFile', methods=['POST'])
@token_required
def route_downloadFile(user_id, key):
    res = downloadFile(user_id, key, request.get_json())
    return res['file']


@api.route('deleteFile', methods=['POST'])
@token_required
def route_deleteFile(user_id, key):
    res = deleteFile(user_id, key, request.get_json())
    return jsonify(res)


@api.route('removeDay', methods=['POST'])
@token_required
def route_removeDay(user_id, key):
    res = removeDay(user_id, key, request.get_json())
    return jsonify(res)


@api.route('changePassword', methods=['POST'])
@token_required
def route_changePassword(user_id, key):
    res = changePassword(user_id, key, request.get_json())
    if not res['success']:
        return jsonify(res)

    token = jwt.encode({
        'sub': res['user_id'],
        'key': res['password_key'].decode(),
        'iat': datetime.utcnow(),
        'exp': datetime.utcnow() + timedelta(days=30)},
        current_app.config['SECRET_KEY']
    )
    return jsonify({'success': True, 'token': token, 'backup_codes_deleted': res['backup_codes_deleted']})


@api.route('createBackupCodes', methods=['POST'])
@token_required
def route_createBackupCodes(user_id, key):
    res = createBackupCodes(user_id, key, request.get_json())
    return jsonify(res)


@api.route('saveTemplate', methods=['POST'])
@token_required
def route_saveTemplate(user_id, key):
    res = saveTemplate(user_id, key, request.get_json())
    return jsonify(res)


@api.route('removeTemplate', methods=['POST'])
@token_required
def route_removeTemplate(user_id, key):
    res = removeTemplate(user_id, key, request.get_json())
    return jsonify(res)


@api.route('loadTemplates', methods=['POST'])
@token_required
def route_loadTemplates(user_id, key):
    res = loadTemplates(user_id, key)
    return jsonify(res)


@api.route('exportData', methods=['POST'])
@token_required
def route_exportData(user_id, key):
    f = exportData(user_id, key, request.get_json())
    if isinstance(f, dict):
        return jsonify(f)
    return send_file(f, as_attachment=True, download_name="export.zip")


@api.route('getRecentVersion', methods=['POST'])
@token_required
def route_getRecentVersion(user_id, key):
    res = getRecentVersion(user_id, key, request.get_json())
    return jsonify(res)


@api.route('getHistory', methods=['POST'])
@token_required
def route_getHistory(user_id, key):
    res = getHistory(user_id, key, request.get_json())
    return jsonify(res)


@api.route('useHistoryVersion', methods=['POST'])
@token_required
def route_useHistoryVersion(user_id, key):
    res = useHistoryVersion(user_id, key, request.get_json())
    return jsonify(res)


@api.route('addBookmark', methods=['POST'])
@token_required
def route_addBookmark(user_id, key):
    res = addBookmark(user_id, key, request.get_json())
    return jsonify(res)


@api.route('removeBookmark', methods=['POST'])
@token_required
def route_removeBookmark(user_id, key):
    res = removeBookmark(user_id, key, request.get_json())
    return jsonify(res)
