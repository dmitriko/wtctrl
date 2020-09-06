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

    <q-drawer overlay behavior="desktop" v-model="isDrawerOpen" side="left" bordered>
        <q-list>
        <q-item-label
          header
          class="text-grey-8 text-body-1"
        >
            Welcome, {{ userTitle }}
        </q-item-label>
        </q-list>
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
