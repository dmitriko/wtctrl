<template>
<q-page >
    <div full-width class="row" style="border:1px solid">
        <div class="col">Folder:</div>
        <div class="col"><q-select
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
        </div>
        <div class="col">
                 <q-btn icon="refresh"  @click="reload()" color="secondary"  />
        </div>
    </div>
    <div class="row justify-start" v-if="currentFolder.kind==6">
                <div class="col">
                    Period, days.
                </div>
                <div class="col" >
                    <q-input dense outlined square
                    @keydown.enter="onDaysEnter"
                    @blur="onDaysEnter" :input-style="{width:'3em'}" v-model="days" />
                </div>
            </div>
            <div class="row" v-if="currentFolder.kind==7">
                    Period, from, to.
                    <div class="col" style="width:180px">
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
                    <div class="col" style="width:180px">
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
    <q-list bordered separator style="max-width:520px" class="q-mx-auto">
        <q-expansion-item v-for="item in items" :key="item.pk" class="q-mx-auto">
          <template v-slot:header>
            <q-item-section avatar>
              <q-checkbox v-model="selected" :val="item.pk" color="secondary" />
            </q-item-section>
            <q-item-section>
                <q-img v-if="item.files.thumb" style="height: 320px; max-width: 320px"
                                               :src="item.files.thumb.url" />
                <audio v-if="item.files.voice"
                        class="q-mx-auto"
                        :src="item.files.voice.url"
                        controls type="audio/ogg; codecs=opus" />
                    <a :href="item.files.bigpic.url" v-if="item.files.bigpic" target="_blank" >big</a>
                <q-item-label class="text-subtitle-1" v-if="item.text">{{item.text}}</q-item-label>
            </q-item-section>
          </template>
          <MsgViewEdit @textUpdated="textUpdated" @umsSet="msgSetUMS" :item="item" :status="$route.params.status" />
       </q-expansion-item>
    </q-list>

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
            selected: [],
            days: 7,
            msg_lists: {},  // UMS as key, array of objects is a value
            uiSettings: {},
            commonFetchStatus: {}  //UMS key, bool a value

        }
    },
    watch: {
        '$route': function(value) {
            if (this.currentFolder.kind===6) {
                let currentSettings = this.uiSettings[this.currentUMS]
                if (currentSettings !== undefined && currentSettings.days !== undefined) {
                    this.days = currentSettings.days
                }
                if (!this.fetchStatus.fetched) {
                    this.fetchMsgs()
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
        if (uiSettings !== null) {
            this.uiSettings = uiSettings
            this.days = uiSettings[this.currentUMS].days
        }
    },
   methods: {
        saveSettings() {
            this.$q.localStorage.set('uiSettings', this.uiSettings)
         },
        onDaysEnter() {
            if (this.uiSettings[this.currentUMS] === undefined) {
                this.uiSettings[this.currentUMS] = {}
            }
            this.$set(this.uiSettings[this.currentUMS], 'days', this.days)
            this.saveSettings()
         //   this.fetchMsgs()
        },
        reload() {
            this.fetchMsgs()
        },
        msgSetUMS(ums) {
            console.log('changing msg ums')
        },
        async fetchMsgs() {
            let started = ~~(Date.now() / 1000)
            while (this.$wsconn.connection.readyState != 1) {
                if ((~~(Date.now() / 1000) - started) > 10) {
                    console.log('failed to fetch data')
                    return
                }
                await new Promise(r => setTimeout(r, 1000));
            }
            this.$wsconn.send({
                'name': 'msgfetchbydays',
                'id':'foo',
                'days': parseInt(this.days),
                'ums': this.currentUMS
            })
            this.fetchStatus.fetched = true
        },
        textUpdated(pk, text) {
           this.$wsconn.send({
               name: "msgupdate",
               pk: pk,
               key: "text",
               value: text,
               id: "someuuid",
           })
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
              if (this.msg_lists[this.currentUMS] === undefined) {
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
                if  (this.$route.params.folder !== undefined) {
                    return this.$route.params.folder
                }
                return ""
            },
            set(val) {
                this.$router.push({name: 'msg', params: {folder: val}})
            }
      },
    currentSettings: {
            get() {
                this.uiSettings[this.currentUMS]
            },
            set(val) {
                if (this.uiSettings === null) {
                    this.uiSettings = {}
                }
                this.uiSettings[this.currentUMS] = val
                this.$q.localStorage.set('uiSettings', this.uiSettings)
            }
        },
    },

}
</script>
