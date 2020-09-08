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
            <div class="col"></div>
            <div class="col-4">
                Get updates: <q-toggle @input="onSubscribeInput" v-model="subscribed" />
            </div>
        </div>
    </q-toolbar>
    <q-list bordered separator style="max-width:520px" class="q-mx-auto">
        <q-expansion-item v-for="item in items" :key="item.pk" class="q-mx-auto">
          <template v-slot:header>
            <q-item-section>
                <q-img v-if="item.files.thumb" style="height: 320px; max-width: 320px"
                        :src="item.files.thumb.url"  />
                <audio v-if="item.files.voice"
                        class="q-mx-auto"
                        :src="item.files.voice.url"
                        controls type="audio/ogg; codecs=opus" />
                <q-item-label class="text-subtitle-1" v-if="item.text">{{item.text}}</q-item-label>
            </q-item-section>
          </template>
          <q-card>
          <q-card-section>{{item.pk}}</q-card-section>
          <q-card-section>
              <q-btn label="Foo" />
          </q-card-section>
        </q-card>
        </q-expansion-item>
    </q-list>
</q-page>
</template>

<script>
import MsgView from 'components/MsgView.vue'

export default {
    name: 'ums',
    components: {MsgView},
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
            items:[]
        }
    },
    watch: {
        '$store.state.ws.message': function(msg) {
            if (msg.name === 'msg_index') {
                this.$wsconn.send({'name':'fetchmsg', 'pk': msg.pk})
                return
            }
            let expected_kind = [2, 3].includes(msg.kind)
            if (msg.name === 'imsg' && expected_kind) {
                this.items.unshift(msg)
                this.items.sort(function(a, b){return b.created-a.created})
                return
            }
            if (msg.name === 'dbevent' && expected_kind) {
                this.$wsconn.send({'name':'fetchmsg', 'pk': msg.pk})
                return
            }
        },
    },
    computed: {
    },
    methods: {
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
        fetchMsgs() {
            //{"name":"msgfetchbydays", "id":"somerandom", "days":20, "status":0, "desc":true}
            this.items = []
            this.$wsconn.send({
                'name': 'msgfetchbydays',
                'id':'foo',
                'days': parseInt(this.days),
                'status': parseInt(this.$route.params.status)
            })
        },
        onDaysSet() {
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
        }
    }
}
</script>
