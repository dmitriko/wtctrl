<template>
    <div class="q-pa-md q-ml-md" ><span :class="error ? 'error' : '' " >{{ msg }}</span></div>
</template>

<script>
export default {
    name: 'SysMsg',
    computed: {
        error() {
            return this.$store.state.ui.sys_msg_error
        },
        msg() {
            return this.$store.state.ui.sys_msg
        }
    }
}
</script>

<style scoped>
    .error {
        color: red
    }
</style>
