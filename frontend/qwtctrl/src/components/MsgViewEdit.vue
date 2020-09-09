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
              <q-btn label="Foo" />
        </q-card-section>
    </q-card>

</template>
<script>
export default {
    name: 'MsgViewEdit',
    created() {
        console.log('in msg edit created')
    },
    data() {
        return {
            text: this.item.text
        }
    },
    props: {
        "item": {required: true}
    }
}
</script>
