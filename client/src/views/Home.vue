<template>
  <div class="home">
    <NavBar />
    <DailyTxT />
  </div>
</template>

<script>
// @ is an alias to /src
import DailyTxT from '@/components/DailyTxT.vue'
import NavBar from '@/components/NavBar.vue'

export default {
  name: 'Home',
  components: {
    NavBar,
    DailyTxT
  }
}
</script>

<style scoped>
.home {
  height: 100%;
}
</style>
