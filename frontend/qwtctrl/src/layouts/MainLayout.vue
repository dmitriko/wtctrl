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
            <DrawerMenuItem symb="N" status="0" label="NEW" caption="Messages to process"  />
            <DrawerMenuItem symb="A" status="1" label="ARCHIVE" caption="Stored messages"  />
            <DrawerMenuItem symb="E" status="3" label="EXPORT" caption="Ready for export"  />
            <DrawerMenuItem symb="T" status="4" label="Trash" caption="Disapear in 30 days"  />

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
import DrawerMenuItem from 'components/DrawerMenuItem.vue'
export default {
  name: 'MainLayout',
    components: {DrawerMenuItem},
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
