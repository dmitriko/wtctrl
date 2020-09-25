<template>
<q-page>
    <q-toolbar>List of messages from <q-select
                filled
                v-model="selectedFolder"
                :options="folders"
                option-value="ums"
                option-label="title"
                emit-value
                map-options
                style="min-width: 250px; max-width: 300px"
                />
    </q-toolbar>
</q-page>
</template>
<script>
import MsgViewEdit from 'components/MsgViewEdit.vue'

export default {
    name: 'MsgList',
    components: {MsgViewEdit},
    data() {
        return {
            selectedFolder:""
        }
    },
    watch: {
        selectedFolder: function(val) {
            if (this.currentUMS !== val) {
                this.$router.push({name: 'msg', params: {folder: val}})
            }
        }
    },
    created() {
        if (this.$route.params.folder == undefined &&  this.$store.state.login.folders !== undefined) {
            if (this.$store.state.login.folders.length > 0) {
                this.$router.push({name:'msg', params:{folder: this.$store.state.login.folders[0]}});

            }
        }
        this.selectedFolder = this.currentUMS
    },
    computed: {
        folders() {
          let result = new Array()
          let folders = this.$store.state.login.folders
          if (folders !== undefined ) {
              for (let i=0; i < folders.length; i++) {
                  let f = {}
                  for (let k in folders[i]) f[k] = folders[i][k]
                  result.push(f)
              }
          }
          return result
      },
        currentFolder() {
          for (let i=0; i<this.folders.length; i++) {
              if (this.folders[i].ums === this.currentUMS) {
                  return this.folders[i]
              }
          }
          return undefined
      },
        currentUMS()  {
                return this.$route.params.folder
    },


 },

}
</script>
