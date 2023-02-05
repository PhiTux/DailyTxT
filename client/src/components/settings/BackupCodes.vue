<template>
  <div class="container">
    <h4>{{ $t('backup-codes') }}</h4>
    <div class="container backup-codes-description">
      {{ $t('backup-codes-description-1') }}
      <br />
      <b>{{ $t('attention') }}:</b>
      {{ $t('backup-codes-description-2') }}
    </div>
    <div class="divider"></div>
    <div class="col s2 m4 l3" id="loading" v-if="backupCodesLoading">
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
    <div class="container" v-if="backupCodesHTML != ''">
      <h6>{{ $t('backup-codes-created-1') }}</h6>
      {{ $t('backup-codes-created-2') }}
      <div class="backup-wrap">
        <pre class="backup-codes-textarea" v-html="backupCodesHTML"></pre>
        <div v-if="canCopy" class="copyToClipboard">
          <a
            class="btn-floating waves-effect waves-light deep-orange lighten-1"
            @click.prevent="copy(backupCodesCopy)"
            ><i class="material-icons">content_copy</i></a
          >
        </div>
      </div>
    </div>
    <div v-if="backupCodesHTML != ''" class="divider"></div>
    <div class="container">
      <div class="row">
        <div class="input-field col s12">
          <input
            id="backupCodesPassword"
            type="password"
            v-model="password"
            @keyup.enter="createBackupCodes()"
          />
          <label for="backupCodesPassword">{{ $t('password-label') }}</label>
          <a
            class="waves-effect waves-light btn right deep-orange lighten-1"
            @click="createBackupCodes()"
            :class="{
              disabled: password == ''
            }"
            ><i class="material-icons left">security</i
            >{{ $t('create-new-backup-codes') }}</a
          >
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import UserService from '../../services/user.service.js'
import { eventBus } from '../../main.js'

export default {
  name: 'BackupCodes',
  data() {
    return {
      backupCodes: [],
      canCopy: false,
      password: '',
      backupCodesLoading: false
    }
  },
  methods: {
    createBackupCodes() {
      this.backupCodesLoading = true
      UserService.createBackupCodes(this.password).then(
        (response) => {
          this.backupCodesLoading = false
          this.password = ''
          if (response.data.success) {
            this.backupCodes = response.data.backupCodes
          } else {
            if ('message' in response.data) {
              console.log(response.data.message)
              eventBus.$emit('toastAlert', response.data.message)
            } else {
              console.log(this.$t('backup-codes-not-successful'))
              eventBus.$emit(
                'toastAlert',
                this.$t('backup-codes-not-successful')
              )
            }
          }
        },
        (error) => {
          this.backupCodesLoading = false
          this.password = ''
          console.log(error.response.data.message)
          eventBus.$emit('toastAlert', error.response.data.message)
        }
      )
    },
    async copy(s) {
      await navigator.clipboard.writeText(s)
      eventBus.$emit('toastSuccess', this.$t('copy-to-clipboard-successful'))
    }
  },
  computed: {
    backupCodesHTML: function () {
      var html = ''
      if (this.backupCodes.length > 0) {
        this.backupCodes.forEach((t) => {
          html += html == '' ? t : '<br />' + t
        })
      }
      return html
    },
    backupCodesCopy: function () {
      var text = ''
      if (this.backupCodes.length > 0) {
        this.backupCodes.forEach((t) => {
          text += text == '' ? t : '\n' + t
        })
      }
      return text
    }
  },
  created: function () {
    this.canCopy = !!navigator.clipboard
  },
  beforeUnmount() {
    this.backupCodes = []
  }
}
</script>

<style scoped>
.divider {
  width: 90%;
  margin: 20px 5% 20px 5% !important;
}

.backup-codes-description {
  text-align: left;
}
</style>
