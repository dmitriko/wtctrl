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

    <q-drawer overlay v-model="isDrawerOpen" side="left" bordered v-if="isLoggedIn">
        <q-scroll-area class="fit">
         <q-list padding>

            <q-item-label
                header
                class="text-grey-8 text-body-1"
             >
                Welcome, {{ userTitle }}
             </q-item-label>

                <q-separator class="q-my-sm" />

            <q-item clickable v-ripple to="/">
              <q-item-section avatar>
                 <q-avatar color="primary" text-color="white">
                     M
                 </q-avatar>
              </q-item-section>

              <q-item-section>
                  <q-item-label>Messages</q-item-label>
              </q-item-section>
            </q-item>

            <q-item clickable v-ripple :to="{name: 'profile'}">
              <q-item-section avatar>
                 <q-avatar color="primary" text-color="white">
                     P
                 </q-avatar>
              </q-item-section>

              <q-item-section>
                  <q-item-label>Profile</q-item-label>
              </q-item-section>
            </q-item>
            <q-separator class="q-my-sm" />
                <q-item class="justify-center flex"><q-btn @click="logout()" label="Logout" /></q-item>
          </q-list>
        </q-scroll-area>
    </q-drawer>

    <q-page-container v-if="isLoggedIn">
        <SysMsg />
      <router-view />
    </q-page-container>

    <Login v-if="!isLoggedIn" @onLoggedIn="loggedIn($event)"/>

  </q-layout>
</template>


<script>
import MsgViewEdit from 'components/MsgViewEdit.vue'
import Login from 'components/Login.vue'

export default {
  name: 'MainLayout',
  components: {MsgViewEdit, Login},
  created() {
      this.$store.dispatch('ui/closeDrawer')
      if (this.$q.localStorage.has('loginUser')) {
            let item = this.$q.localStorage.getItem('loginUser')
            this.loggedIn(item)
      }
      if (this.isLoggedIn && this.$route.params.folder === undefined) {
          let folders = this.$store.state.login.folders
          if (folders !== undefined && folders.length > 0) {
            this.$router.push({name:'msg', params: {"folder": folders[0].ums}})
          }
      }

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
      isLoggedIn() {
          return this.$store.state.login.isLoggedIn
      },
 },
  methods: {
        loggedIn(data) {
            console.log("in loggedIn")
            console.log(data)
            this.$q.localStorage.set('loginUser', {
                "token": data.token,
                "title": data.title,
                "folders": data.folders,
                "user_pk": data.user_pk,
                "created":data.created})
            this.$wsconn.connect(this.ws_api_url + '?token=' + data.token)
            this.$store.dispatch('ui/SysMsgInfo', 'Welcome, ' + data.title)
            this.$store.dispatch('login/setLoggedUser', data)
       },
       logout() {
          console.log("logging out")
          this.$store.dispatch("login/setLoggedOut")
          this.$store.dispatch('ui/closeDrawer')
          this.$wsconn.close()
      },
  },
  data () {
    return {
        ws_api_url: "wss://io2hsa5u5a.execute-api.us-west-2.amazonaws.com/prod1"
    }
  }
}
</script>
