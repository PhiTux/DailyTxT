import json
import os
import errno
from flask import current_app
from pathlib import Path
from shutil import rmtree


def users_file():
    return os.path.join(current_app.config['DATA_PATH'], current_app.config['USERS_FILE'])


def log_file(user_id, year, month):
    return os.path.join(current_app.config['DATA_PATH'], str(user_id), str(year),  str("{:02d}".format(month)) + '.json')


def export_log_file(user_id, year, month):
    return os.path.join(current_app.config['DATA_PATH'], str(user_id), 'export', str(year),  str("{:02d}".format(month)) + '.json')


def read_log(user_id, year, month):
    return read_json(log_file(user_id, year, month))


def write_log(user_id, year, month, data):
    return write_json(log_file(user_id, year, month), data)


def write_export_log(user_id, year, month, data):
    return write_json(export_log_file(user_id, int(year), int(month)), data)


def write_export_file(user_id, filename, uuid, file_content):
    new_filename = os.path.join(current_app.config['DATA_PATH'], str(
        user_id), 'export', 'files', uuid + '_' + filename)

    if not os.path.exists(os.path.dirname(new_filename)):
        try:
            os.makedirs(os.path.dirname(new_filename))
        except OSError as exc:  # Guard against race condition
            if exc.errno != errno.EEXIST:
                raise

    try:
        with open(new_filename, 'wb') as outfile:
            outfile.write(file_content)
            return new_filename
    except IOError:
        return ''


def read_users():
    return read_json(users_file())


def write_users(data):
    return write_json(users_file(), data)


def delete_file_by_uuid(uuid_filename):
    path = os.path.join(
        current_app.config['DATA_PATH'], 'files', uuid_filename)

    try:
        os.remove(path)
    except:
        return {'success': False, 'message': 'Error while deleting file!'}

    return {'success': True}


def delete_export_directory(user_id):
    rmtree(os.path.join(
        current_app.config['DATA_PATH'], str(user_id), 'export'))


def read_file(uuid_filename):
    path = os.path.join(
        current_app.config['DATA_PATH'], 'files', uuid_filename)

    try:
        with open(path, 'r') as infile:
            return infile.read()
    except:
        return ''


def write_file(content, uuid_filename):
    path = os.path.join(
        current_app.config['DATA_PATH'], 'files', uuid_filename)
    if not os.path.exists(os.path.dirname(path)):
        try:
            os.makedirs(os.path.dirname(path))
        except OSError as exc:  # Guard against race condition
            if exc.errno != errno.EEXIST:
                raise

    try:
        with open(path, 'w') as outfile:
            outfile.write(content)
            return True
    except IOError:
        return False


def getAllFiles(user_id):
    root = os.path.join(current_app.config['DATA_PATH'], str(user_id))
    res = []

    for path, subdirs, files in os.walk(root):
        for name in files:
            res.append(os.path.join(path, name))

    return res


def read_json(path):
    try:
        with open(path, 'r') as infile:
            return json.load(infile)
    except:
        return ''


def write_json(path, data):
    if not os.path.exists(os.path.dirname(path)):
        try:
            os.makedirs(os.path.dirname(path))
        except OSError as exc:  # Guard against race condition
            if exc.errno != errno.EEXIST:
                raise

    try:
        with open(path, 'w') as outfile:
            json.dump(data, outfile, ensure_ascii=False, indent=current_app.config['DATA_INDENT'])
            return True
    except IOError:
        return False
