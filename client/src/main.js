import router from './router'
import Vue from 'vue'
import App from './App.vue'
import './registerServiceWorker'
import store from './store'
import 'materialize-css/dist/css/materialize.min.css'
import 'material-design-icons-iconfont/dist/material-design-icons.css'
import $ from 'jquery'
import 'animate.css/animate.min.css'
import VueI18n from 'vue-i18n'
import messages from './lang'

Vue.config.productionTip = false

export const eventBus = new Vue()

Vue.use(VueI18n)
export const i18n = new VueI18n({
  locale: navigator.language.split('-')[0],
  fallbackLocale: 'en',
  messages
})

new Vue({
  router,
  store,
  i18n,
  $,
  render: (h) => h(App)
}).$mount('#app')
