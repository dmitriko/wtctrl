<template>
    <div></div>
</template>

<script>
export default {
    data() {
        return {
            error: false,
            msg: "sys messages go here"
        }
    },
    methods: {
        Info(msg) {
            this.error = false
            this.msg = msg
        }
    }
}
</script>
