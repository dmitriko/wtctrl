<template>
  <div id="q-app">
    <router-view />
  </div>
</template>

<script>


export default {
    name: 'App',
    created () {
        if (!this.isLoggedIn) {
            this.$router.push("/login")
        }
    },
    data() {
        return {
            isLoggedIn: false
        }
    }
}
</script>
