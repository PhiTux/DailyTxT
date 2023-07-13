<template>
  <div class="full-site">
    <div class="logo-container">
      <img
        class="logo"
        src="../../public/img/icons/locked_heart_with_keyhole.svg"
      />
    </div>
    <h2>DailyTxT</h2>
    <ul
      class="collapsible login-container"
      :class="{ 'animate__animated animate__shakeX': error_shake }"
      @animationend="error_shake = false"
    >
      <li class="active">
        <div class="collapsible-header">
          <i class="material-icons">login</i>{{ $t('login-header') }}
        </div>
        <div class="collapsible-body">
          <form @submit.prevent="handleLogin">
            <div class="row">
              <div class="input-field col s12">
                <input
                  id="username"
                  type="text"
                  v-model="login_username"
                  name="username"
                  autofocus
                />
                <label for="username">{{ $t('username-label') }}</label>
              </div>
            </div>
            <div class="row">
              <div class="input-field col s12">
                <input id="password" type="password" v-model="login_password" />
                <label for="password">{{ $t('password-label') }}</label>
              </div>
            </div>
            <div class="row">
              <button
                v-if="!login_loading"
                class="btn"
                :class="{
                  disabled: login_username == '' || login_password == ''
                }"
              >
                <span>{{ $t('login-button') }}</span>
              </button>
              <div v-if="login_loading" class="preloader-wrapper small active">
                <div class="spinner-layer spinner-green-only">
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
            <div class="row alert" v-if="login_error">
              {{ login_error.message }}
            </div>
            <div class="row success" v-if="register_success">
              {{ $t('registration-successful-please-login') }}
            </div>
          </form>
        </div>
      </li>
      <li>
        <div class="collapsible-header">
          <i class="material-icons">person_add</i>{{ $t('register-header') }}
        </div>
        <div class="collapsible-body">
          <form @submit.prevent="handleRegister">
            <div class="row">
              <div class="input-field col s12">
                <input id="name" type="text" v-model="register_username" />
                <label for="name">{{ $t('username-label') }}</label>
              </div>
            </div>
            <div class="row">
              <div class="input-field col s12">
                <input
                  id="password1"
                  type="password"
                  v-model="register_password1"
                />
                <label for="password1">{{ $t('password-label') }}</label>
              </div>
              <div class="input-field col s12">
                <input
                  id="password2"
                  type="password"
                  v-model="register_password2"
                />
                <label for="password2">{{
                  $t('confirm-password-label')
                }}</label>
              </div>
              <div
                class="alert"
                v-if="register_password1 != register_password2"
              >
                {{ $t('password-does-not-match') }}
              </div>
            </div>
            <div class="row">
              <button
                v-if="!register_loading"
                class="btn"
                :class="{
                  disabled:
                    register_username == '' ||
                    register_password1 == '' ||
                    register_password1 != register_password2
                }"
              >
                <span>{{ $t('register-button') }}</span>
              </button>
              <div
                v-if="register_loading"
                class="preloader-wrapper small active"
              >
                <div class="spinner-layer spinner-green-only">
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
            <div class="row alert" v-if="register_error">
              {{ register_error.message }}
            </div>
          </form>
        </div>
      </li>
    </ul>
    <div class="fill"></div>
    <div class="version">
      {{ clientVersion }}
    </div>
  </div>
</template>

<script>
import $ from 'jquery'
import M from 'materialize-css'
import User from '../models/user'
import { version } from '../../package.json'

export default {
  name: 'Login',
  data() {
    return {
      login_loading: false,
      register_loading: false,
      login_error: '',
      register_error: '',
      register_success: false,
      error_shake: false,
      register_username: '',
      register_password1: '',
      register_password2: '',
      login_username: '',
      login_password: '',
      clientVersion: version
    }
  },
  computed: {
    loggedIn() {
      return this.$store.state.auth.status.loggedIn
    }
  },
  created: function () {
    if (this.loggedIn) {
      this.$router.push('/')
    }
  },
  methods: {
    handleLogin() {
      if (this.login_username && this.login_password) {
        this.login_loading = true
        this.$store
          .dispatch(
            'auth/login',
            new User(this.login_username, this.login_password)
          )
          .then(
            (response) => {
              if (
                response.remaining_backup_codes > 0 &&
                response.remaining_backup_codes <= 3
              ) {
                let message = this.$t('few-backup-codes-left', [
                  response.remaining_backup_codes
                ])
                M.toast({ html: message, classes: 'rounded red' })
              } else if (response.used_backup_code) {
                let message = this.$t('x-backup-codes-left', [
                  response.remaining_backup_codes
                ])
                M.toast({ html: message, classes: 'rounded red' })
              }

              this.$parent.transitionName = 'slideLeft'
              this.$router.push('/')
            },
            (error) => {
              this.login_loading = false
              this.error_shake = true
              this.login_error =
                (error.response && error.response.data) ||
                error.message ||
                error.toString()
            }
          )
      }
    },
    handleRegister() {
      if (this.register_username && this.register_password1) {
        this.register_loading = true
        this.$store
          .dispatch(
            'auth/register',
            new User(this.register_username, this.register_password1)
          )
          .then(
            () => {
              this.register_loading = false
              this.login_error = ''
              this.register_error = ''
              this.register_success = true
              var elem = document.querySelectorAll('.collapsible')[0]
              var instance = M.Collapsible.getInstance(elem)
              instance.open(0)
            },
            (error) => {
              this.register_loading = false
              this.error_shake = true
              this.register_error =
                (error.response && error.response.data) ||
                error.message ||
                error.toString()
            }
          )
      }
    }
  },
  mounted() {
    $(document).ready(function () {
      var elems = document.querySelectorAll('.collapsible')
      M.Collapsible.init(elems, {})
      document.getElementById('username').focus()
    })
  }
}
</script>

<style scoped>
.fill {
  display: flex;
  flex: 1;
}

.version {
  display: flex;
  margin: 10px;
  font-size: large;
  height: 30px;
  align-self: end;
}

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

.collapsible-header {
  background-color: inherit;
}

.full-site {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.logo-container {
  min-width: calc(min(300px, 90%));
  max-width: calc(min(400px, 90%));
  margin-top: 10px;
  filter: drop-shadow(0 0 10px grey);
}

.logo {
  width: 50%;
  height: auto;
  transition: ease 0.3s;
}

.logo:hover {
  transform: scale(1.1);
}

h2 {
  margin-top: 0.5rem;
  color: #1565c0;
  text-decoration-line: underline;
  text-decoration-color: #f57c00;
}

.login-container {
  min-width: calc(min(300px, 90%));
  max-width: calc(min(450px, 90%));
  width: 100%;
  box-shadow: 0 0 5px 2px #039be5;
}

.alert {
  border: 1px solid #f44336;
  border-radius: 5px;
  color: #f44336;
}

.success {
  border: 1px solid #4caf50;
  border-radius: 5px;
  color: #4caf50;
}

.btn {
  background-color: #039be5;
}
</style>
