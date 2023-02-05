<template>
  <div class="home">
    <NavBar />
    <transition
      v-on:after-enter="transitionEnd"
      v-on:before-leave="transitionStart"
      :name="transitionName"
      mode="out-in"
      appear
    >
      <DailyTxT v-if="!settings" />
      <Settings v-else />
    </transition>
  </div>
</template>

<script>
// @ is an alias to /src
import DailyTxT from '@/components/DailyTxT.vue'
import NavBar from '@/components/NavBar.vue'
import Settings from '@/components/Settings.vue'
import M from 'materialize-css'
import { eventBus } from '../main.js'

export default {
  name: 'Home',
  data() {
    return {
      settings: false,
      transitionName: ''
    }
  },
  created: function () {
    eventBus.$off('switchSettings')
    eventBus.$on('switchSettings', () => {
      if (this.settings) {
        this.transitionName = 'slideUp'
      } else {
        this.transitionName = 'slideDown'
        eventBus.$emit('hideHistoryButton')
      }
      this.settings = !this.settings
      this.$store.state.historyAvailable = false
    })
    eventBus.$off('toastSuccess')
    eventBus.$on('toastSuccess', (message) => {
      this.toastSuccess(message)
    })
    eventBus.$off('toastAlert')
    eventBus.$on('toastAlert', (message) => {
      this.toastAlert(message)
    })
  },
  components: {
    NavBar,
    DailyTxT,
    Settings
  },
  methods: {
    transitionEnd() {
      document.getElementById('app').style.overflow = 'auto'
      this.transitionName = 'fade'
    },
    transitionStart() {
      document.getElementById('app').style.overflow = 'hidden'
    },
    toastSuccess(message) {
      M.toast({ html: message, classes: 'rounded green' })
    },
    toastAlert(message) {
      M.toast({ html: message, classes: 'rounded red' })
    }
  }
}
</script>

<style scoped>
.home {
  height: 100%;
}

.slideDown-leave-active {
  animation: down-fade-out 0.3s;
}

.slideDown-enter-active {
  animation: down-fade-in 0.3s;
}

@keyframes down-fade-out {
  0% {
    opacity: 1;
    transform: translateY(0);
  }

  100% {
    opacity: 0;
    transform: translateY(5%);
  }
}

@keyframes down-fade-in {
  0% {
    opacity: 0;
    transform: translateY(-5%);
  }

  100% {
    opacity: 1;
    transform: translateY(0);
  }
}

.slideUp-leave-active {
  animation: up-fade-out 0.3s;
}

.slideUp-enter-active {
  animation: up-fade-in 0.3s;
}

@keyframes up-fade-out {
  0% {
    opacity: 1;
    transform: translateY(0);
  }

  100% {
    opacity: 0;
    transform: translateY(-5%);
  }
}

@keyframes up-fade-in {
  0% {
    opacity: 0;
    transform: translateY(5%);
  }

  100% {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
