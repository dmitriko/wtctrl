<template>
<q-page>
    <q-toolbar>
        <div class="row justify-center flex flex-center" style="width: 100%">
            <div class="col text-center" > Period, days: </div>
            <div class="col">
                <q-input dense outlined square
                    @keydown.enter="onDaysEnter"
                    @blur="onDaysEnter" :input-style="{width:'3em'}" v-model="days" />
            </div>
            <div class="col" style="text-align:center">
                <q-btn class="q-mx-auto" @click="reload()" color="secondary" label="reload" /></div>
            <div class="col-4">
                Get updates: <q-toggle @input="onSubscribeInput" v-model="subscribed" />
            </div>
        </div>
    </q-toolbar>
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
                <q-item-label class="text-subtitle-1" v-if="item.text">{{item.text}}</q-item-label>
            </q-item-section>
          </template>
          <MsgViewEdit @textUpdated="textUpdated" :item="item" />
       </q-expansion-item>
    </q-list>
</q-page>
</template>

<script>
import MsgViewEdit from 'components/MsgViewEdit.vue'

export default {
    name: 'ums',
    components: {MsgViewEdit},
    created() {
        this.restoreSettings()
        this.fetchMsgs()
        this.subscr()
    },
     data() {
        return {
            days: 21,
            subscribed: false,
            msg_lists: {},  // UMS as key, array of objects is a value
            msgs_store: {}, //msg.pk a key, full msg a value
            selected:[]
        }
    },
    watch: {
        'isOnline': function(isOnline) {
            if (!isOnline) {console.log("browser went offline")}
            else {console.log("browser went online")}
            if (isOnline && !this.$store.state.ws.isConnected) {
                console.log("websocket is closed, reconnecting")
                this.$store.dispatch('ws/clearReconnectCount')
                this.$store.dispatch('ws/reconnect')
            }
        },
        '$store.state.ws.isConnected': function(isConnected) {
            if (isConnected) {
                 this.subscr()
                 this.fetchMsgs()
            }
        },
        '$store.state.ws.message': function(msg) {
            if (msg.name === 'msg_index') {
                this.$wsconn.send({'name':'fetchmsg', 'pk': msg.pk})
                return
            }
            let expected_kind = [1, 2, 3].includes(msg.kind)
            if (msg.name === 'imsg' && expected_kind) {
                console.log('got new item')
                let items = this.msg_lists[this.id()]
                if (items === undefined) {
                        items = []
                }
                 for (let i=0; i < items.length; i++) {
                    if (items[i].pk == msg.pk) {
                        console.log('item is in the list')
                        if (items[i].updated < msg.updated) {
                            this.$set(items, i, msg)
                            console.log('updating it in place')
                        }
                    return
                    }
                }
                items.push(msg)
                items.sort(function(a, b){return b.created-a.created})
                this.$set(this.msg_lists, this.id(), items)
                return
            }
            if (msg.name === 'dbevent' && expected_kind) {
                this.$wsconn.send({'name':'fetchmsg', 'pk': msg.pk})
                return
            }
        },
    },
    computed: {
        items: {
            get () {
              if (this.msg_lists[this.id()] === undefined) {
                    return []
               }
               return this.msg_lists[this.id()]
            },
            set (val) {
               this.$set(this.msg_lists, this.id(), val)
            }
        }
    },
    methods: {
       textUpdated(pk, text) {
           this.$wsconn.send({
               name: "msgupdate",
               pk: pk,
               key: "text",
               value: text,
               id: "someuuid",
           })
       },
       restoreSettings() {
        if (this.$q.localStorage.has(this.id())) {
            let item = this.$q.localStorage.getItem(this.id())
            this.days = item.days
            this.subscribed = item.subscribed
        } else {
            this.$q.localStorage.set(this.id(), {
                "days": 21,
                "subscribed": false
            })
        }
       },

        id () {
            return this.$store.state.login.userPK + '#' + this.$route.params.status
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
                'status': parseInt(this.$route.params.status)
            })
        },
        onDaysSet() {
            this.items = []
            this.fetchMsgs()
        },
        onDaysEnter() {
            let item = this.$q.localStorage.getItem(this.id())
            if (item === null) {
                 item = {}
                 item.days = 0
            }
            if (item.days != this.days) {
                this.onDaysSet()
                item.days = this.days
                this.$q.localStorage.set(this.id(), item)
            }
        },
        subscr() {
            // Subscribe or unsubscribe
            if (this.subscribed) {
                this.$wsconn.send({
                    'name':'subscr',
                    'umspk': this.$store.state.login.userPK,
                    'status': parseInt(this.$route.params.status)
                })
            } else {
                this.$wsconn.send({
                    'name':'unsubscr',
                    'umspk': 'this.$store.state.login.userPK',
                    'status': parseInt(this.$route.params.status)
                })
            }
        },
        onSubscribeInput () {
            let item = this.$q.localStorage.getItem(this.id())
            if (item === null) {
                item = {}
            }
            item.subscribed = this.subscribed
            this.$q.localStorage.set(this.id(), item)
            this.subscr()
        },
        reload () {
            console.log("reloading")
            this.items = []
            this.fetchMsgs()
        }
    }
}
</script>
