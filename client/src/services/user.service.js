import axios from 'axios'
import authHeader from './auth-headers'

const API_URL = '/api/'

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

  exportData() {
    return axios.post(
      API_URL + 'exportData',
      {},
      {
        headers: authHeader(),
        responseType: 'blob'
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

  removeDay(date) {
    return axios.post(
      API_URL + 'removeDay',
      {
        year: date.getFullYear(),
        month: date.getMonth() + 1,
        day: date.getDate()
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
}

export default new UserService()
