<template>
<q-page>
    <q-markup-table dense>
        <thead>
            <tr>
                <th colspan="5"></th>
            </tr>
        </thead>
        <tbody>
            <tr>
                <td>List of messages from</td>
                <td colspan="3">
                    <q-select
                        filled
                        dense
                        v-model="selectedFolder"
                        :options="folders"
                        option-value="ums"
                        option-label="title"
                        emit-value
                        map-options
                        style="min-width: 250px; max-width: 300px"
                    />
                </td>
                <td>
                 <q-btn icon="refresh" class="q-mx-auto" @click="reload()" color="secondary"  />
                </td>
            </tr>
            <tr v-if="currentFolder.kind==6">
                <td>
                    Period, days.
                </td>
                <td>
                    <q-input dense outlined square
                    @keydown.enter="onDaysEnter"
                    @blur="onDaysEnter" :input-style="{width:'3em'}" v-model="days" />
                </td>
                <td colspan="3"></td>
            </tr>
            <tr v-if="currentFolder.kind==7">
                <td>
                    Period, from, to.
                </td>
                <td colspan="2">
                    <div style="width:180px">
                    <q-input dense filled v-model="periodStarts" mask="date" :rules="['date']">
                        <template v-slot:append>
                            <q-icon name="event" class="cursor-pointer">
                            <q-popup-proxy ref="qDateProxy" transition-show="scale" transition-hide="scale">
                             <q-date v-model="periodStarts">
                                <div class="row items-center justify-end">
                                 <q-btn v-close-popup label="Close" color="primary" flat />
                                </div>
                            </q-date>
                            </q-popup-proxy>
                            </q-icon>
                         </template>
                         </q-input>
                    </div>
                </td>
                <td colspan="2">
                    <div style="width:180px">
                    <q-input dense filled v-model="periodEnds" mask="date" :rules="['date']">
                        <template v-slot:append>
                            <q-icon name="event" class="cursor-pointer">
                            <q-popup-proxy ref="qDateProxy" transition-show="scale" transition-hide="scale">
                             <q-date v-model="periodEnds">
                                <div class="row items-center justify-end">
                                 <q-btn v-close-popup label="Close" color="primary" flat />
                                </div>
                            </q-date>
                            </q-popup-proxy>
                            </q-icon>
                         </template>
                         </q-input>
                         </div>
                    </div>
                </td>
            </tr>
        </tbody>
    </q-markup-table>

</q-page>
</template>
<script>
import MsgViewEdit from 'components/MsgViewEdit.vue'

export default {
    name: 'MsgList',
    components: {MsgViewEdit},
    data() {
        return {
            selectedFolder:"",
            days: 7,
            periodStarts:"",
            periodEnds:""
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
        if (this.$route.params.folder === undefined &&  this.$store.state.login.folders !== undefined) {
            if (this.$store.state.login.folders.length > 0) {
                this.$router.push({name:'msg', params:{folder: this.$store.state.login.folders[0]}});

            }
        }
        this.selectedFolder = this.currentUMS
    },
    methods: {
        onDaysEnter() {
            console.log('on days enter')
        },
        reload() {
            console.log("reloading")
        }

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
