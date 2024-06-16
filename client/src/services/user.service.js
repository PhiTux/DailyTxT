import axios from 'axios'
import authHeader from './auth-headers'

const API_URL =
  process.env.NODE_ENV === 'production'
    ? window.location.pathname.replace(/\/+$/, '') + process.env.VUE_APP_API_URL
    : process.env.VUE_APP_API_URL

class UserService {
  getHistory(dateSelected) {
    return axios.post(
      API_URL + 'getHistory',
      {
        year: dateSelected.getFullYear(),
        month: dateSelected.getMonth() + 1,
        day: dateSelected.getDate()
      },
      { headers: authHeader() }
    )
  }

  useHistoryVersion(version, dateSelected) {
    return axios.post(
      API_URL + 'useHistoryVersion',
      {
        version: version,
        year: dateSelected.getFullYear(),
        month: dateSelected.getMonth() + 1,
        day: dateSelected.getDate()
      },
      { headers: authHeader() }
    )
  }

  downloadFile(uuid) {
    return axios.post(
      API_URL + 'downloadFile',
      { uuid: uuid },
      { headers: authHeader(), responseType: 'blob' }
    )
  }

  uploadFile(file, dateSelected, onUploadProgress) {
    let formData = new FormData()
    formData.append('file', file)
    formData.append('year', dateSelected.getFullYear())
    formData.append('month', dateSelected.getMonth() + 1)
    formData.append('day', dateSelected.getDate())

    return axios.post(API_URL + 'uploadFile', formData, {
      headers: authHeader(),
      onUploadProgress
    })
  }

  importData(f, onUploadProgress) {
    let formData = new FormData()
    formData.append('file', f)

    return axios.post(API_URL + 'importData', formData, {
      headers: authHeader(),
      onUploadProgress
    })
  }

  deleteFile(uuid, dateSelected) {
    return axios.post(
      API_URL + 'deleteFile',
      {
        uuid: uuid,
        year: dateSelected.getFullYear(),
        month: dateSelected.getMonth() + 1,
        day: dateSelected.getDate()
      },
      { headers: authHeader() }
    )
  }

  removeDay(dateSelected) {
    return axios.post(
      API_URL + 'removeDay',
      {
        year: dateSelected.getFullYear(),
        month: dateSelected.getMonth() + 1,
        day: dateSelected.getDate()
      },
      { headers: authHeader() }
    )
  }

  changePassword(old_password, new_password) {
    return axios.post(
      API_URL + 'changePassword',
      {
        old_password: old_password,
        new_password: new_password
      },
      { headers: authHeader() }
    )
  }

  createBackupCodes(password) {
    return axios.post(
      API_URL + 'createBackupCodes',
      {
        password: password
      },
      { headers: authHeader() }
    )
  }

  saveTemplate(number, name, text) {
    return axios.post(
      API_URL + 'saveTemplate',
      {
        number: number,
        name: name,
        text: text
      },
      { headers: authHeader() }
    )
  }

  removeTemplate(number) {
    return axios.post(
      API_URL + 'removeTemplate',
      {
        number: number
      },
      { headers: authHeader() }
    )
  }

  loadTemplates() {
    return axios.post(API_URL + 'loadTemplates', {}, { headers: authHeader() })
  }

  exportData(password) {
    return axios.post(
      API_URL + 'exportData',
      {
        password: password
      },
      {
        headers: authHeader(),
        responseType: 'blob'
      }
    )
  }

  getRecentVersion(version) {
    return axios.post(
      API_URL + 'getRecentVersion',
      { client_version: version },
      {
        headers: authHeader()
      }
    )
  }

  saveLog(text, dateSelected, date_written) {
    return axios.post(
      API_URL + 'saveLog',
      {
        log: text,
        year: dateSelected.getFullYear(),
        month: dateSelected.getMonth() + 1,
        day: dateSelected.getDate(),
        date_written: date_written
      },
      { headers: authHeader() }
    )
  }

  loadDay(dateSelected) {
    return axios.post(
      API_URL + 'loadDay',
      {
        year: dateSelected.getFullYear(),
        month: dateSelected.getMonth() + 1,
        day: dateSelected.getDate()
      },
      { headers: authHeader() }
    )
  }

  getDaysWithLogs(page) {
    return axios.post(
      API_URL + 'getDaysWithLogs',
      {
        year: page.year,
        month: page.month
      },
      { headers: authHeader() }
    )
  }

  search(searchString) {
    return axios.post(
      API_URL + 'search',
      {
        searchString: searchString
      },
      { headers: authHeader() }
    )
  }

  addBookmark(date) {
    return axios.post(
      API_URL + 'addBookmark',
      {
        year: date.getFullYear(),
        month: date.getMonth() + 1,
        day: date.getDate()
      },
      { headers: authHeader() }
    )
  }

  removeBookmark(date) {
    return axios.post(
      API_URL + 'removeBookmark',
      {
        year: date.getFullYear(),
        month: date.getMonth() + 1,
        day: date.getDate()
      },
      { headers: authHeader() }
    )
  }
}

export default new UserService()
