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
    <div id="modal_remove_day" class="modal">
      <div class="modal-content">
        <h4>{{ $t('modal-remove-day-header') }}</h4>
        <p>
          <i18n path="modal-remove-day-text">
            <b>{{ dateDescription.split(',')[1] }}</b>
          </i18n>
        </p>
      </div>
      <div class="modal-footer">
        <a class="modal-close waves-effect waves-red btn-flat">{{
          $t('abort')
        }}</a>
        <a
          class="modal-close waves-effect waves-green btn-flat"
          @click="removeDay()"
          >{{ $t('delete') }}</a
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
            <div class="calendar-footer row">
              <div class="col s3 l3 xl2"></div>
              <div class="col s6 l6 xl8">
                <a
                  class="waves-effect waves-light btn todayBtn"
                  @click="moveToToday"
                >
                  {{ $t('today') }}
                </a>
              </div>
              <div class="col s3 l3 xl2" style="padding: 0 !important">
                <a
                  v-if="!dateIsBookmarked"
                  class="waves-effect waves-light btn bookmarkBtn tooltipped"
                  @click="addBookmark"
                  :data-tooltip="$t('tooltip-add-bookmark')"
                >
                  <i class="material-icons">bookmark_add</i>
                </a>
                <a
                  v-else
                  class="waves-effect waves-light btn bookmarkBtn tooltipped"
                  @click="removeBookmark"
                  :data-tooltip="$t('tooltip-remove-bookmark')"
                >
                  <i class="material-icons">bookmark_remove</i>
                </a>
              </div>
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
              @input="(e) => searchInputUpdated(e)"
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
        <div
          class="row searchResults"
          :class="{ searchResultsPulse: searchResultsAttention }"
          @animationend="searchResultsAttention = false"
        >
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
            >
              <div class="col s3 searchResultLeft">
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
              </div>
            </a>
          </div>
        </div>
      </div>
    </ul>
    <div class="right-main">
      <div class="row">
        <div class="col s12 hide-on-xlarge">
          <div v-if="templates.length > 0">
            <a class="dropdown-trigger btn left" data-target="dropdown1"
              >{{ $t('select-template') }}
              <i class="material-icons right">arrow_drop_down</i>
            </a>
            <ul id="dropdown1" class="dropdown-content">
              <li v-for="t in templatesSorted" :key="t.number">
                <a @click="selectTemplate(t.number)">{{ t.name }}</a>
              </li>
            </ul>
          </div>
        </div>
      </div>
      <div class="row main-header-row">
        <div class="col s5 m3 l3 xl2" id="left">
          <div
            class="dateDescription"
            :class="{ dateDescriptionBookmarked: dateIsBookmarked }"
          >
            <span>{{ dateDescription.split(',')[0] }}</span>
            <hr />
            <span>{{ dateDescription.split(',')[1] }}</span>
          </div>
        </div>
        <div class="col s5 m3 l3 xl2" id="right">
          <transition name="fade-only-opacity">
            <div class="dateWritten" v-if="!isLoading && dateWritten != ''">
              <span>{{ $t('last-edited') }}</span>
              <hr />
              <span>{{ dateWritten }}</span>
            </div>
          </transition>
        </div>
        <div class="col s12 l12 xl4 hide-on-large-and-down">
          <div class="left" v-if="templates.length > 0">
            <a class="dropdown-trigger btn" data-target="dropdown2"
              >{{ $t('select-template') }}
              <i class="material-icons right">arrow_drop_down</i>
            </a>
            <ul id="dropdown2" class="dropdown-content">
              <li v-for="t in templatesSorted" :key="t.number">
                <a @click="selectTemplate(t.number)">{{ t.name }}</a>
              </li>
            </ul>
          </div>
        </div>
        <div class="col s2 m2 l2 xl1" id="loading" v-if="isLoading">
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
        <div
          class="col s2 m3 l3 xl1 valign-wrapper"
          id="removeDay"
          v-if="!isLoading && (dateWritten != '' || files.length != 0)"
        >
          <a
            class="removeDay valign-wrapper tooltipped"
            :data-tooltip="$t('remove-day')"
            @click="removeDayModal()"
            ><i class="material-icons">delete</i></a
          >
        </div>
      </div>
      <div class="row">
        <div class="col s12 m9">
          <textarea
            :value="logText"
            @input="(e) => (logText = e.target.value)"
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
      dateIsBookmarked: false,
      lastDateSelected: new Date(),
      dateWritten: '',
      logText: '',
      savedLogText: '',
      searchString: '',
      searchResults: [],
      datesWithBookmarkRaw: [],
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
      versionHistory: [],
      selectedHistoryText: '',
      selectedHistoryVersion: 0,
      recentDailytxtVersion: version,
      clientVersion: version,
      password: '',
      searchResultsAttention: false,
      templates: [],
      templatesReload: false
    }
  },
  updated: function () {
    this.$nextTick(function () {
      var elems = document.querySelectorAll('.tooltipped')
      M.Tooltip.init(elems, {})

      if (this.templatesReload) {
        elems = document.querySelectorAll('.dropdown-trigger')
        M.Dropdown.init(elems, {})
        this.templatesReload = false
      }
    })
  },
  computed: {
    templatesSorted: function () {
      function compare(a, b) {
        if (a.number < b.number) {
          return -1
        } else {
          return 1
        }
      }
      var arr = this.templates

      return arr.sort(compare)
    },
    fileUploadProgressesActive: function () {
      return this.fileUploadProgresses.filter((i) => i !== 100)
    },
    datesWithLogs: function () {
      var datesBookmark = this.datesWithBookmarkRaw.map((o) => {
        return new Date(this.yearShown, this.monthShown - 1, o)
      })
      var datesLogs = this.datesWithLogsRaw.map((o) => {
        return new Date(this.yearShown, this.monthShown - 1, o)
      })
      var datesFiles = this.datesWithFilesRaw.map((o) => {
        return new Date(this.yearShown, this.monthShown - 1, o)
      })

      //remove dates from datesLogs that are already in datesBookmark
      let datesLogsFiltered = datesLogs.filter(
        (o) => !this.datesWithBookmarkRaw.includes(o.getDate())
      )
      this.checkIfBookmarked()
      return [
        {
          highlight: 'orange',
          dates: datesBookmark
        },
        {
          highlight: 'green',
          dates: datesLogsFiltered
        },
        {
          dot: 'red',
          dates: datesFiles
        }
      ]
    },
    dateDescription: function () {
      return this.dateSelected.toLocaleDateString([], {
        weekday: 'long',
        year: 'numeric',
        month: 'long',
        day: 'numeric'
      })
    },
    searchResultsSorted: function () {
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
    isSaved: function () {
      return this.logText == this.savedLogText
    }
  },
  beforeMount() {
    UserService.getRecentVersion(version).then(
      (response) => {
        if (response.data.recent_version != version) {
          this.$root.$emit('dailytxt_version_update', {
            update_available:
              response.data.recent_version != version ? true : false
          })
          this.recentDailytxtVersion = response.data.recent_version
        }
      },
      (error) => {
        console.log(error.response.data.message)
      }
    )
  },
  mounted() {
    $(document).ready(function () {
      var sidenav = document.querySelectorAll('.sidenav')[0]
      M.Sidenav.init(sidenav, {})

      var modals = document.querySelectorAll('.modal')
      M.Modal.init(modals, {})
    })

    this.daySelected()

    this.loadTemplates()

    $(document).keydown((event) => {
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

    window.addEventListener('beforeunload', (e) => {
      if (!this.isSaved) {
        e.preventDefault()
        this.toastAlert(this.$t('not-yet-saved'))
      }
    })
  },
  watch: {
    dateSelected: function () {
      if (this.dateSelected == null) {
        this.dateSelected = this.lastDateSelected
        return
      }
      this.daySelected()
      this.$refs.calendar.$children[0].move(this.dateSelected)
    },
    logText: function () {
      this.debouncedAutoSave()
    }
  },
  created: function () {
    document.addEventListener('swUpdated', this.updateAvailable, { once: true })
    this.canCopy = !!navigator.clipboard
    this.debouncedAutoSave = _.debounce(function () {
      this.autoSave(this.dateSelected)
    }, 1000)
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
    addBookmark() {
      UserService.addBookmark(this.dateSelected).then(
        (response) => {
          if (response.data.success) {
            this.datesWithBookmarkRaw.push(this.dateSelected.getDate())
            eventBus.$emit('toastSuccess', this.$t('bookmark-added'))
          } else {
            console.log(response.data.message)
            eventBus.$emit('toastAlert', this.$t('bookmark-added-error'))
          }
          this.checkIfBookmarked()
        },
        (error) => {
          console.log(error.response.data.message)
          eventBus.$emit('toastAlert', this.$t('bookmark-added'))
        }
      )
    },
    removeBookmark() {
      UserService.removeBookmark(this.dateSelected).then(
        (response) => {
          if (response.data.success) {
            this.datesWithBookmarkRaw = this.datesWithBookmarkRaw.filter(
              (o) => o != this.dateSelected.getDate()
            )
            eventBus.$emit('toastSuccess', this.$t('bookmark-removed'))
          } else {
            console.log(response.data.message)
            eventBus.$emit('toastAlert', this.$t('bookmark-removed-error'))
          }
          this.checkIfBookmarked()
        },
        (error) => {
          console.log(error.response.data.message)
          eventBus.$emit('toastAlert', this.$t('bookmark-removed-error'))
        }
      )
    },
    searchInputUpdated(e) {
      this.searchString = e.target.value

      if (!this.searchString.length) {
        this.searchResults = []
      }
    },
    loadTemplates() {
      UserService.loadTemplates().then(
        (response) => {
          if (response.data.success) {
            this.templates = response.data.templates
            this.templatesReload = true
          } else {
            this.templates = []
            console.log(response.data.message)
            eventBus.$emit('toastAlert', response.data.message)
          }
        },
        (error) => {
          this.templates = []
          console.log(error.response.data.message)
          eventBus.$emit('toastAlert', error.response.data.message)
        }
      )
    },
    selectTemplate(i) {
      this.templates.forEach((t) => {
        if (t.number == i) {
          if (this.logText == '') this.logText = t.text
          else this.logText = this.logText + '\n' + t.text
        }
      })
    },
    updateAvailable() {
      this.toastSuccess(this.$t('update-installed-reload'))
    },
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
        this.datesWithBookmarkRaw = []
        this.datesWithLogsRaw = []
        this.datesWithFilesRaw = []
        this.getDaysWithLogs(page)
      }
      this.lastPage = page
    },
    downloadFileModal(uuid) {
      this.isLoading = true
      UserService.downloadFile(uuid).then(
        (response) => {
          let blob = new Blob([response.data])
          this.isLoading = false
          var href = window.URL.createObjectURL(blob)
          this.fileToDownload = this.files.find((f) => f.uuid == uuid)
          this.fileToDownload.href = href
          if (
            this.fileToDownload.filename
              .toLowerCase()
              .match(/\.(jpg|jpeg|png|gif)$/)
          ) {
            var modal = document.querySelector('#modal_preview_file')
            M.Modal.getInstance(modal).open()
          } else {
            this.downloadFile()
          }
        },
        (error) => {
          this.isLoading = false
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    downloadFile() {
      this.loading = true
      UserService.downloadFile(this.fileToDownload.uuid).then(
        (response) => {
          let blob = new Blob([response.data])
          this.loading = false
          let link = document.createElement('a')
          link.href = window.URL.createObjectURL(blob)
          link.download = this.fileToDownload.filename
          link.click()
        },
        (error) => {
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
      UserService.uploadFile(f, this.dateSelected, (event) => {
        Vue.set(
          this.fileUploadProgresses,
          myProgress,
          Math.round((100 * event.loaded) / event.total)
        )
      }).then(
        (response) => {
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
        (error) => {
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
      this.fileToDelete = this.files.find((f) => f.uuid == uuid)
      var modal = document.querySelector('#modal_delete_file')
      M.Modal.getInstance(modal).open()
    },
    deleteFile() {
      UserService.deleteFile(this.fileToDelete.uuid, this.dateSelected).then(
        (response) => {
          if (response.data.success) {
            this.files = this.files.filter((obj) => {
              return obj.uuid !== this.fileToDelete.uuid
            })
            this.getDaysWithLogs({
              month: this.monthShown,
              year: this.yearShown
            })
          }
        },
        (error) => {
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    removeDayModal() {
      var modal = document.querySelector('#modal_remove_day')
      M.Modal.getInstance(modal).open()
    },
    removeDay() {
      UserService.removeDay(this.dateSelected).then(
        (response) => {
          if (response.data.success) {
            this.getDaysWithLogs({
              month: this.monthShown,
              year: this.yearShown
            })
            this.daySelected()
          }
        },
        (error) => {
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    historyModal() {
      this.isLoading = true
      UserService.getHistory(this.dateSelected).then(
        (response) => {
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
        (error) => {
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
        (response) => {
          if (response.data.success) {
            this.daySelected()
          } else {
            console.log(response.data.message)
            this.toastAlert(response.data.message)
          }
        },
        (error) => {
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    updateModal() {
      var modal = document.querySelector('#modal_update_available')
      M.Modal.getInstance(modal).open()
    },
    async copy(s) {
      await navigator.clipboard.writeText(s)
      this.toastSuccess(this.$t('copy-to-clipboard-successful'))
    },
    uploadFilesBtn(event) {
      Array.prototype.forEach.call(event.target.files, (f) => {
        this.uploadFile(f)
      })
    },
    uploadFilesDrop(event) {
      event.preventDefault()
      this.dragging = false
      Array.prototype.forEach.call(event.dataTransfer.files, (f) => {
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
        (data) => {
          this.searchResults = data.data.results
          this.searchResultSelected = null
          this.searchResultsAttention = true
        },
        (error) => {
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    getDaysWithLogs(page) {
      this.yearShown = page.year
      this.monthShown = page.month
      UserService.getDaysWithLogs(page).then(
        (dates) => {
          this.datesWithLogsRaw = dates.data.logs
          this.datesWithFilesRaw = dates.data.files
          this.datesWithBookmarkRaw = dates.data.bookmarks
        },
        (error) => {
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
        (error) => {
          console.log(error.response.data.message)
          this.toastAlert(error.response.data.message)
        }
      )
    },
    moveToToday() {
      this.dateSelected = new Date()
    },
    checkIfBookmarked() {
      if (this.dateSelected == null) {
        this.dateIsBookmarked = false
        return
      }
      if (this.datesWithBookmarkRaw.includes(this.dateSelected.getDate())) {
        this.dateIsBookmarked = true
      } else {
        this.dateIsBookmarked = false
      }
    },
    async daySelected() {
      if (this.dateSelected == null) {
        console.log('date is null')
        return
      }

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

      this.checkIfBookmarked()

      if (!this.isSaved) {
        await this.autoSave(this.lastDateSelected)
      }

      this.lastDateSelected = this.dateSelected

      this.isLoading = true

      var loadingDay = this.dateSelected

      UserService.loadDay(this.dateSelected).then(
        (response) => {
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
        (error) => {
          this.isLoading = false
          this.toastAlert(error.response.data.message)
          this.$parent.$parent.transitionName = 'slideRight'
          this.$store.dispatch('auth/logout')
          this.$router.push('/login')
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
.bookmarkBtn {
  margin: 10px 0;
  background-color: #f57c00;
  border-radius: 5px;
}
.bookmarkBtn:hover {
  background-color: #ff9800;
}

/* Disable dropdown animation on iOS because of Webkit-bug */
@supports (-webkit-touch-callout: none) {
  .dropdown-content {
    transform: none !important;
  }
}

.dropdown-content li {
  margin: 0;
}

.dropdown-trigger {
  background-color: #2196f3;
  color: white;
  border-radius: 5px;
  border: 1px solid #2196f3;
  text-decoration: none;
  transition: box-shadow ease 0.3s;
}

.dropdown-trigger:hover {
  background-color: #2196f3;
  box-shadow: 0 0px 4px 4px rgba(0, 0, 0, 0.14);
}

.dropdown-content li > a {
  color: #2196f3;
}

.backup-wrap {
  position: relative;
}

.backup-codes-textarea {
  border: 1px solid #d5d5d5;
}

.copyToClipboard {
  position: absolute;
  top: 0em;
  right: 2.5em;
  margin-top: 4px;
  margin-right: 4px;
  width: 11px;
  height: 13px;
  cursor: pointer;
}

.copyToClipboard:before {
  top: -1px;
  left: 2px;
  width: 5px;
  height: 1px;
}

.copyToClipboard:after {
  width: 3px;
  height: 1px;
  background-color: #333333;
  box-shadow: 8px 0 0 0 #333333;
}

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

.removeDay {
  float: right;
  color: #e57373;
  margin-left: auto;
  cursor: pointer;
  opacity: 0.4;
  border: 1px solid transparent;
  border-radius: 4px;
  transition: ease 0.3s;
}

.removeDay:hover {
  opacity: 1;
  border: 1px solid #bdbdbd;
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

.dateDescriptionBookmarked {
  background-color: #ffa803;
}

.dateDescription > hr {
  width: 100%;
  border: none;
  margin: 0;
}

@media only screen and (max-width: 1450px) and (min-width: 1316px) {
  .dateDescription {
    font-size: 12.5px;
  }

  .dateWritten {
    font-size: 12.5px;
  }
}

@media only screen and (max-width: 1315px) and (min-width: 1200px) {
  .dateDescription {
    font-size: 11px;
  }

  .dateWritten {
    font-size: 12px;
  }
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

.modal {
  box-shadow: 0px 0px 5px 5px #2196f3;
  border-radius: 20px;
}

.input-container {
  padding: 0 1rem;
  border-radius: 10px;
  border: 1px solid #e0e0e0;
}

.input-divider {
  height: 0;
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
  height: calc(100vh - 84px);
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

.searchResultsPulse {
  animation: searchResultsPulseAnimation 0.75s;
}

@keyframes searchResultsPulseAnimation {
  0% {
    box-shadow: 0px 0px 4px 1px #ff7043;
  }

  40%,
  60% {
    box-shadow: 0px 0px 4px 3px #2196f3;
  }

  100% {
    box-shadow: 0px 0px 4px 1px #ff7043;
  }
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
  margin-bottom: auto;
}

.main {
  margin-top: 20px;
  height: calc(100% - 64px - 20px);
}

@media only screen and (min-width: 1201px) {
  .hide-on-xlarge {
    display: none;
  }
}

@media only screen and (max-width: 1200px) {
  .hide-on-large-and-down {
    display: none;
  }
}

@media only screen and (min-width: 1600px) {
  .col#left,
  .col#right {
    width: 16.7%;
    margin-left: auto;
  }
}

@media only screen and (max-width: 630px) {
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
