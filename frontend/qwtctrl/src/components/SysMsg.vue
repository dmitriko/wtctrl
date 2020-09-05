<template>
    <div class="q-pa-md q-ml-md" ><span :class="error ? 'error' : '' " >{{ msg }}</span></div>
</template>

<script>
export default {
    name: 'SysMsg',
    created() {
        this.$root.$on("sys-msg-info", this.Info),
        this.$root.$on("sys-msg-error", this.Error)
    },
    beforeDestroy() {
        this.$root.$off("sys-msg-info", this.Info),
        this.$root.$off("sys-msg-error", this.Error)
    },
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
        color: red
    }
</style>
