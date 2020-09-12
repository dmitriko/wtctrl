<template>

    <q-card>
        <q-card-section>{{item.pk}}</q-card-section>
        <q-card-section>
            <q-input
                v-model="text"
                filled
                autogrow
             />
        </q-card-section>
        <q-card-section>
              <q-btn label="Save Text" @click="saveText" />
        </q-card-section>
        <q-card-section>
            <q-btn v-if="status !== '2'" label="To Export" @click="setUMS('2')" />
        </q-card-section>
    </q-card>

</template>
<script>
export default {
    name: 'MsgViewEdit',
    created() {
    },
    methods: {
        saveText() {
            this.$emit("textUpdated", this.item.pk, this.text)
        },
        setUMS(newStatus) {
            let ums = this.item.author + '#' + newStatus
            this.$emit('umsSet', this.item, ums)
        }
    },
    data() {
        return {
            text: this.item.text
        }
    },
    props: {
        "item": {required: true},
        "status": {required: true}
    }
}
</script>
