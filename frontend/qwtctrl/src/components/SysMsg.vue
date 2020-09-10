<template>
    <div class="fit row no-wrap justify-between" >
        <span class="q-mt-sm" :class="error ? 'error' : '' " >
            {{ msg }}
        </span>
    </div>
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
