<template>
<q-page>
    <q-toolbar>List of messages
    </q-toolbar>
</q-page>
</template>
<script>
import MsgViewEdit from 'components/MsgViewEdit.vue'

export default {
    name: 'MsgList',
    components: {MsgViewEdit},
    created() {
        console.log('in created')
        console.log(this.$store.state.login)
        if (this.$route.params.folder == undefined &&  this.$store.state.login.folders !== undefined) {
            if (this.$store.state.login.folders.length > 0) {
               this.$router.push(this.$store.state.login.folders[0]);
            }
        }

   },
}
</script>
