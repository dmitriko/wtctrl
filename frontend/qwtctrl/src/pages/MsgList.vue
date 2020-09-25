<template>
<q-page>
    <q-toolbar><span v-if="currentFolder">List of messages from {{ currentFolder.title }}</span>
        <span v-else>empty</span>
    </q-toolbar>
</q-page>
</template>
<script>
import MsgViewEdit from 'components/MsgViewEdit.vue'

export default {
    name: 'MsgList',
    components: {MsgViewEdit},
    created() {
        console.log('in created MsgList')
        console.log(this.$store.state.login)
        console.log(this.folders)
        if (this.$route.params.folder == undefined &&  this.$store.state.login.folders !== undefined) {
            if (this.$store.state.login.folders.length > 0) {
                this.$router.push({name:'msg', params:{folder: this.$store.state.login.folders[0]}});

            }
        }
    },
    computed: {
        folders() {
          if (this.$store.state.login.folders !== undefined && this.$store.state.login.folders.length >0 ) {
              return this.$store.state.login.folders
          }
          return []
      },
        currentFolder() {
          for (let i=0; i<this.folders.length; i++) {
              console.log(this.folders[i].ums)
              console.log(this.currentUMS)
              if (this.folders[i].ums = this.currentUMS) {
                  return this.folders[i]
              }
          }
          return undefined
      },
        currentUMS() {
          return this.$route.params.folder
      }
 },

}
</script>
