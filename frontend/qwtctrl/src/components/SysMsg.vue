<template>
    <div class="q-pa-md q-ml-md" ><span :class="error ? 'error' : '' " >{{ msg }}</span></div>
</template>

<script>
export default {
    name: 'SysMsg',
    data() {
        return {
            error: false,
            msg: ""
        }
    },
    methods: {
        Info(msg) {
            this.error = false
            this.msg = msg
        },
        Error(msg) {
            this.error = true
            this.msg = msg
        }
    }
}
</script>

<style scoped>
    .error {
        font-color: red
    }
</style>
