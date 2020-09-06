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
      <!-- drawer content -->
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
      }
  },
  data () {
    return {
    }
  }
}
</script>
