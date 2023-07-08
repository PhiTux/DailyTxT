import shortuuid
from .file_handling import *
from .encryption import *
from .models import *
import zipfile
from io import BytesIO


def downloadFile(user_id, key, download):
    uuid = download['uuid']

    file_content = read_file(uuid)
    if file_content == '':
        return {'success': False}
    file = decrypt_file_by_userid(file_content, user_id, key)

    return {'success': True, 'file': file['text']}


def removeDay(user_id, key, delete):
    file_content = read_log(user_id, delete['year'], delete['month'])

    if isinstance(file_content, dict):
        new_log = {'days': []}
        for day in file_content['days']:
            if day['day'] == delete['day']:
                if 'files' in day.keys():
                    for f in day['files']:
                        delete_file_by_uuid(f['uuid_filename'])
            else:
                new_log['days'].append(day)
        if write_log(user_id, delete['year'], delete['month'], new_log):
            return {'success': True}

    return {'success': False, 'message': 'Internal error!? File seems not to exist.'}


def deleteFile(user_id, key, delete):

    file_content = read_log(user_id, delete['year'], delete['month'])

    if isinstance(file_content, dict):
        for day in file_content['days']:
            if day['day'] == delete['day']:
                new_files = []
                for f in day['files']:
                    if f['uuid_filename'] != delete['uuid']:
                        new_files.append(f)
                day['files'] = new_files
                if write_log(user_id, delete['year'], delete['month'], file_content):
                    res = delete_file_by_uuid(delete['uuid'])
                    return res

    return {'success': False, 'message': 'Internal error!? File seems not to exist.'}


def uploadFile(user_id, key, upload):
    file_content = read_log(
        user_id, int(upload.form.get('year')), int(upload.form.get('month'))
    )

    file = upload.files['file']
    if file.filename == '':
        return {'success': False}

    enc_file = encrypt_file_by_userid(file.read(), user_id, key)
    if not enc_file['success']:
        return {'success': False}

    uuid_filename = shortuuid.uuid()
    if not write_file(enc_file['text'], uuid_filename):
        return {'success': False}

    enc_filename = encrypt_by_userid(file.filename, user_id, key)

    new_file = {
        'enc_filename': enc_filename['text'],
        'uuid_filename': uuid_filename
    }

    written = False
    if isinstance(file_content, dict):
        for day in file_content['days']:
            if day['day'] == int(upload.form.get('day')):
                if not 'files' in day.keys():
                    day['files'] = []
                day['files'].append(new_file)
                written = True
        if not written:
            file_content['days'].append(
                {'day': int(upload.form.get('day')), 'files': [new_file]})
    else:
        file_content = {
            'days': [{'day': int(upload.form.get('day')), 'files': [new_file]}]}

    if write_log(user_id, int(upload.form.get('year')), int(upload.form.get('month')), file_content):
        return {'success': True, 'uuid_filename': uuid_filename}
    # file.save(os.path.join(current_app.config['DATA_PATH'], 'test.jpg'))

    return {'success': False}


def importData(user_id, key, f):
    zf = f.files['file']
    path = import_directory(user_id)

    # extract zip-content directly without saving zip itself
    with zipfile.ZipFile(zf, 'r') as zip_ref:
        zip_ref.extractall(path)

    # go through decrypted content and 'import'

    # find export root - zip-structure is ./data/<userID>/export/
    counter = 3
    for root, dirs, files in os.walk(path):
        if counter == 0:
            path = root
            break
        counter -= 1

    # find all files
    log_paths = []
    file_paths = []
    for root, dirs, files in os.walk(path):
        for f in files:
            if (not root.endswith('files')):
                log_paths.append(os.path.join(root, f))
            else:
                file_paths.append(os.path.join(root, f))

    for l in log_paths:
        year = int(l.split('/')[-2])
        month = int(l.split('/')[-1].split('.')[0])
        file_content = read_log(user_id, year, month)
        file_content_import = read_json(l)

        if not isinstance(file_content, dict):
            file_content = {'days': []}

        if isinstance(file_content_import, dict):
            for day_import in file_content_import['days']:
                file_content = move_text_to_history(
                    file_content, day_import['day'])
                if 'history' in day_import.keys():
                    for history in sorted(day_import['history'], key=lambda h: h['version']):
                        file_content = append_history(
                            file_content, history, day_import['day'], user_id, key)
                if 'text' in day_import.keys() and 'date_written' in day_import.keys():
                    file_content = set_new_text(
                        file_content, day_import, user_id, key)
                if 'files' in day_import.keys():
                    for file in day_import['files']:
                        file_content = import_file(
                            file_content, file, day_import['day'], file_paths, user_id, key)

        write_log(user_id, year, month, file_content)

    delete_import_directory(user_id)

    return {'success': True}


def import_file(file_content, import_file, day_date, file_paths, user_id, key):

    # handle filename
    written = False

    enc_filename = encrypt_by_userid(import_file['filename'], user_id, key)
    new_file = {
        'enc_filename': enc_filename['text'], 'uuid_filename': import_file['uuid']}

    for day in file_content['days']:
        if day['day'] == day_date:
            if 'files' in day.keys():
                for file in day['files']:
                    if file['uuid_filename'] == import_file['uuid']:
                        return file_content
            else:
                day['files'] = []
            day['files'].append(new_file)
            written = True

    if not written:
        file_content['days'].append({'day': day_date, 'files': [new_file]})

    # handle file-content
    filepath = ''
    for p in file_paths:
        if p.split('/')[-1].startswith(import_file['uuid']):
            filepath = p
            break
    with open(filepath, 'rb') as f:
        enc_file = encrypt_file_by_userid(f.read(), user_id, key)
        write_file(enc_file['text'], import_file['uuid'])

    return file_content


def set_new_text(file_content, import_text, user_id, key):
    written = False

    enc_res = {'text': ''}
    if import_text['text'] != '':
        enc_res = encrypt_by_userid(import_text['text'], user_id, key)

    for day in file_content['days']:
        if day['day'] == import_text['day']:
            day['text'] = enc_res['text']
            day['date_written'] = import_text['date_written']
            written = True
            break

    if not written:
        file_content['days'].append(
            {'day': import_text['day'], 'text': enc_res['text'], 'date_written': import_text['date_written']})

    return file_content


def append_history(file_content, history, day_date, user_id, key):
    written = False

    enc_res = {'text': ''}
    if history['text'] != '':
        enc_res = encrypt_by_userid(history['text'], user_id, key)

    for day in file_content['days']:
        if day['day'] == day_date:
            if not 'history' in day.keys():
                day['history'] = []

            day['history'].append({'version': len(
                day['history']) + 1, 'date_written': history['date_written'], 'text': enc_res['text']})
            written = True
            break

    if not written:
        file_content['days'].append({'day': day_date, 'history': [
                                    {'version': 1, 'date_written': history['date_written'], 'text': enc_res['text']}]})

    return file_content


def move_text_to_history(file_content, day_date):
    for day in file_content['days']:
        if day['day'] == day_date:
            if not 'history' in day.keys():
                day['history'] = []
            if 'text' in day.keys() and 'date_written' in day.keys():
                day['history'].append({'version': len(
                    day['history']) + 1, 'date_written': day['date_written'], 'text': day['text']})
            break
    return file_content


def saveLog(user_id, key, log):
    file_content = read_log(
        user_id, log['year'], log['month']
    )

    logIsEmpty = False
    if log['log'] == "":
        logIsEmpty = True

    enc_res = encrypt_by_userid(log['log'], user_id, key)
    if not enc_res['success']:
        return {'success': False, 'message': 'Encryption error'}

    # in case a new day gets started
    new_day = {'day': log['day'], 'text': enc_res['text'],
               'date_written': log['date_written']}

    written = False
    if isinstance(file_content, dict):
        for day in file_content['days']:
            if day['day'] == log['day']:
                # append 'old' text to history:
                if not 'history' in day.keys():
                    day['history'] = []
                if 'text' in day.keys() and 'date_written' in day.keys():
                    day['history'].append({'version': len(
                        day['history']) + 1, 'date_written': day['date_written'], 'text': day['text']})

                # save new text
                if logIsEmpty:
                    day['text'] = ""
                else:
                    day['text'] = enc_res['text']
                day['date_written'] = log['date_written']
                written = True
                break
        if not written:
            file_content['days'].append(new_day)
    else:
        file_content = {'days': [new_day]}

    if write_log(user_id, log['year'], log['month'], file_content):
        return {'success': True}

    return {'success': False, 'message': 'Error on saving new Diary Log'}


def getHistory(user_id, key, date):
    file_content = read_log(user_id, date['year'], date['month'])

    if isinstance(file_content, dict):
        for day in file_content['days']:
            if day['day'] == date['day']:
                if not 'history' in day.keys():
                    return {'success': False, 'message': 'No history available yet!'}
                res = {'success': True, 'history': []}
                for h in day['history']:
                    if h['text'] != '':
                        dec_res = decrypt_by_userid(h['text'], user_id, key)
                        if not dec_res['success']:
                            continue
                        h['text'] = dec_res['text']
                    res['history'].append(h)
                return res

    return {'success': False, 'message': 'No history available yet!'}


def useHistoryVersion(user_id, key, data):
    file_content = read_log(user_id, data['year'], data['month'])

    if isinstance(file_content, dict):
        for day in file_content['days']:
            if day['day'] == data['day']:
                if not 'history' in day.keys():
                    return {'success': False, 'message': 'No history available yet!'}
                # move actual newest text into history
                day['history'].append({'version': len(
                    day['history']) + 1, 'date_written': day['date_written'], 'text': day['text']})

                # and move selected history version to 'newest text'
                for h in day['history']:
                    if h['version'] == data['version']:
                        day['text'] = h['text']
                        day['date_written'] = h['date_written']
                        if write_log(user_id, data['year'], data['month'], file_content):
                            return {'success': True}
                        else:
                            return {'success': False, 'message': 'Server-Error on writing changes!'}

    return {'success': False, 'message': 'Server error - invalid file'}


def loadDay(user_id, key, date):
    file_content = read_log(user_id, date['year'], date['month'])

    day_info = {
        'enc_error': False,
        'text': '',
        'date_written': '',
        'files': [],
        'historyAvailable': False
    }

    if isinstance(file_content, dict):
        for day in file_content['days']:
            if day['day'] == date['day']:
                if 'text' in day.keys():
                    if day['text'] == '':
                        day_info['text'] = ''
                    else:
                        dec_res = decrypt_by_userid(day['text'], user_id, key)
                        if not dec_res['success']:
                            day_info['enc_error'] = True
                            return day_info
                        day_info['text'] = dec_res['text']
                    day_info['date_written'] = day['date_written']
                if 'files' in day.keys():
                    for f in day['files']:
                        filename = decrypt_by_userid(
                            f['enc_filename'], user_id, key)
                        if not filename['success']:
                            continue
                        day_info['files'].append(
                            {'uuid': f['uuid_filename'], 'filename': filename['text']})
                if 'history' in day.keys():
                    if len(day['history']) > 0:
                        day_info['historyAvailable'] = True

    return day_info


def getDaysWithLogs(user_id, key, page):
    file_content = read_log(user_id, page['year'], page['month'])

    bookmarks = []
    logs = []
    files = []

    if isinstance(file_content, dict):
        for day in file_content['days']:
            if 'isBookmarked' in day.keys() and day['isBookmarked']:
                bookmarks.append(day['day'])
            if 'text' in day.keys() and day['text'] != "":
                logs.append(day['day'])
            if 'files' in day.keys() and day['files'] != []:
                files.append(day['day'])

    return {'logs': logs, 'files': files, 'bookmarks': bookmarks}


def createOrArray(s):
    if s.startswith('"') and s.endswith('"'):
        return [s.strip('"').lower()]

    orArray = s.split('|')
    orArray = [o.strip().lower() for o in orArray]

    return orArray


def getStartIndex(text, index):
    # try to find a whitespace two places before index

    if index == 0:
        return index

    startIndex = text.rfind(' ', 0, index-1)
    if startIndex == -1:
        if text[0] == ' ':
            return 1
        return 0

    startIndex = text.rfind(' ', 0, startIndex)
    if startIndex == -1:
        if text[0] == ' ':
            return 1
        return 0

    return startIndex + 1


def getEndIndex(text, index):
    # try to find a whitespace three places after index

    endIndex = text.find(' ', index)
    if endIndex == -1:
        return len(text)

    endIndex = text.find(' ', endIndex+1)
    if endIndex == -1:
        return len(text)

    endIndex = text.find(' ', endIndex+1)
    if endIndex == -1:
        return len(text)

    endIndex = text.find(' ', endIndex+1)
    if endIndex == -1:
        return len(text)

    return endIndex


def search(user_id, key, request):
    searchArray = createOrArray(request['searchString'])

    allFiles = getAllFiles(user_id)

    enc_key = get_enc_key(user_id, key)
    if enc_key == '':
        return {'success': False}

    res = []

    for file in allFiles:
        file_content = read_json(file)

        if isinstance(file_content, dict):
            for day in file_content['days']:
                if not 'text' in day.keys() or day['text'] == "":
                    continue
                dec_res = decrypt_by_key(
                    day['text'].encode(), enc_key).decode()
                dec_res_upper = dec_res.lower()
                for search in searchArray:
                    # if search in dec_res:
                    index = dec_res_upper.find(search)

                    if index != -1:

                        startIndex = getStartIndex(dec_res, index)
                        endIndex = getEndIndex(dec_res, index)

                        snippetStart = dec_res[startIndex:index]
                        snippetBold = dec_res[index:index+len(
                            search)]
                        snippetEnd = dec_res[index+len(search):endIndex]

                        res.append({'year': file.split(
                            '/')[-2], 'month': file.split('/')[-1].split('.')[0], 'day': str("{:02d}".format(day['day'])),
                            'snippetStart': snippetStart, 'snippetBold': snippetBold, 'snippetEnd': snippetEnd})
                        break

    return {'success': True, 'results': res}


def filename_to_export(old):
    temp = old.split('/')
    temp[-3] = temp[-3] + ('/export')
    return '/'.join(temp)


def exportData(user_id, key, p):
    with lock:
        pwd_check = check_for_password_and_backup_codes(user_id, p['password'])
        if not pwd_check['success']:
            return {'success': False, 'message': 'Wrong password!'}

        allLogs = getAllFiles(user_id)
        allFiles = []

        enc_key = get_enc_key(user_id, key)
        if enc_key == '':
            return {'success': False}

        for logfile in allLogs:
            file_content = read_json(logfile)

            if isinstance(file_content, dict):
                for day in file_content['days']:
                    if 'files' in day.keys():
                        new_files = []
                        for f in day['files']:
                            new_f = {'uuid': f['uuid_filename'], 'filename': decrypt_by_key(
                                f['enc_filename'].encode(), enc_key).decode('utf-8')}
                            new_files.append(new_f)
                            allFiles.append(new_f)
                        day['files'] = new_files
                    if 'text' in day.keys():
                        if day['text'] != '':
                            day['text'] = decrypt_by_key(
                                day['text'].encode(), enc_key).decode('utf-8')
                    if 'history' in day.keys():
                        for h in day['history']:
                            if h['text'] != '':
                                h['text'] = decrypt_by_key(
                                    h['text'].encode(), enc_key).decode('utf-8')

                write_export_log(user_id, logfile.split(
                    '/')[-2], logfile.split('/')[-1].split('.')[0], file_content)

        allExportLogs = set(map(filename_to_export, allLogs))

        mem_zip = BytesIO()

        with zipfile.ZipFile(mem_zip, mode="w", compression=zipfile.ZIP_DEFLATED) as zf:
            for f in allExportLogs:
                zf.write(f)

            for f in allFiles:
                file_content = read_file(f['uuid'])
                res = decrypt_file_by_userid(
                    file_content, user_id, key)
                if res['success']:
                    filename = write_export_file(
                        user_id, f['filename'], f['uuid'], res['text'])
                    if filename != '':
                        zf.write(filename)

            delete_export_directory(user_id)

        mem_zip.seek(0)
        return mem_zip


def saveTemplate(user_id, key, p):
    file_content = read_templates(user_id)

    enc_res_name = encrypt_by_userid(p['name'], user_id, key)
    if not enc_res_name['success']:
        return {'success': False, 'message': 'Encryption error'}

    enc_res_text = encrypt_by_userid(p['text'], user_id, key)
    if not enc_res_text['success']:
        return {'success': False, 'message': 'Encryption error'}

    if isinstance(file_content, dict):
        if not 'templates' in file_content.keys():
            return {'success': False, 'message': 'File error - templates corrupted'}

        # number 0 means 'new template'
        if p['number'] == 0:
            max_number = 0
            for template in file_content['templates']:
                if template['number'] > max_number:
                    max_number = template['number']
            max_number += 1
            file_content['templates'].append({"number": max_number, 'name': enc_res_name['text'],
                                              'text': enc_res_text['text']})
        else:
            for template in file_content['templates']:
                if template['number'] == p['number']:
                    template['name'] = enc_res_name['text']
                    template['text'] = enc_res_text['text']
                    break
    else:
        if p['number'] == 0:
            file_content = {'templates': [{'number': 1, 'name': enc_res_name['text'],
                                           'text': enc_res_text['text']}]}
        else:
            return {'success': False, 'message': 'File error - templates file corrupted'}

    if write_templates(user_id, file_content):
        return {'success': True}

    return {'success': False, 'message': 'Error on saving template'}


def removeTemplate(user_id, key, p):
    file_content = read_templates(user_id)
    new_content = {'templates': []}

    if isinstance(file_content, dict):
        if not 'templates' in file_content.keys():
            return {'success': False, 'message': 'File error - templates file corrupted'}
        for template in file_content['templates']:
            if template['number'] != p['number']:
                new_content['templates'].append(template)
        if write_templates(user_id, new_content):
            return {'success': True}
        return {'success': False, 'message': 'Error on deleting template'}
    else:
        return {'success': False, 'message': 'File error - templates file already empty'}


def loadTemplates(user_id, key):
    file_content = read_templates(user_id)

    if isinstance(file_content, dict):
        if not 'templates' in file_content.keys():
            return {'success': False, 'message': 'File error - templates corrupted'}

        templates_dec = []
        for template in file_content['templates']:
            name_dec_res = decrypt_by_userid(template['name'], user_id, key)
            if not name_dec_res['success']:
                return {'success': False, 'message': 'Encryption error'}

            text_dec_res = decrypt_by_userid(template['text'], user_id, key)
            if not text_dec_res['success']:
                return {'success': False, 'message': 'Encryption error'}

            templates_dec.append(
                {'number': template['number'], 'name': name_dec_res['text'], 'text': text_dec_res['text']})

        return {'success': True, 'templates': templates_dec}

    else:
        return {'success': True, 'templates': []}


def addBookmark(user_id, key, date):
    file_content = read_log(
        user_id, date['year'], date['month']
    )

    # in case a new day gets started
    new_day = {'day': date['day'], 'isBookmarked': True}

    written = False
    if isinstance(file_content, dict):
        for day in file_content['days']:
            if day['day'] == date['day']:
                # set bookmarked
                day['isBookmarked'] = True

                written = True
                break
        if not written:
            file_content['days'].append(new_day)
    else:
        file_content = {'days': [new_day]}

    if write_log(user_id, date['year'], date['month'], file_content):
        return {'success': True}

    return {'success': False, 'message': 'Error on setting bookmark'}


def removeBookmark(user_id, key, date):
    file_content = read_log(
        user_id, date['year'], date['month']
    )

    if isinstance(file_content, dict):
        for day in file_content['days']:
            if day['day'] == date['day']:
                # set not bookmarked
                day['isBookmarked'] = False
                break

    if write_log(user_id, date['year'], date['month'], file_content):
        return {'success': True}

    return {'success': False, 'message': 'Error on removing bookmark'}
