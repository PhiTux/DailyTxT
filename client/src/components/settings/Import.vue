<template>
  <div class="container">
    <h4>{{ $t('import-data') }}</h4>
    <div
      class="container import-data-description"
      v-html="$t('import-data-description')"
    ></div>
    <div class="container">
      <div class="row">
        <div class="input-field col s12">
          <a
            class="waves-effect waves-light btn deep-orange lighten-1"
            @click="triggerImportData"
            ><i class="material-icons left">cloud_upload</i
            >{{ $t('start-import') }}</a
          >
          <input
            type="file"
            id="importData"
            name="importData"
            @change="importData($event)"
          />
        </div>
      </div>
    </div>

    <div class="container import-steps" v-if="isImporting">
      <h4>{{ $t('import-data-header') }}</h4>
      <li class="upload-icons">
        <i v-if="importStep == 1" class="material-icons">arrow_forward</i>
        <i v-if="importStep > 1" class="material-icons">check</i>
      </li>
      {{ $t('import-data-text-1') }}
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
      {{ $t('import-data-text-2') }}
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
  </div>
</template>

<script>
import UserService from '../../services/user.service.js'
import { eventBus } from '../../main.js'

export default {
  name: 'Export',
  data() {
    return {
      isImporting: false,
      importProgress: 0,
      importStep: 0
    }
  },
  methods: {
    importData(event) {
      var f = event.target.files[0]

      this.isImporting = true
      this.importStep = 1

      UserService.importData(f, (event) => {
        this.importProgress = Math.round((100 * event.loaded) / event.total)

        if (event.loaded == event.total) {
          this.importStep = 2
        }
      }).then(
        (response) => {
          if (response.data.success) {
            this.importStep = 3
            eventBus.$emit('toastSuccess', this.$t('import-successful'))
          }
        },
        (error) => {
          if (typeof error.response.data.message !== 'undefined') {
            console.log(error.response.data.message)
            eventBus.$emit('toastAlert', error.response.data.message)
            this.importStep = 3
          } else {
            console.log(error.response.data)
            eventBus.$emit('toastAlert', this.$t('error-uploading-file'))
            this.importStep = 3
          }
        }
      )
    },
    triggerImportData(e) {
      e.preventDefault()
      document.querySelector('#importData').click()
    }
  }
}
</script>

<style scoped>
.import-steps {
  margin-top: 50px;
  box-shadow: 0 0 40px 10px #2196f3;
  border-radius: 20px;
  padding: 30px;
}

.import-steps > h4 {
  margin: 0 0 1.2rem 0;
}

#importData {
  display: none;
}

.upload-icons {
  vertical-align: middle;
}

.importProgress {
  display: list-item;
  list-style: none;
}

li {
  display: inline-block;
  margin: 0 10px;
}

.divider {
  width: 90%;
  margin: 20px 5% 20px 5% !important;
}

.import-data-description {
  text-align: left;
}
</style>
