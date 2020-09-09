<template>
     <q-item clickable v-ripple :to="{name: 'ums', params:{status: status}}">
              <q-item-section avatar>
                 <q-avatar color="primary" text-color="white">
                     {{symb}}
                 </q-avatar>
              </q-item-section>

              <q-item-section>
                  <q-item-label>{{label}}</q-item-label>
                  <q-item-label caption lines="1">{{caption}}</q-item-label>
              </q-item-section>
            </q-item>

</template>

<script>
export default {
    name: 'DrawerMenuItem',
    props: ['status', 'label', 'caption', 'symb']
}
</script>
