<template>
  <div class="main" @dragenter.prevent="dragging = true" @dragover.prevent>
    <div
      class="dropzone"
      v-if="dragging"
      @dragend="dragging = false"
      @drop.prevent="uploadFilesDrop"
      @dragover.prevent
      @dragleave.prevent="dragging = false"
    >
      {{ $t('upload-files-overlay') }}
    </div>
    <div id="modal_delete_file" class="modal">
      <div class="modal-content">
        <h4>{{ $t('modal-delete-file-header') }}</h4>
        <p>
          <i18n path="modal-delete-file-text">
            <b>{{ fileToDelete.filename }}</b>
          </i18n>
        </p>
      </div>
      <div class="modal-footer">
        <a class="modal-close waves-effect waves-red btn-flat">{{
          $t('abort')
        }}</a>
        <a
          class="modal-close waves-effect waves-green btn-flat"
          @click="deleteFile()"
          >{{ $t('delete') }}</a
        >
      </div>
    </div>
    <div id="modal_import_data" class="modal">
      <div class="modal-content">
        <h4>{{ $t('modal-import-data-header') }}</h4>

        <li class="upload-icons">
          <i v-if="importStep == 1" class="material-icons">arrow_forward</i>
          <i v-if="importStep == 2 || importStep == 3" class="material-icons"
            >check</i
          >
        </li>
        {{ $t('modal-import-data-text-1') }}
        <br />
        <li class="collection-item importProgress">
          <div class="progress">
            <div
              class="determinate"
              :style="{ width: importProgress + '%' }"
            ></div>
          </div>
        </li>
        <div class="divider"></div>
        <li class="upload-icons">
          <i v-if="importStep == 2" class="material-icons">arrow_forward</i>
          <i v-if="importStep == 3" class="material-icons">check</i>
        </li>
        {{ $t('modal-import-data-text-2') }}
        <div class="col s2 m4 l3" id="loading" v-if="importStep == 2">
          <div class="preloader-wrapper small active">
            <div class="spinner-layer spinner-blue">
              <div class="circle-clipper left">
                <div class="circle"></div>
              </div>
              <div class="gap-patch">
                <div class="circle"></div>
              </div>
              <div class="circle-clipper right">
                <div class="circle"></div>
              </div>
            </div>

            <div class="spinner-layer spinner-red">
              <div class="circle-clipper left">
                <div class="circle"></div>
              </div>
              <div class="gap-patch">
                <div class="circle"></div>
              </div>
              <div class="circle-clipper right">
                <div class="circle"></div>
              </div>
            </div>

            <div class="spinner-layer spinner-yellow">
              <div class="circle-clipper left">
                <div class="circle"></div>
              </div>
              <div class="gap-patch">
                <div class="circle"></div>
              </div>
              <div class="circle-clipper right">
                <div class="circle"></div>
              </div>
            </div>

            <div class="spinner-layer spinner-green">
              <div class="circle-clipper left">
                <div class="circle"></div>
              </div>
              <div class="gap-patch">
                <div class="circle"></div>
              </div>
              <div class="circle-clipper right">
                <div class="circle"></div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div v-if="importStep == 3" class="modal-footer">
        <a
          @click="reloadPage"
          class="modal-close waves-effect waves-green btn-flat"
          >{{ $t('close') }}</a
        >
      </div>
    </div>
    <div id="modal_preview_file" class="modal modal-fixed-footer">
      <div class="modal-content">
        <h4 id="modal_preview_file_titletext">
          {{ $t('modal-preview-file-header') }}
        </h4>
        <p id="modal_preview_file_zoomtext">
          {{ $t('modal-preview-file-click-to-zoom') }}
        </p>
        <p>
          <img
            class="responsive-img materialboxed"
            id="modal_preview_file_img"
            :src="fileToDownload.href"
          />
        </p>
      </div>
      <div class="modal-footer">
        <a
          class="modal-close waves-effect waves-red btn-flat"
          @click="downloadFile()"
          >{{ $t('download') }}</a
        >
        <a class="modal-close waves-effect waves-green btn-flat">{{
          $t('close')
        }}</a>
      </div>
    </div>
    <div id="modal_update_available" class="modal modal-fixed-footer">
      <div class="modal-content">
        <h4>
          {{ $t('modal-update-available-header') }}
        </h4>
        <p class="version_modal_text">
          {{ $t('update-available-client-version') }}:
          <b>{{ clientVersion }}</b>
        </p>
        <p class="version_modal_text">
          {{ $t('update-available-recent-version') }}:
          <b>{{ recentDailytxtVersion }}</b>
        </p>
      </div>
      <div class="modal-footer">
        <a class="modal-close waves-effect waves-green btn-flat">{{
          $t('close')
        }}</a>
      </div>
    </div>
    <div id="modal_history" class="modal">
      <div class="modal-content">
        <h4>{{ $t('modal-history-title') }}</h4>
        <div>
          <div class="row">
            <div class="col s12">
              <ul class="historyTabs">
                <li
                  class="historyTab"
                  v-for="(entry, index) in versionHistory"
                  :key="index"
                >
                  <a
                    @click.prevent="setHistoryActive(entry.version)"
                    :class="{ active: isHistoryActive(entry.version) }"
                    >{{ entry.date_written }}</a
                  >
                </li>
              </ul>
            </div>
            <div class="col s12">
              <textarea
                disabled
                class="historyText"
                name=""
                id=""
                cols="30"
                rows="10"
                :value="selectedHistoryText"
              ></textarea>
            </div>
            <div class="col s12"></div>
          </div>
        </div>
      </div>
      <div class="modal-footer">
        <span>{{ $t('use-selected-history-as-default') }}</span>
        <a class="modal-close waves-effect waves-red btn-flat">{{
          $t('abort')
        }}</a>
        <a
          class="modal-close waves-effect waves-green btn-flat"
          @click="useHistoryVersion()"
          >{{ $t('yes') }}</a
        >
      </div>
    </div>
    <div id="modal_change_password" class="modal">
      <div class="modal-content">
        <h4>{{ $t('modal-change-password-header') }}</h4>
        <div class="container">
          <div class="input-container">
            <div class="input-field">
              <input id="old_password" type="password" v-model="old_password" />
              <label for="old_password">{{ $t('old-password') }}</label>
            </div>
          </div>
        </div>
        <div class="divider input-divider"></div>
        <div class="container">
          <div class="input-container">
            <div class="input-field">
              <input
                id="new_password1"
                type="password"
                v-model="new_password1"
              />
              <label for="new_password1">{{ $t('new-password') }}</label>
            </div>
            <div class="input-field">
              <input
                id="new_password2"
                type="password"
                v-model="new_password2"
              />
              <label for="new_password2">{{
                $t('confirm-new-password')
              }}</label>
            </div>
            <div
              class="alert-password"
              v-if="
                (new_password1 != '' || new_password2 != '') &&
                  new_password1 != new_password2
              "
            >
              {{ $t('new-password-does-not-match') }}
            </div>
          </div>
        </div>
      </div>
      <div class="modal-footer">
        <a class="modal-close waves-effect waves-red btn-flat">{{
          $t('abort')
        }}</a>
        <a
          class="modal-close waves-effect waves-green btn-flat"
          @click="changePassword()"
          :class="{
            disabled:
              old_password == '' ||
              new_password1 == '' ||
              new_password1 != new_password2
          }"
          >{{ $t('save') }}</a
        >
      </div>
    </div>
    <ul id="slide-out" class="sidenav sidenav-fixed">
      <div class="calendar-box section">
        <vc-date-picker
          :value="new Date()"
          v-model="dateSelected"
          :attributes="datesWithLogs"
          ref="calendar"
          @update:from-page="getDaysWithLogsTrigger"
          :locale="$t('calendar-locale')"
          ><template v-slot:footer>
            <div class="calendar-footer">
              <a
                class="waves-effect waves-light btn todayBtn"
                @click="moveToToday"
              >
                {{ $t('today') }}
              </a>
            </div>
          </template>
        </vc-date-picker>
      </div>
      <div class="divider"></div>
      <div class="searchArea section">
        <div class="row valign-wrapper searchHeader">
          <div class="input-field col s10">
            <input
              id="search"
              type="text"
              @keypress.enter="search"
              :value="searchString"
              @input="e => (searchString = e.target.value)"
            />
            <label for="search">{{ $t('search-label') }}</label>
          </div>
          <div class="col s2 searchBtn">
            <a
              class="btn-floating waves-effect waves-light deep-orange lighten-1"
              @click="search"
              ><i class="material-icons">search</i></a
            >
          </div>
        </div>
        <div class="row searchResults">
          <div class="collection" v-if="searchResults.length > 0">
            <a
              class="collection-item searchResultRow"
              :class="{ searchResultSelected: index == searchResultSelected }"
              v-for="(searchResult, index) in searchResultsSorted"
              :key="index"
              @click="
                selectDay(
                  searchResult.year,
                  searchResult.month,
                  searchResult.day
                )
                searchResultSelected = index
              "
              ><div class="col s3 searchResultLeft">
                {{
                  $t('full-date', {
                    day: searchResult.day,
                    month: searchResult.month,
                    year: searchResult.year
                  })
                }}
              </div>
              <div class="col s9 searchResultRight">
                {{ searchResult.snippetStart
                }}<b>{{ searchResult.snippetBold }}</b
                >{{ searchResult.snippetEnd }}
              </div></a
            >
          </div>
        </div>
      </div>
    </ul>
    <div class="right-main">
      <div class="row main-header-row">
        <div class="col s5 m4 l3" id="left">
          <div class="dateDescription">
            <span>{{ dateDescription.split(',')[0] }}</span>
            <hr />
            <span>{{ dateDescription.split(',')[1] }}</span>
          </div>
        </div>
        <div class="col s5 m4 l3" id="right">
          <transition name="fade-only-opacity">
            <div class="dateWritten" v-if="!isLoading && dateWritten != ''">
              <span>{{ $t('last-edited') }}</span>
              <hr />
              <span>{{ dateWritten }}</span>
            </div>
          </transition>
        </div>
        <div class="col s2 m4 l3" id="loading" v-if="isLoading">
          <div class="preloader-wrapper small active">
            <div class="spinner-layer spinner-blue">
              <div class="circle-clipper left">
                <div class="circle"></div>
              </div>
              <div class="gap-patch">
                <div class="circle"></div>
              </div>
              <div class="circle-clipper right">
                <div class="circle"></div>
              </div>
            </div>

            <div class="spinner-layer spinner-red">
              <div class="circle-clipper left">
                <div class="circle"></div>
              </div>
              <div class="gap-patch">
                <div class="circle"></div>
              </div>
              <div class="circle-clipper right">
                <div class="circle"></div>
              </div>
            </div>

            <div class="spinner-layer spinner-yellow">
              <div class="circle-clipper left">
                <div class="circle"></div>
              </div>
              <div class="gap-patch">
                <div class="circle"></div>
              </div>
              <div class="circle-clipper right">
                <div class="circle"></div>
              </div>
            </div>

            <div class="spinner-layer spinner-green">
              <div class="circle-clipper left">
                <div class="circle"></div>
              </div>
              <div class="gap-patch">
                <div class="circle"></div>
              </div>
              <div class="circle-clipper right">
                <div class="circle"></div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="row">
        <div class="col s12 m9">
          <textarea
            :value="logText"
            @input="e => (logText = e.target.value)"
            name="main-text"
            cols="30"
            rows="10"
            class="main-textarea"
            v-bind:class="{ saved: isSaved }"
          >
          </textarea>
        </div>
        <div class="col s12 m3">
          <div class="uploadArea">
            <label class="uploadBtn" for="fileUpload"
              ><i class="material-icons center-align">cloud_upload</i></label
            >
            <input
              type="file"
              id="fileUpload"
              name="fileUpload"
              @change="uploadFilesBtn"
              multiple
            />
            <div
              class="collection"
              v-if="fileUploadProgressesActive.length > 0"
            >
              <li
                class="collection-item uploadProgress"
                v-for="(progress, index) in fileUploadProgressesActive"
                :key="index"
              >
                <div class="progress">
                  <div
                    class="determinate"
                    :style="{ width: progress + '%' }"
                  ></div>
                </div>
              </li>
            </div>
            <div class="collection" v-if="files.length > 0">
              <li
                class="collection-item fileList valign-wrapper"
                v-for="file in files"
                :key="file.uuid"
              >
                <a
                  class="file tooltipped"
                  data-position="left"
                  :data-tooltip="file.filename"
                  @click="downloadFileModal(file.uuid)"
                >
                  {{ file.filename }}
                </a>
                <a
                  class="fileDelete valign-wrapper"
                  @click="deleteFileModal(file.uuid)"
                  ><i class="material-icons">delete</i></a
                >
              </li>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import $ from 'jquery'
import M from 'materialize-css'
import VCalendar from 'v-calendar'
import Vue from 'vue'
import UserService from '../services/user.service.js'
import _ from 'lodash'
import { eventBus } from '../main.js'
import { version } from '../../package'

Vue.use(VCalendar, {
  componentPrefix: 'vc'
})

export default {
  name: 'DailyTxT',
  data() {
    return {
      dateSelected: new Date(),
      lastDateSelected: new Date(),
      dateWritten: '',
      logText: '',
      savedLogText: '',
      searchString: '',
      searchResults: [],
      datesWithLogsRaw: [],
      datesWithFilesRaw: [],
      yearShown: new Date(),
      monthShown: new Date(),
      isLoading: false,
      fileUploadProgresses: [],
      files: [],
      fileToDelete: {},
      fileToDownload: {},
      old_password: '',
      new_password1: '',
      new_password2: '',
      lastPage: {},
      dragging: false,
      searchResultSelected: null,
      isExporting: false,
      versionHistory: [],
      selectedHistoryText: '',
      selectedHistoryVersion: 0,
      recentDailytxtVersion: version,
      clientVersion: version,
      importProgress: 0,
      importStep: 0
    }
  },
  updated: function() {
    this.$nextTick(function() {
      var elems = document.querySelectorAll('.tooltipped')
      M.Tooltip.init(elems, {})
    })
  },
  computed: {
    fileUploadProgressesActive: function() {
      return this.fileUploadProgresses.filter(i => i !== 100)
    },
    datesWithLogs: function() {
      var datesLogs = this.datesWithLogsRaw.map(o => {
        return new Date(this.yearShown, this.monthShown - 1, o)
      })
      var datesFiles = this.datesWithFilesRaw.map(o => {
        return new Date(this.yearShown, this.monthShown - 1, o)
      })

      return [
        {
          highlight: 'green',
          dates: datesLogs
        },
        {
          dot: 'red',
          dates: datesFiles
        }
      ]
    },
    dateDescription: function() {
      return this.dateSelected.toLocaleDateString([], {
        weekday: 'long',
        year: 'numeric',
        month: 'long',
        day: 'numeric'
      })
    },
    searchResultsSorted: function() {
      function compare(a, b) {
        if (
          a.year < b.year ||
          (a.year == b.year && a.month < b.month) ||
          (a.year == b.year && a.month == b.month && a.day < b.day)
        ) {
          return -1
        } else {
          return 1
        }
      }
      var arr = this.searchResults

      return arr.sort(compare)
    },
    isSaved: function() {
      return this.logText == this.savedLogText
    }
  },
  beforeMount() {
    UserService.getRecentVersion(version).then(
      response => {
        if (response.data.recent_version != version) {
          this.$root.$emit('dailytxt_version_update', {
            update_available:
              response.data.recent_version != version ? true : false
          })
          this.recentDailytxtVersion = response.data.recent_version
        }
      },
      error => {
        console.log(error.response.data.message)
      }
    )
  },
  mounted() {
    $(document).ready(function() {
      var sidenav = document.querySelectorAll('.sidenav')[0]
      M.Sidenav.init(sidenav, {})

      var modals = document.querySelectorAll('.modal')
      M.Modal.init(modals, {})

      var dropdown = document.querySelector('.dropdown-trigger')
      M.Dropdown.init(dropdown, { coverTrigger: false, constrainWidth: false })
    })

    this.daySelected()

    $(document).keydown(event => {
      if (event.altKey && event.key == 'ArrowLeft') {
        event.preventDefault()
        this.dateSelected = new Date(
          this.dateSelected.getFullYear(),
          this.dateSelected.getMonth(),
          this.dateSelected.getDate() - 1
        )
      } else if (event.altKey && event.key == 'ArrowRight') {
        event.preventDefault()
        this.dateSelected = new Date(
          this.dateSelected.getFullYear(),
          this.dateSelected.getMonth(),
          this.dateSelected.getDate() + 1
        )
      } else if (event.ctrlKey && event.key == 'f') {
        event.preventDefault()
        $('#search').focus()
      }
    })

    window.addEventListener('beforeunload', e => {
      if (!this.isSaved) {
        e.preventDefault()
        this.toastAlert(this.$t('not-yet-saved'))
      }
    })
  },
  watch: {
    dateSelected: function() {
      this.daySelected()
      this.$refs.calendar.$children[0].move(this.dateSelected)
    },
    logText: function() {
      this.debouncedAutoSave()
    }
  },
  created: function() {
    this.debouncedAutoSave = _.debounce(function() {
      this.autoSave(this.dateSelected)
    }, 1000)
    eventBus.$off('changePassword')
    eventBus.$on('changePassword', () => {
      this.changePasswordModal()
    })
    eventBus.$off('exportData')
    eventBus.$on('exportData', () => {
      this.exportData()
    })
    eventBus.$off('importData')
    eventBus.$on('importData', e => {
      this.importData(e)
    })
    eventBus.$off('historyModal')
    eventBus.$on('historyModal', () => {
      this.historyModal()
    })
    eventBus.$off('updateModal')
    eventBus.$on('updateModal', () => {
      this.updateModal()
    })
  },
  methods: {
    reloadPage() {
      window.location.reload()
    },
    setHistoryActive(version) {
      this.selectedHistoryVersion = version
      var h
      for (h of this.versionHistory) {
        if (h.version == version) {
          this.selectedHistoryText = h.text
          break
        }
      }
    },
    isHistoryActive(version) {
      return this.selectedHistoryVersion == version
    },
    toastSuccess(message) {
      M.toast({ html: message, classes: 'rounded green' })
    },
    toastAlert(message) {
      M.toast({ html: message, classes: 'rounded red' })
    },
    getDaysWithLogsTrigger(page) {
      if (
        this.lastPage.month != page.month ||
        this.lastPage.year != page.year
      ) {
        this.datesWithLogsRaw = []
        this.datesWithFilesRaw = []
        this.getDaysWithLogs(page)
      }
      this.lastPage = page
    },
    downloadFileModal(uuid) {
      this.isLoading = true
      UserService.downloadFile(uuid).then(
        response => {
          let blob = new Blob([response.data])
          this.isLoading = false
          var href = window.URL.createObjectURL(blob)
          this.fileToDownload = this.files.find(f => f.uuid == uuid)
          this.fileToDownload.href = href
          if (this.fileToDownload.filename.match(/\.(jpg|jpeg|png|gif)$/)) {
            var modal = document.querySelector('#modal_preview_file')
            M.Modal.getInstance(modal).open()
          } else {
            this.downloadFile()
          }
        },
        error => {
          this.isLoading = false
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    downloadFile() {
      this.loading = true
      UserService.downloadFile(this.fileToDownload.uuid).then(
        response => {
          let blob = new Blob([response.data])
          this.loading = false
          let link = document.createElement('a')
          link.href = window.URL.createObjectURL(blob)
          link.download = this.fileToDownload.filename
          link.click()
        },
        error => {
          this.loading = false
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    uploadFile(f) {
      this.isLoading = true

      var myProgress = this.fileUploadProgresses.length
      this.fileUploadProgresses.push(0)
      UserService.uploadFile(f, this.dateSelected, event => {
        Vue.set(
          this.fileUploadProgresses,
          myProgress,
          Math.round((100 * event.loaded) / event.total)
        )
      }).then(
        response => {
          this.isLoading = false
          if (response.data.success) {
            this.files.push({
              filename: f.name,
              uuid: response.data.uuid_filename
            })
            this.getDaysWithLogs({
              month: this.monthShown,
              year: this.yearShown
            })
          }
        },
        error => {
          this.isLoading = false
          if (typeof error.response.data.message !== 'undefined') {
            console.log(error.response.data.message)
            this.toastAlert(error.response.data.message)
          } else {
            console.log(error.response.data)
            this.toastAlert(this.$t('error-uploading-file'))
          }
        }
      )
    },
    deleteFileModal(uuid) {
      this.fileToDelete = this.files.find(f => f.uuid == uuid)
      var modal = document.querySelector('#modal_delete_file')
      M.Modal.getInstance(modal).open()
    },
    deleteFile() {
      UserService.deleteFile(this.fileToDelete.uuid, this.dateSelected).then(
        response => {
          if (response.data.success) {
            this.files = this.files.filter(obj => {
              return obj.uuid !== this.fileToDelete.uuid
            })
            this.getDaysWithLogs({
              month: this.monthShown,
              year: this.yearShown
            })
          }
        },
        error => {
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    historyModal() {
      this.isLoading = true
      UserService.getHistory(this.dateSelected).then(
        response => {
          this.isLoading = false
          if (response.data.success) {
            this.versionHistory = response.data.history
            var modal = document.querySelector('#modal_history')
            M.Modal.getInstance(modal).open()
            var h
            var maxversion = 0
            for (h of this.versionHistory) {
              if (h.version > maxversion) maxversion = h.version
            }
            this.setHistoryActive(maxversion)
            var tabs = document.querySelector('.historyTabs')
            tabs.scrollLeft = tabs.scrollWidth
          } else {
            console.log(this.$t('no-history-available-yet'))
            this.toastAlert(this.$t('no-history-available-yet'))
          }
        },
        error => {
          this.isLoading = false
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    useHistoryVersion() {
      UserService.useHistoryVersion(
        this.selectedHistoryVersion,
        this.dateSelected
      ).then(
        response => {
          if (response.data.success) {
            this.daySelected()
          } else {
            console.log(response.data.message)
            this.toastAlert(response.data.message)
          }
        },
        error => {
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    changePasswordModal() {
      var modal = document.querySelector('#modal_change_password')
      M.Modal.getInstance(modal).open()
    },
    updateModal() {
      var modal = document.querySelector('#modal_update_available')
      M.Modal.getInstance(modal).open()
    },
    changePassword() {
      UserService.changePassword(this.old_password, this.new_password1).then(
        response => {
          if (response.data.success) {
            if (response.data.token) {
              localStorage.setItem('user', JSON.stringify(response.data))
              this.toastSuccess(this.$t('password-change-successful'))
            }
          } else {
            console.log(response.data.message)
            this.toastAlert(response.data.message)
          }
        },
        error => {
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    uploadFilesBtn(event) {
      Array.prototype.forEach.call(event.target.files, f => {
        this.uploadFile(f)
      })
    },
    uploadFilesDrop(event) {
      event.preventDefault()
      this.dragging = false
      Array.prototype.forEach.call(event.dataTransfer.files, f => {
        if (!f.type && f.size % 4096 == 0) {
          this.toastAlert(this.$t('no-valid-file'))
        } else {
          this.uploadFile(f)
        }
      })
    },
    async selectDay(year, month, day) {
      this.dateSelected = new Date(year, (parseInt(month) - 1).toString(), day)
    },
    search() {
      var s = this.searchString.trim()
      if (s.length < 3) {
        this.toastAlert(this.$t('search-string-too-short'))
        return
      }

      this.searchResults = []
      UserService.search(s).then(
        data => {
          this.searchResults = data.data.results
          this.searchResultSelected = null
        },
        error => {
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    getDaysWithLogs(page) {
      this.yearShown = page.year
      this.monthShown = page.month
      UserService.getDaysWithLogs(page).then(
        dates => {
          this.datesWithLogsRaw = dates.data.logs
          this.datesWithFilesRaw = dates.data.files
        },
        error => {
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
          this.$parent.$parent.transitionName = 'slideRight'
          this.$store.dispatch('auth/logout')
          this.$router.push('/login')
        }
      )
    },
    async autoSave(date) {
      if (this.savedLogText == this.logText) {
        return
      }

      var toSave = this.logText

      var now = new Date()
      var date_written = now.toLocaleString([], {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      })

      await UserService.saveLog(toSave, date, date_written).then(
        () => {
          this.savedLogText = toSave
          this.dateWritten = date_written
          if (!this.datesWithLogsRaw.includes(date.getDate())) {
            this.getDaysWithLogs({
              year: this.yearShown,
              month: this.monthShown
            })
          }
        },
        error => {
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    moveToToday() {
      this.dateSelected = new Date()
    },
    async daySelected() {
      this.debouncedAutoSave.cancel()

      if ($(window).width() <= 992) {
        var sidenav_el = document.querySelector('.sidenav')
        var instance = M.Sidenav.getInstance(sidenav_el)
        if (instance !== undefined) {
          if (instance.isOpen) {
            instance.close()
          }
        }
      }

      if (!this.isSaved) {
        await this.autoSave(this.lastDateSelected)
      }

      this.lastDateSelected = this.dateSelected

      this.isLoading = true

      var loadingDay = this.dateSelected

      UserService.loadDay(this.dateSelected).then(
        response => {
          if (loadingDay != this.dateSelected) {
            return
          }
          this.isLoading = false
          if (response.data.enc_error) {
            this.toastAlert(this.$t('encryption-error-try-relogin'))
          }
          this.logText = response.data.text
          this.savedLogText = this.logText
          this.dateWritten = response.data.date_written
          this.files = response.data.files
          this.$store.commit(
            'setHistoryAvailable',
            response.data.historyAvailable
          )
          if ($(window).width() > 992) {
            document.querySelector('.main-textarea').focus()
          }
        },
        error => {
          this.isLoading = false
          this.toastAlert(error.response.data.message)
          this.$parent.$parent.transitionName = 'slideRight'
          this.$store.dispatch('auth/logout')
          this.$router.push('/login')
        }
      )
    },
    exportData() {
      if (this.isExporting) {
        return
      }
      this.isExporting = true
      this.isLoading = true

      this.toastSuccess(this.$t('export-started-1'))
      this.toastSuccess(this.$t('export-started-2'))

      UserService.exportData().then(
        response => {
          this.isLoading = false
          this.isExporting = false
          let blob = new Blob([response.data], { type: 'application/zip' })
          let link = document.createElement('a')
          link.href = window.URL.createObjectURL(blob)
          var now = new Date()
          var filename =
            'DailyTxT_Export_' +
            now.getFullYear() +
            '-' +
            now.getMonth() +
            '-' +
            now.getDate() +
            '_' +
            now.getHours() +
            '-' +
            now.getMinutes() +
            '-' +
            now.getSeconds() +
            '.zip'

          link.download = filename
          link.click()
        },
        error => {
          this.isLoading = false
          this.isExporting = false
          console.log(error)
        }
      )
    },
    importData(event) {
      var f = event.target.files[0]

      var modal = document.querySelector('#modal_import_data')
      M.Modal.init(modal, { dismissible: false })
      M.Modal.getInstance(modal).open()
      this.importStep = 1

      UserService.importData(f, event => {
        this.importProgress = Math.round((100 * event.loaded) / event.total)

        if (event.loaded == event.total) {
          this.importStep = 2
        }
      }).then(
        response => {
          if (response.data.success) {
            this.importStep = 3
            this.toastSuccess(this.$t('import-successful'))
          }
        },
        error => {
          if (typeof error.response.data.message !== 'undefined') {
            console.log(error.response.data.message)
            this.toastAlert(error.response.data.message)
            this.importStep = 3
          } else {
            console.log(error.response.data)
            this.toastAlert(this.$t('error-uploading-file'))
            this.importStep = 3
          }
        }
      )
    }
  }
}
</script>

<style>
span.vc-day-content.vc-focusable:focus {
  background-color: #03a9f4 !important;
}

span.vc-day-content.vc-focusable {
  transition: ease 0.3s;
}

span.vc-day-content.vc-focusable:hover {
  background-color: #29b6f6;
}

#toast-container {
  top: auto !important;
  left: auto !important;
  bottom: 10%;
  right: 7%;
}

body {
  background-color: #eeeeee;
}

.sidenav-overlay {
  opacity: 0 !important;
}

@media only screen and (max-width: 992px) {
  #toast-container {
    top: auto !important;
    left: auto !important;
    bottom: 10%;
    right: auto !important;
  }
}
</style>

<style scoped>
.upload-icons {
  vertical-align: middle;
}

.version_modal_text {
  font-size: 17px;
  text-align: left;
}

#modal_preview_file_titletext {
  margin-bottom: 0;
}

#modal_preview_file_zoomtext {
  margin-top: 0;
}

.historyTabs {
  overflow-x: auto;
  white-space: nowrap;
  border: 1px solid lightgray;
}

.historyTab {
  border-left: 1px solid grey;
  padding: 5px 0;
  margin: auto;
  display: inline-block;
  transition: ease 0.3s;
}

.historyTab:nth-child(1) {
  border-left: none;
}

.historyTab:hover {
  background: #e0e0e0;
}

.historyTab > a {
  cursor: pointer;
  color: #424242;
  text-decoration-color: #f57c00;
  transition: ease 0.3s;
  padding: 10px;
}

.historyTab > a.active {
  color: #1565c0;
  text-decoration: underline;
  text-decoration-thickness: 2px;
  text-decoration-color: #f57c00;
}

.historyText {
  border: 1px solid grey;
  box-shadow: none;
  color: #616161;
  resize: vertical;
  min-height: 200px;
}

textarea {
  font-size: 16px;
}

.main-header-row {
  margin-bottom: 0 !important;
  height: 50px;
}

.main-header-row > .col {
  height: 100%;
}

.main-header-row > .col#left {
  padding-right: 0;
}

.main-header-row > .col#right {
  padding-left: 0;
  transition: opacity ease 0.5s;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.1s;
}

.fade-enter,
.fade-leave.to {
  opacity: 0;
}

.fade-only-opacity-enter-active,
.fade-only-opacity-leave-active {
  transition: opacity 0.1s;
}

.fade-only-opacity-enter,
.fade-only-opacity-leave.to {
  opacity: 0;
}

.main-header-row > .col.s3#loading {
  display: flex;
  align-items: center;
  justify-content: center;
}

.dateDescription {
  background-color: #9ccc65;
  border-top: 1px solid #4caf50;
  border-left: 1px solid #4caf50;
  border-right: 1px solid #4caf50;
  border-top-left-radius: 10px;
  height: 100%;
  display: flex;
  align-items: center;
  flex-flow: wrap;
  padding: 5px;
}

.dateDescription > hr {
  width: 100%;
  border: none;
  margin: 0;
}

.dateWritten {
  background-color: #aed581;
  border-top: 1px solid #4caf50;
  border-right: 1px solid #4caf50;
  border-top-right-radius: 10px;
  height: 100%;
  display: flex;
  align-items: center;
  flex-flow: wrap;
  padding: 5px;
}

.dateWritten > hr {
  width: 100%;
  border: none;
  margin: 0;
}

.dropzone {
  position: fixed;
  top: 0;
  left: 0;
  z-index: 99999;
  background-color: rgba(0, 0, 0, 0.6);
  height: 100vh;
  width: 100vw;
  border: 30px dashed #424242;
  color: white;
  font-size: 10vw;
}

.modal-close {
  cursor: pointer;
}

.input-container {
  padding: 0 1rem;
  border-radius: 10px;
  border: 1px solid #e0e0e0;
}

.input-divider {
  height: 0;
}

.alert-password {
  color: #f44336;
  border: 1px solid #f44336;
  border-radius: 10px;
}

h3 {
  margin: 40px 0 0;
}
ul {
  list-style-type: none;
  padding: 0;
}
li {
  display: inline-block;
  margin: 0 10px;
}

.sidenav {
  min-width: 400px;
  max-width: 550px;
  width: 35%;
  height: calc(100vh - 64px);
  top: inherit;
  box-shadow: none;
  flex-flow: column;
  display: flex;
  background-color: #eeeeee;
}

.divider {
  width: 90%;
  margin: 20px 5% 20px 5% !important;
}

.searchArea {
  width: 80%;
  margin: auto;
  display: flex;
  flex-flow: column;
  flex: 1;
  min-height: 400px;
}

/* label focus color */
.input-field input[type='text']:focus + label,
input[type='password']:focus + label {
  color: #ff7043 !important;
}

/* label underline focus color */
.input-field input[type='text']:focus,
input[type='password']:focus {
  border-bottom: 1px solid #ff7043 !important;
  box-shadow: 0 1px 0 0 #ff7043 !important;
}

.searchHeader {
  margin-left: 0px !important;
  margin-right: 0px !important;
}

.searchBtn {
  margin: auto;
}

.searchResults {
  width: 100%;
  flex: 1;
  overflow-y: auto;
  box-shadow: 0px 0px 4px 1px #ff7043;
}

.searchResults .collection {
  margin: 1px 0px;
}

.searchResultSelected {
  background-color: #42a5f5 !important;
}

.searchResultRow:not(.searchResultSelected):hover {
  background-color: #90caf9 !important;
}

.searchResultRow {
  padding: 0 !important;
  min-height: 40px;
  display: flex !important;
  justify-content: center;
  align-items: center;
  cursor: pointer;
}

.searchResultLeft {
  min-width: 100px;
  font-size: 1.1rem;
  color: #212121;
}

.searchResultRight {
  border-left: 1px solid grey;
  color: #212121;
}

.right-main {
  height: 100%;
  margin-left: min(max(35%, 400px), 550px);
  padding: 10px;
}

.main-textarea {
  width: 100%;
  float: left;
  height: 630px;
  resize: vertical;
  background-color: #f5f5f5;
  transition: box-shadow ease 0.3s;
  margin-bottom: 20px;
  padding: 2px;
  border-bottom-left-radius: 10px;
  border-top-right-radius: 10px;
}

.main-textarea:not(.saved) {
  border: 1px solid #ff9800;
}

.main-textarea:not(.saved):focus {
  outline: none;
  box-shadow: 0px 0px 0px 1px #ff9800;
}

.main-textarea.saved {
  border: 1px solid #4caf50 !important;
}

.main-textarea.saved:focus {
  outline: none;
  box-shadow: 0px 0px 0px 1px #4caf50 !important;
}

.uploadArea {
  position: relative;
  width: 80%;
  left: 10%;
}

.uploadBtn {
  position: relative;
  background-color: #e0e0e0;
  display: flex;
  justify-content: center;
  align-items: center;
  cursor: pointer;
  border-radius: 4px;
  transition: ease 0.3s;
}

.uploadBtn:hover {
  box-shadow: 0 0 5px 5px #bdbdbd;
  background-color: #eeeeee;
}

.uploadBtn:before {
  content: '';
  display: block;
  padding-top: 50%;
}

.uploadBtn i {
  font-size: 40px;
  transition: ease 0.3s;
}

.uploadBtn:hover i {
  font-size: 60px;
}

#fileUpload {
  display: none;
}

.uploadProgress {
  display: list-item;
  list-style: none;
}

.importProgress {
  display: list-item;
  list-style: none;
}

.fileList {
  display: flex;
  padding: 5px 10px !important;
}

.file {
  color: #424242;
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fileDelete {
  color: #e57373;
  margin-left: auto;
  cursor: pointer;
  opacity: 0.4;
  border: 1px solid #e0e0e0;
  border-radius: 4px;
  transition: ease 0.3s;
}

.fileDelete:hover {
  opacity: 1;
  border: 1px solid #bdbdbd;
}

.vc-container {
  margin: auto;
  width: 100%;
}

.calendar-box {
  width: 80%;
  margin: auto;
}

.todayBtn {
  margin: 10px;
  border-radius: 5px;
  background-color: #2196f3;
  text-transform: none;
}

.todayBtn:hover {
  background-color: #64b5f6;
}

.calendar-footer {
  border-top: 1px solid #e0e0e0;
}

.main {
  margin-top: 20px;
  height: calc(100% - 64px - 20px);
}

@media only screen and (min-width: 1600px) {
  .col#left,
  .col#right {
    width: 16.7%;
    margin-left: auto;
  }
}

@media only screen and (max-width: 600px) {
  .dateWritten {
    font-size: 12px;
  }

  .dateDescription {
    font-size: 12px;
  }
}

@media only screen and (max-width: 992px) {
  .right-main {
    margin-left: 0;
  }

  .sidenav {
    min-width: 0px;
    max-width: 100%;
    width: 100%;
  }
}
</style>
