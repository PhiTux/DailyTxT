<template>
  <div class="main">
    <nav class="transparent">
      <div class="nav-wrapper">
        <a class="brand-logo" id="settings-header">
          <span class="left">
            {{ $t('settings') }}
            <i v-if="!isMobile" class="material-icons right">arrow_forward</i>
          </span>
          <span class="right" v-if="isMenuSelected('password') && !isMobile">
            {{ $t('change-password') }}
          </span>
          <span
            class="right"
            v-if="isMenuSelected('backup-codes') && !isMobile"
          >
            {{ $t('create-backup-codes') }}
          </span>
          <span class="right" v-if="isMenuSelected('templates') && !isMobile">
            {{ $t('templates') }}
          </span>
          <span class="right" v-if="isMenuSelected('export') && !isMobile">
            {{ $t('export-data') }}
          </span>
          <span class="right" v-if="isMenuSelected('import') && !isMobile">
            {{ $t('import-data') }}
          </span>
        </a>
        <ul class="right">
          <li>
            <a
              id="close-settings-btn"
              class="waves-effect waves-light btn-large"
              @click.prevent="switchSettings"
              ><span v-if="!isMobile">{{ $t('close-settings') }}</span>
              <i class="material-icons" :class="{ right: !isMobile }"
                >close</i
              ></a
            >
          </li>
        </ul>
      </div>
    </nav>
    <ul id="slide-out" class="settings-sidenav sidenav sidenav-fixed">
      <li>
        <a
          @click.prevent="setting('password')"
          :class="{ active: isMenuSelected('password') }"
          class="waves-effect"
          ><i class="material-icons">vpn_key</i>{{ $t('change-password') }}</a
        >
      </li>
      <li class="divider"></li>
      <li>
        <a
          @click.prevent="setting('backup-codes')"
          :class="{ active: isMenuSelected('backup-codes') }"
          class="waves-effect"
          ><i class="material-icons">security</i
          >{{ $t('create-backup-codes') }}</a
        >
      </li>
      <li class="divider"></li>
      <li class="spacer"></li>
      <li class="divider"></li>
      <li>
        <a
          href="#"
          :class="{ active: isMenuSelected('templates') }"
          class="waves-effect"
          @click.prevent="setting('templates')"
          ><i class="material-icons">library_books</i>{{ $t('templates') }}</a
        >
      </li>
      <li class="divider"></li>
      <li class="spacer"></li>
      <li class="divider"></li>
      <li>
        <a
          href="#"
          :class="{ active: isMenuSelected('export') }"
          class="waves-effect"
          @click.prevent="setting('export')"
          ><i class="material-icons">cloud_download</i
          >{{ $t('export-data') }}</a
        >
      </li>
      <li class="divider"></li>
      <li>
        <a
          href="#"
          :class="{ active: isMenuSelected('import') }"
          class="waves-effect"
          for="importData"
          @click.prevent="setting('import')"
          ><i class="material-icons">cloud_upload</i>{{ $t('import-data') }}</a
        >
      </li>
      <li class="divider"></li>
      <li class="version" id="version">
        <span>{{ $t('dailytxt-version') }}: {{ clientVersion }}</span>
      </li>
    </ul>
    <div class="settings-area">
      <transition
        v-on:after-enter="transitionEnd"
        v-on:before-leave="transitionStart"
        :name="transitionName"
        mode="out-in"
      >
        <Password v-if="isMenuSelected('password')" />
        <BackupCodes v-else-if="isMenuSelected('backup-codes')" />
        <Templates v-else-if="isMenuSelected('templates')" />
        <Export v-else-if="isMenuSelected('export')" />
        <Import v-else-if="isMenuSelected('import')" />
      </transition>
    </div>
  </div>
</template>

<script>
import $ from 'jquery'
import M from 'materialize-css'
import { eventBus } from '../main.js'
import Password from '@/components/settings/Password.vue'
import BackupCodes from '@/components/settings/BackupCodes.vue'
import Templates from '@/components/settings/Templates.vue'
import Export from '@/components/settings/Export.vue'
import Import from '@/components/settings/Import.vue'
import { version } from '../../package'

export default {
  name: 'Settings',
  components: {
    Password,
    BackupCodes,
    Templates,
    Export,
    Import
  },
  data() {
    return {
      selectedMenu: 'password',
      transitionName: 'fade',
      isMobile: false,
      clientVersion: version
    }
  },
  methods: {
    onResize() {
      this.isMobile = window.innerWidth < 900
    },
    switchSettings() {
      eventBus.$emit('switchSettings')
    },
    setting(menu) {
      this.selectedMenu = menu
    },
    isMenuSelected(check) {
      return this.selectedMenu == check
    },
    transitionEnd() {
      document.getElementById('app').style.overflow = 'auto'
      this.transitionName = 'fade'
    },
    transitionStart() {
      document.getElementById('app').style.overflow = 'hidden'
    }
  },
  mounted() {
    this.onResize()
    window.addEventListener('resize', this.onResize, { passive: true })

    $(document).ready(function () {
      var sidenav = document.querySelectorAll('.sidenav')[0]
      M.Sidenav.init(sidenav, {})
    })
  }
}
</script>

<style scoped>
.version {
  margin-top: 20px;
}

.fade-enter-active,
.fade-leave-active {
  transition: all 0.2s ease-out;
}

.fade-enter-from {
  opacity: 0;
}

.fade-leave-to {
  opacity: 0;
}

.main {
  width: 100%;
  height: calc(100vh - 64px) !important;
}

ul > .divider {
  margin: 4px 0 4px 0;
}

.settings-sidenav {
  position: fixed !important;
  top: inherit !important;
}

.spacer {
  height: 40px;
}

.waves-effect {
  overflow: clip;
}

.active {
  box-shadow: 0px 0px 8px 6px rgba(76, 175, 80, 0.8);
  border-radius: 7px;
  background-color: rgba(76, 175, 80, 0.1) !important;
}

.sidenav li > a {
  margin: 5px;
  height: auto;
  line-height: 25px;
  padding: 20px 32px;
  background-color: rgba(0, 0, 0, 0.03);
  outline: none;
}

.sidenav li > a > i.material-icons {
  line-height: inherit;
}

#settings-header {
  font-size: 1.5rem;
  color: #0d47a1;
  left: 20px;
  transform: none;
}

#settings-header > i {
  line-height: 64px;
}

#close-settings-btn {
  border: 2px solid #f57c00;
  background-color: transparent;
  border-radius: 10px;
  text-transform: none;
  transition: 0.3s;
  color: #2196f3;
}

#close-settings-btn:hover {
  border: 2px solid #2196f3;
  background-color: #f57c00;
  border-radius: 10px;
}

.settings-area {
  margin-top: 50px;
}

@media only screen and (max-width: 600px) {
  .settings-sidenav {
    height: calc(100% - 56px) !important;
    top: 2px;
  }
}

@media only screen and (min-width: 601px) {
  .settings-sidenav {
    height: calc(100%) !important;
    top: 2px;
  }
}

@media only screen and (max-width: 992px) {
  .settings-area {
    margin-left: 0;
  }
}

@media only screen and (min-width: 993px) {
  .settings-area {
    margin-left: 300px;
  }
}
</style>
