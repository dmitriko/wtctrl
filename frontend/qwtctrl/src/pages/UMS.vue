<template>
<q-page>
    <q-toolbar>
        <div class="row justify-center flex flex-center" style="width: 100%">
            <div class="col text-center" > Period, days: </div>
            <div class="col">
                <q-input dense outlined square @blur="onDaysBlur" :input-style="{width:'3em'}" v-model="days" />
            </div>
            <div class="col"></div>
            <div class="col-4">
                Get updates: <q-toggle @input="onSubscribeInput" v-model="subscribed" />
            </div>
        </div>
    </q-toolbar>
    <div>{{id()}}</div>
    <ul>
        <li v-for="item in items">{{item}}</li>
    </ul>
</q-page>
</template>

<script>
export default {
    name: 'ums',
    created() {
        this.restoreSettings()
    },
     data() {
        return {
            days: 21,
            subscribed: false,
            msg_lists: {},  // UMS as key, array of objects is a value
            msgs_store: {}, //msg.PK a key, full msg a value
            items:[]
        }
    },
    watch: {
        '$store.state.ws.message': function(msg) {
            console.log('got message from sever in component')
            console.log(msg)
            if (msg.name === 'msg_index') {
                this.$wsconn.send({'name':'fetchmsg', 'pk': msg.PK})
                return
            }
            if (msg.name === 'imsg' && msg.kind === 3) {
                this.items.push(msg)
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
        onDaysBlur() {
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
        onSubscribeInput () {
            let item = this.$q.localStorage.getItem(this.id())
            if (item === null) {
                item = {}
            }
            item.subscribed = this.subscribed
            this.$q.localStorage.set(this.id(), item)
        }
    }
}
</script>
