<template>
  <div id="q-app">
    <router-view v-if="isLoggedIn" />
    <Login v-else @loggedIn="isLoggedIn = true" />
  </div>
</template>

<script>

import Login from 'pages/Login.vue'

export default {
    name: 'App',
    components: {Login},
    data() {
        return {
            isLoggedIn: false
        }
    }
}
</script>
