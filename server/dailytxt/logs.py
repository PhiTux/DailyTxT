import shortuuid
from .file_handling import *
from .encryption import *
import zipfile
from io import BytesIO


def downloadFile(user_id, key, download):
    uuid = download['uuid']

    file_content = read_file(uuid)
    if file_content == '':
        return {'success': False}
    file = decrypt_file_by_userid(file_content, user_id, key)

    return {'success': True, 'file': file['text']}


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


def saveLog(user_id, key, log):
    file_content = read_log(
        user_id, log['year'], log['month']
    )

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
                day['history'].append({'version': len(
                    day['history']) + 1, 'date_written': day['date_written'], 'text': day['text']})

                # save new text
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

    logs = []
    files = []

    if isinstance(file_content, dict):
        for day in file_content['days']:
            if 'text' in day.keys():
                logs.append(day['day'])
            if 'files' in day.keys() and day['files'] != []:
                files.append(day['day'])

    return {'logs': logs, 'files': files}


def removeDay(user_id, key, date):
    file_content = read_log(user_id, date['year'], date['month'])

    res = {'success': False}

    new_days = []

    if isinstance(file_content, dict):
        for day in file_content['days']:
            if day['day'] == date['day']:
                if 'files' in day.keys() and day['files'] != []:
                    new_days.append({'day': day['day'], 'files': day['files']})

            else:
                new_days.append(day)
    else:
        return res

    file_content['days'] = new_days

    if write_log(user_id, date['year'], date['month'], file_content):
        return {'success': True}

    return res


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
                if not 'text' in day.keys():
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


def exportData(user_id, key):
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
                    day['text'] = decrypt_by_key(
                        day['text'].encode(), enc_key).decode('utf-8')
                if 'history' in day.keys():
                    for h in day['history']:
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
