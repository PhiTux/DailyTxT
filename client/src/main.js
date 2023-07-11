import router from './router'
import Vue from 'vue'
import App from './App.vue'
//import './registerServiceWorker'
import useRegisterSW from '@/mixins/useRegisterSW'
//import './useRegisterSW'
import store from './store'
import 'materialize-css/dist/css/materialize.min.css'
import 'material-design-icons-iconfont/dist/material-design-icons.css'
import $ from 'jquery'
import 'animate.css/animate.min.css'
import VueI18n from 'vue-i18n'
import messages from './lang'

Vue.config.productionTip = false

export const eventBus = new Vue()

const intervalMS = 60 * 1000 // every minute (in ms)

//export default {
//name: 'ReloadPrompt',
//mixins: [useRegisterSW]
/* methods: {
    handleSWManualUpdates(r) {
      r &&
        setInterval(() => {
          r.update()
        }, intervalMS)
    }
  } */
//}

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
