<template>
  <nav class="blue">
    <div class="nav-wrapper">
      <a data-target="slide-out" class="left sidenav-trigger show-on-small"
        ><i class="material-icons">menu</i></a
      >
      <a class="brand-logo center">
        <img src="../../public/img/icons/locked_heart_with_keyhole.svg" />
        <div class="title hide-on-small-only">
          <div class="title-text">DailyTxT</div>
        </div>
      </a>
      <ul id="nav-mobile" class="right">
        <li v-if="this.newerDailyTxTVersion">
          <a @click.prevent="versionToast"
            ><i style="color: #f57c00" class="material-icons"
              >info_outline</i
            ></a
          >
        </li>
        <li v-if="this.$store.state.historyAvailable">
          <a @click.prevent="historyModal"
            ><i class="material-icons">history</i></a
          >
        </li>
        <li>
          <a class="settings" @click.prevent="switchSettings"
            ><i class="material-icons">settings</i></a
          >
        </li>
        <li>
          <a @click.prevent="logOut"><i class="material-icons">logout</i></a>
        </li>
      </ul>
    </div>
  </nav>
</template>

<script>
import { eventBus } from '../main.js'
import supported_locales from '../lang/supported-locales.json'

export default {
  name: 'NavBar',
  data() {
    return {
      locales: supported_locales.langs,
      newerDailyTxTVersion: false
    }
  },
  methods: {
    switchSettings() {
      eventBus.$emit('switchSettings')
    },
    historyModal() {
      eventBus.$emit('historyModal')
    },
    logOut() {
      this.$parent.$parent.transitionName = 'slideRight'
      this.$store.dispatch('auth/logout')
      this.$router.push('/login')
    },
    versionToast() {
      eventBus.$emit('updateModal')
    }
  },
  beforeMount() {
    this.$root.$on('dailytxt_version_update', (data) => {
      this.newerDailyTxTVersion = data.update_available
    })
  }
}
</script>

<style scoped>
.brand-logo {
  height: 100%;
  display: inline-flex;
  flex-direction: row;
  align-items: center;
}

.brand-logo > img {
  height: 80%;
  margin-right: 1rem;
  transition: ease 0.3s;
}

.brand-logo:hover > img {
  transform: scale(1.1);
  filter: drop-shadow(0px 0px 5px #90caf9);
}

.title-text {
  position: relative;
}

.title-text::before {
  content: '';
  position: absolute;
  width: 100%;
  height: 2px;
  bottom: 10px;
  left: 0;
  background-color: #f57c00;
  visibility: hidden;
  transform: scaleX(0);
  transition: all 0.3s ease;
}

.brand-logo:hover .title-text::before {
  visibility: visible;
  transform: scaleX(1);
}

a {
  cursor: pointer;
}

.settings > i {
  transition: transform 0.4s ease-in-out;
}

.settings:hover > i {
  transform: rotate(30deg);
}

.dropdown-content,
.dropdown-content:hover {
  min-width: 250px;
  max-width: 450px;
  background: #fff0;
  box-shadow: none;
}

.dropdown-content > li:hover {
  background: none;
}

.divider {
  width: 0;
}

.dropdown-content > li {
  cursor: inherit;
  width: auto;
  float: right !important;
  min-height: auto;
  height: auto;
}

.dropdown-content > li > a {
  border-radius: 40px;
  color: #424242 !important;
  background: #29b6f6;
  margin-left: 0;
  margin-right: 0;
  margin-top: 5px;
  margin-bottom: 5px;
  text-transform: inherit;
  display: flex;
  align-items: center;
  transition: ease 0.3s;
}

.dropdown-content > li > a:hover {
  background: #81d4fa;
  box-shadow: 0 3px 4px 0 rgba(0, 0, 0, 0.14), 0 3px 7px 0 rgba(0, 0, 0, 0.12),
    0 3px 1px -1px rgba(0, 0, 0, 0.2);
}

.dropdown-content > li > a > i {
  display: flex;
  align-items: center;
}
</style>
