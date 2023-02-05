<template>
  <div class="container">
    <h4>{{ $t('modal-change-password-header') }}</h4>
    <div class="container">
      <div class="input-container">
        <div class="input-field">
          <input
            id="old_password"
            type="password"
            v-model="old_password"
            v-on:keyup.enter="onEnter"
          />
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
            v-on:keyup.enter="onEnter"
          />
          <label for="new_password1">{{ $t('new-password') }}</label>
        </div>
        <div class="input-field">
          <input
            id="new_password2"
            type="password"
            v-model="new_password2"
            v-on:keyup.enter="onEnter"
          />
          <label for="new_password2">{{ $t('confirm-new-password') }}</label>
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
    <div class="container">
      <a
        class="waves-effect waves-light btn orange darken-2"
        @click="changePassword()"
        :class="{
          disabled: inputError
        }"
        ><i class="material-icons left">save</i>{{ $t('save') }}</a
      >
    </div>
  </div>
</template>

<script>
import UserService from '../../services/user.service.js'
import { eventBus } from '../../main.js'

export default {
  name: 'Password',
  data() {
    return {
      old_password: '',
      new_password1: '',
      new_password2: ''
    }
  },
  computed: {
    inputError: function () {
      return (
        this.old_password == '' ||
        this.new_password1 == '' ||
        this.new_password1 != this.new_password2
      )
    }
  },
  methods: {
    onEnter() {
      if (!this.inputError) {
        this.changePassword()
      }
    },
    changePassword() {
      UserService.changePassword(this.old_password, this.new_password1).then(
        (response) => {
          if (response.data.success) {
            if (response.data.token) {
              localStorage.setItem('user', JSON.stringify(response.data))
              eventBus.$emit(
                'toastSuccess',
                this.$t('password-change-successful')
              )
              if (response.data.backup_codes_deleted) {
                console.log(this.$t('backup-codes-deleted'))
                eventBus.$emit('toastAlert', this.$t('backup-codes-deleted'))
              }
            }
          } else {
            console.log(response.data.message)
            eventBus.$emit('toastAlert', response.data.message)
          }
        },
        (error) => {
          console.log(error.response.data.message)
          eventBus.$emit('toastAlert', error.response.data.message)
        }
      )
    }
  }
}
</script>

<style scoped>
.btn {
  margin-top: 20px;
}

.divider {
  width: 90%;
  margin: 20px 5% 20px 5% !important;
}

/*.alert-password {
  color: #f44336;
  border: 1px solid #f44336;
  border-radius: 10px;
}*/
</style>
