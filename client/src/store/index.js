import Vue from 'vue'
import Vuex from 'vuex'

import { auth } from './auth.module'

Vue.use(Vuex)

export default new Vuex.Store({
  modules: {
    auth
  },
  state: {
    historyAvailable: false
  },
  mutations: {
    setHistoryAvailable(state, value) {
      state.historyAvailable = value
    }
  }
})
