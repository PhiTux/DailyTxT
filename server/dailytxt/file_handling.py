import json
import os
import errno
import requests
import re
from flask import current_app
from shutil import rmtree
from datetime import datetime


def import_directory(user_id):
    return os.path.join(current_app.config['DATA_PATH'], str(user_id), 'import_tmp')


def read_templates(user_id):
    return read_json(template_file(user_id))


def write_templates(user_id, data):
    return write_json(template_file(user_id), data)


def template_file(user_id):
    return os.path.join(current_app.config['DATA_PATH'], str(user_id), current_app.config['TEMPLATES_FILE'])


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


def delete_import_directory(user_id):
    rmtree(os.path.join(
        current_app.config['DATA_PATH'], str(user_id), 'import_tmp'))


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
            if name != "templates.json":
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
            json.dump(data, outfile, ensure_ascii=False,
                      indent=current_app.config['DATA_INDENT'])
            return True
    except IOError:
        return False


def major_minor_micro(version):
    major, minor, micro = re.search('(\d+)\.(\d+)\.(\d+)', version).groups()

    return int(major), int(minor), int(micro)


def docker_api_get_recent_version():
    r = requests.get(
        'https://hub.docker.com/v2/repositories/phitux/dailytxt/tags')
    r = r.json()
    versions = [v['name'] for v in r['results'] if v['name']
                != 'latest' and v['name'].count('.') == 2]

    return max(versions, key=major_minor_micro)


def getRecentVersion(user_id, key, v):
    if 'ENABLE_UPDATE_CHECK' in os.environ:
        if os.environ.get('ENABLE_UPDATE_CHECK').lower() == 'false':
            return {'recent_version': v['client_version']}

    datestring = "%Y-%m-%d_%H:%M:%S"

    filename = os.path.join(
        current_app.config['DATA_PATH'], 'recent_version.json')
    # load file with recent version
    file_content = read_json(filename)

    write_new = False
    if (file_content == ''):
        write_new = True
    else:
        # check if version was checked in the last hour
        diff = datetime.now() - \
            datetime.strptime(file_content['timestamp'], datestring)
        if diff.total_seconds() > 3600:
            write_new = True

    if write_new:
        file_content = {'timestamp': datetime.now().strftime(datestring),
                        'version': docker_api_get_recent_version()}

        write_json(filename, file_content)

    return {'recent_version': max([file_content['version'], v['client_version']], key=major_minor_micro)}
