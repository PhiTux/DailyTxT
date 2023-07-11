<template>
  <div class="container">
    <h4>{{ $t('templates') }}</h4>
    <div class="row">
      <div class="col s12 l12 xl4">
        <a class="dropdown-trigger btn" data-target="dropdown1"
          >{{ $t('select-template') }}
          <i class="material-icons right">arrow_drop_down</i>
        </a>
        <ul id="dropdown1" class="dropdown-content">
          <li key="0">
            <a @click="selectTemplate(0)">{{ $t('create-new-template') }}</a>
          </li>
          <li v-for="t in templatesSorted" :key="t.number">
            <a @click="selectTemplate(t.number)">{{ t.name }}</a>
          </li>
        </ul>
      </div>

      <div class="col s12 l12 xl8">
        <div class="row">
          <div class="input-field col s12 m6">
            <input id="templateName" type="text" v-model="templateName" />
            <label :class="{ active: templateName != '' }" for="templateName">{{
              $t('template-name')
            }}</label>
          </div>
        </div>
        <div class="row">
          <textarea
            v-model="templateText"
            name="template-textarea"
            cols="30"
            rows="10"
            class="template-textarea"
          >
          </textarea>
        </div>
        <div class="row">
          <div
            class="left valign-wrapper"
            id="removeTemplate"
            v-if="templateIndex != 0"
          >
            <a
              class="removeTemplate valign-wrapper tooltipped"
              :data-tooltip="
                $t('remove-template-tooltip', { name: templateName })
              "
              data-position="top"
              @click="removeTemplate()"
              ><i class="material-icons">delete</i></a
            >
          </div>
          <a
            class="waves-effect waves-light btn right deep-orange lighten-1"
            @click="saveTemplate()"
            :class="{
              disabled: this.templateName == '' || this.templateText == ''
            }"
            ><i class="material-icons left">save</i
            ><span v-if="templateIndex == 0">{{ $t('save-new-template') }}</span
            ><span v-else>{{ $t('save-changes') }}</span></a
          >
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import $ from 'jquery'
import M from 'materialize-css'
import UserService from '../../services/user.service.js'
import { eventBus } from '../../main.js'

export default {
  name: 'Templates',
  data() {
    return {
      templates: [],
      templateIndex: 0,
      templateName: '',
      templateText: ''
    }
  },
  updated: function () {
    this.$nextTick(function () {
      var elems = document.querySelectorAll('.tooltipped')
      M.Tooltip.init(elems, {})
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
    }
  },
  methods: {
    selectTemplate(i) {
      this.templateIndex = i
      if (i == 0) {
        this.templateName = ''
        this.templateText = ''
      }
      this.templates.forEach((t) => {
        if (t.number == i) {
          this.templateName = t.name
          this.templateText = t.text
        }
      })
    },
    saveTemplate() {
      UserService.saveTemplate(
        this.templateIndex,
        this.templateName,
        this.templateText
      ).then(
        (response) => {
          if (response.data.success) {
            console.log(this.$t('saving-template-success'))
            eventBus.$emit('toastSuccess', this.$t('saving-template-success'))
            this.loadTemplates()
          } else {
            if (typeof response.data.message !== 'undefined') {
              console.log(response.data.message)
              eventBus.$emit('toastAlert', response.data.message)
            } else {
              console.log(this.$t('saving-template-error'))
              eventBus.$emit('toastAlert', this.$t('saving-template-error'))
            }
          }
        },
        (error) => {
          if (typeof error.response.data.message !== 'undefined') {
            console.log(error.response.data.message)
            eventBus.$emit('toastAlert', error.response.data.message)
          } else {
            console.log(this.$t('saving-template-error'))
            eventBus.$emit('toastAlert', this.$t('saving-template-error'))
          }
        }
      )
    },
    removeTemplate() {
      this.templateName = ''
      this.templateText = ''
      UserService.removeTemplate(this.templateIndex).then(
        (response) => {
          if (response.data.success) {
            console.log(this.$t('removing-template-success'))
            eventBus.$emit('toastSuccess', this.$t('removing-template-success'))
            this.templateIndex = 0
          } else {
            console.log(response.data.message)
            eventBus.$emit('toastAlert', response.data.message)
          }
          this.loadTemplates()
        },
        (error) => {
          console.log(error.response.data.message)
          eventBus.$emit('toastAlert', error.response.data.message)
          this.loadTemplates()
        }
      )
    },
    loadTemplates() {
      UserService.loadTemplates().then(
        (response) => {
          if (response.data.success) {
            this.templates = response.data.templates

            var elems = document.querySelectorAll('.dropdown-trigger')
            M.Dropdown.init(elems, {})
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
    }
  },
  mounted() {
    $(document).ready(function () {
      var select = document.querySelectorAll('select')[0]
      M.FormSelect.init(select, {})

      var dropdown = document.querySelectorAll('.dropdown-trigger')
      M.Dropdown.init(dropdown, {})
    })

    this.loadTemplates()
  }
}
</script>

<style scoped>
.removeTemplate {
  float: right;
  color: #e57373;
  margin-left: auto;
  cursor: pointer;
  opacity: 0.4;
  border: 1px solid transparent;
  border-radius: 4px;
  transition: ease 0.3s;
}

.removeTemplate:hover {
  opacity: 1;
  border: 1px solid #bdbdbd;
}

.dropdown-trigger.btn {
  width: 100%;
  background-color: #2196f3;
  transition: all ease 0.3s;
  overflow-inline: auto;
  overflow-y: clip;
  margin: 25px 0;
}

/* Disable dropdown animation on iOS because of Webkit-bug */
@supports (-webkit-touch-callout: none) {
  .dropdown-content {
    transform: none !important;
  }
}
.dropdown-content li > a {
  color: #2196f3;
}

.template-textarea {
  width: 100%;
  float: left;
  height: 500px;
  resize: vertical;
  background-color: #f5f5f5;
  transition: box-shadow ease 0.3s;
  margin-bottom: 20px;
  padding: 2px;
  border-bottom-left-radius: 10px;
  border-top-right-radius: 10px;
  border: 1px solid #ff9800;
}

.template-textarea:focus {
  outline: none;
  box-shadow: 0px 0px 0px 1px #ff9800;
}
</style>
