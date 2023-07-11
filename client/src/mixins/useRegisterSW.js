export default {
  name: 'useRegisterSW',
  data() {
    return {
      updateSW: undefined,
      offlineReady: false,
      needRefresh: false
    }
  },
  async mounted() {
    try {
      console.log('mounted')
      const { registerSW } = await import('virtual:pwa-register')
      const vm = this
      this.updateSW = registerSW({
        immediate: true,
        onNeedRefresh() {
          vm.needRefresh = true
          vm.onNeedRefreshFn()
        }
      })
    } catch {
      console.log('DailyTxT disabled - try reloading')
    }
  },
  methods: {
    async closePromptUpdateSW() {
      this.offlineReady = false
      this.needRefresh = false
    },
    onNeedRefreshFn() {
      console.log('onNeedRefresh')
    },
    updateServiceWorker() {
      this.updateSW && this.updateSW(true)
    },
    handleSWManualUpdates(swRegistration) {},
    handleSWRegisterError(error) {}
  }
}
