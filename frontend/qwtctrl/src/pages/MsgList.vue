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
                        v-model="currentUMS"
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
            periodStarts:"",
            periodEnds:"",
            msg_lists: {},  // UMS as key, array of objects is a value
            uiSettings: {},
            commonFetchStatus: {}  //UMS key, bool a value

        }
    },
    watch: {
        '$route': function(value) {
            if (this.currentFolder.kind===6){
                if (!this.fetchStatus.fetched) {
                    console.log('fetching msgs')
                    this.fetchStatus.fetched = true
                }
                if (!this.fetchStatus.subscribed) {
                    console.log('subscribing')
                    this.fetchStatus.subscribed = true
                }
            }
        },
        '$store.state.ws.message': function(msg) {
            if (msg.name === 'msg_index') {
                this.$wsconn.send({'name':'fetchmsg', 'pk': msg.pk})
                return
            }
            let expected_kind = [1, 2, 3].includes(msg.kind)
            if (msg.name === 'imsg' && expected_kind) {
                this.msg_push(msg)
            }
            if (msg.name === 'dbevent' && expected_kind) {
                this.$wsconn.send({'name':'fetchmsg', 'pk': msg.pk})
                return
            }
        },
    },
    created() {
        if (this.$route.params.folder === undefined &&  this.$store.state.login.folders !== undefined) {
            if (this.$store.state.login.folders.length > 0) {
                this.$router.push({name:'msg', params:{folder: this.$store.state.login.folders[0]}});

            }
        }
        let uiSettings = this.$q.localStorage.getItem('uiSettings')
        if (uiSettings !== undefined) {
            this.uiSettings = uiSettings
        }
    },
    methods: {
        onDaysEnter() {
            console.log('on days enter')
        },
        reload() {
            console.log("reloading")
        },
        msg_push(store_msg) {
           let msg = {}
            for(let k in store_msg) msg[k] = store_msg[k]
            let items = this.items
            for (let i=0; i < items.length; i++) {
                if (items[i].pk === msg.pk) {
                    if (items[i].updated < msg.updated) {
                        this.$set(items, i, msg)
                    }
                return
                }
            }
            items.push(msg)
            items.sort(function(a, b){return b.created-a.created})
            this.items = items
            return
       },

    },
    computed: {
        items: {
            get () {
              if (this.msg_lists[this.id()] === undefined) {
                  return []
               }
               return this.msg_lists[this.currentUMS]
            },
            set (val) {
               this.$set(this.msg_lists, this.currentUMS, val)
            }
        },
        fetchStatus: { //Status of fetching data for given folder
            get () {
                if (this.commonFetchStatus === null) {
                    this.commonFetchStatus = {}
                }
                if (this.commonFetchStatus[this.currentUMS] === undefined) {
                    let status = {}
                    status.fetched = false
                    status.subscribed = false
                    this.commonFetchStatus[this.currentUMS] = status
                }
                return this.commonFetchStatus[this.currentUMS]
            },
            set (val) {
                return this.commonFetchStatus[this.currentUMS] = val
            },
        },
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
        currentUMS:  {
            get() {
                return this.$route.params.folder
            },
            set(val) {
                this.$router.push({name: 'msg', params: {folder: val}})
            }
    },
        days: {
            get() {
                let days = this.currentSettings.days
                if (days === undefined) {
                    // one more try
                    days = this.currentSettings.days
                    if (days === undefined) return 7
                }
                return days
            },
            set(val) {
                let settings = this.currentSettings
                settings.days = val
                this.currentSettings = settings
                this.$q.localStorage.set('uiSettings', this.uiSettings)
            }
        },
        currentSettings: {
            get() {
                if (this.uiSettings === null) return {}
                let settings = this.uiSettings[this.currentUMS]
                if (settings === null || settings === undefined) {
                    return {}
                }
                return settings
            },
            set(val) {
                if (this.uiSettings === null) {
                    this.uiSettings = {}
                }
                this.uiSettings[this.currentUMS] = val
            }
        },
    },

}
</script>
