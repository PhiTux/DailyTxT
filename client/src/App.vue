<template>
  <div id="app">
    <transition
      v-on:after-enter="transitionEnd"
      v-on:before-leave="transitionStart"
      :name="transitionName"
      mode="out-in"
      appear
    >
      <router-view />
    </transition>
  </div>
</template>

<script>
import M from 'materialize-css'

export default {
  data() {
    return {
      transitionName: 'fade'
    }
  },
  mounted() {
    M.AutoInit()
  },
  methods: {
    transitionEnd() {
      document.getElementById('app').style.overflow = 'auto'
      this.transitionName = 'fade'
    },
    transitionStart() {
      document.getElementById('app').style.overflow = 'hidden'
    }
  }
}
</script>

<style>
html {
  height: 100%;
}

body {
  height: 100%;
}

#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
  overflow: hidden;
  height: 100%;
}

#nav {
  padding: 30px;
}

#nav a {
  font-weight: bold;
  color: #2c3e50;
}

#nav a.router-link-exact-active {
  color: #42b983;
}

.sidenav-overlay {
  background-color: inherit !important;
}

.slideRight-leave-active {
  animation: my-fade-out 0.5s;
}

.slideRight-enter-active {
  animation: slide-left-in 0.5s;
}

@keyframes my-fade-out {
  0% {
    opacity: 1;
    transform: scale(1);
  }

  100% {
    opacity: 0;
    transform: scale(0.92);
  }
}

@keyframes slide-right-in {
  0% {
    opacity: 0;
    transform: translateX(20%) scale(0.92);
  }

  50% {
    transform: translateX(0) scale(0.92);
  }

  100% {
    opacity: 1;
    transform: translateX(0) scale(1);
  }
}

@keyframes slide-left-in {
  0% {
    opacity: 0;
    transform: translateX(-20%) scale(0.92);
  }

  50% {
    transform: translateX(0) scale(0.92);
  }

  100% {
    opacity: 1;
    transform: translateX(0) scale(1);
  }
}

.slideLeft-leave-active {
  animation: my-fade-out 0.5s;
}

.slideLeft-enter-active {
  animation: slide-right-in 0.5s;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity ease 0.4s, transform ease 0.4s;
}

.fade-enter {
  opacity: 0;
  transform: scale(0.92);
}

.fade-leave-to {
  opacity: 0;
  transform: scale(0.92);
}
</style>
