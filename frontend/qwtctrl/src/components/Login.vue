<template>
    <q-page-container>
  <q-page class="bg-secondary window-height window-width row justify-center items-start">
       <q-card square bordered class="q-ma-lg q-pa-lg shadow-1 window-width">
          <q-card-section>
            <q-form class="q-gutter-md" @submit="onSubmit()">
              <q-input square outlined clearable
                  label="email or phone"
                  v-model="loginKey"
                  style="font-size:1.5em"
                  class="q-ml-md q-mr-md" />
              <q-input square outlined
                  label="OTP"
                  @keydown.enter="onSubmit()"
                  v-model="otp"
                  style="font-size:1.5em;text-align:center;width:8em;"
                  v-if="askOTP"
                  class="q-mx-auto" />
            </q-form>
          </q-card-section>
          <q-card-actions class="q-px-md">
            <q-btn @click="onSubmit()"
                unelevated color="light-green-7" size="lg" class="q-mx-auto"
                :label="btnLabel" :disabled="btnDisabled" />
          </q-card-actions>
       </q-card>
 </q-page>
    </q-page-container>
</template>

<script>
//import { LocalStorage } from 'quasar'

export default {
    name: 'Login',
    created() {
        if (this.$q.localStorage.has('loginKey')) {
            this.loginKey = this.$q.localStorage.getItem('loginKey')
        }
    },
    data() {
        return {
            loginKey: "",
            otp: "",
            askOTP: false,
            btnLabel: "Request OTP",
            btnDisabled: false,
            requestPK: "",
       }
    },
    methods: {
        requestOtp() {
            this.SysInfo('Sending data...')
            this.btnDisabled = true
            this.btnLabel = "Loading..."
            this.$axios.post("https://app.wtctrl.com/reqotp", {"key": this.loginKey})
                    .then((response) => {
                        if (response.data.ok) {
                            this.SysInfo('Done.')
                            this.requestPK = response.data.request_pk
                            this.btnLabel = 'Login'
                            this.btnDisabled = false
                            this.askOTP = true
                        } else {
                            this.SysInfo(response.data.error)
                        }
                    }).catch((error) => {
                        this.btnLabel = 'Login'
                        this.btnDisabled = false
                        this.SysInfo(error)})

        },
       sendOtp() {
            this.SysInfo('Sending data...')
            this.btnDisabled = true
            this.btnLabel = "Loading..."
            this.$axios.post("https://app.wtctrl.com/login", {"request_pk": this.requestPK, "otp": this.otp})
            .then((response) => {
                this.btnDisabled = false
                this.btnLabel = "Login"
                if (response.data.ok) {
                    console.log('OK from login')
                    this.$emit('onLoggedIn', response.data)
                } else {
                    this.btnDisabled = false
                    this.btnLabel = "Login"
                    this.SysError(response.data.error)
                }
            })
            .catch((err) => {
                console.log("could not POST " + err)
                })
        },
        onSubmit() {
            if (this.otp !== "" && this.requestPK !== "") {
                this.sendOtp()
                return
            }
             if (this.loginKey !== "") {
                 this.$q.localStorage.set('loginKey', this.loginKey)
                 this.requestOtp()
            }
       },
        SysInfo(msg) {
            this.$store.dispatch('ui/SysMsgInfo', msg)
        },
        SysError(msg) {
            this.$store.dispatch('ui/SysMsgError', msg)
        }
    }
}
</script>

<style scoped>
.otp-input {
    font-size: 4em
}

</style>
