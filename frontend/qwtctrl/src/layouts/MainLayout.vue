<template>
  <q-layout view="hHh lpR fFf">

    <q-header class="bg-primary text-white">
      <q-toolbar>
        <q-btn dense flat round icon="menu" @click="isDrawerOpen = !isDrawerOpen" />

        <q-toolbar-title>
         Trust with Web Tech Control
        </q-toolbar-title>
      </q-toolbar>
    </q-header>

    <q-drawer overlay v-model="isDrawerOpen" side="left" bordered>
        <q-scroll-area class="fit">
         <q-list padding>

            <q-item-label
                header
                class="text-grey-8 text-body-1"
             >
                Welcome, {{ userTitle }}
             </q-item-label>
            <q-separator class="q-my-sm" />

                <q-item clickable v-ripple :to="{name: 'ums', params:{status: '0'}}">
              <q-item-section avatar>
                 <q-avatar color="primary" text-color="white">
                   N
                 </q-avatar>
              </q-item-section>

              <q-item-section>
                  <q-item-label>NEW</q-item-label>
                  <q-item-label caption lines="1">Messages to process</q-item-label>
              </q-item-section>
            </q-item>
         </q-list>
        </q-scroll-area>
    </q-drawer>

    <q-page-container>
        <SysMsg />
      <router-view />
    </q-page-container>

  </q-layout>
</template>
<script>

export default {
  name: 'MainLayout',
  created() {
      this.$store.dispatch('ui/closeDrawer')
  },
  computed: {
      isDrawerOpen: {
          get() {
              return this.$store.state.ui.is_drawer_open
          },
          set(val) {
              this.$store.commit('ui/SET_DRAWER', val)
          }
      },
      userTitle () {
          return this.$store.state.login.title
      },
  },
  data () {
    return {
    }
  }
}
</script>
