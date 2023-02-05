<template>
  <div class="container">
    <h4>{{ $t('export-data') }}</h4>
    <div
      class="container export-data-description"
      v-html="$t('export-data-description')"
    ></div>
    <div class="divider"></div>
    <div class="col s2 m4 l3" id="loading" v-if="isExporting">
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
    <div class="container">
      <div class="row">
        <div class="input-field col s12">
          <input
            id="exportPassword"
            type="password"
            v-model="password"
            @keyup.enter="exportData()"
          />
          <label for="exportPassword">{{ $t('password-label') }}</label>
          <a
            class="waves-effect waves-light btn right deep-orange lighten-1"
            @click="exportData()"
            :class="{
              disabled: password == ''
            }"
            ><i class="material-icons left">cloud_download</i
            >{{ $t('create-export') }}</a
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
  name: 'Export',
  data() {
    return {
      password: '',
      isExporting: false
    }
  },
  methods: {
    exportData() {
      if (this.isExporting) {
        return
      }
      this.isExporting = true

      eventBus.$emit('toastSuccess', this.$t('export-started-1'))
      eventBus.$emit('toastSuccess', this.$t('export-started-2'))

      UserService.exportData(this.password).then(
        async (response) => {
          if (response.data.type == 'application/json') {
            let blob = new Blob([response.data], { type: 'application/json' })
            await blob.text().then((a) => {
              var res = JSON.parse(a)
              if (!res.success) {
                eventBus.$emit('toastAlert', res.message)
              }
              this.isExporting = false
            })
          } else {
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
          }
        },
        (error) => {
          this.isExporting = false
          console.log(error)
        }
      )
    }
  }
}
</script>

<style scoped>
.divider {
  width: 90%;
  margin: 20px 5% 20px 5% !important;
}

.export-data-description {
  text-align: left;
}
</style>
