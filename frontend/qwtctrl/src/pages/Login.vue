<template>
  <q-page class="bg-secondary window-height window-width row justify-center items-start">
       <q-card square bordered class="q-ma-lg q-pa-lg shadow-1 window-width">
          <q-card-section>
            <q-form class="q-gutter-md" @submit="onSubmit()">
              <q-input square outlined clearable
                  label="email or phone"
                  v-model="loginKey"
                  style="font-size:2em"
                  class="q-ml-md q-mr-md" />
              <q-input square outlined
                  label="OTP"
                  v-model="otp"
                  style="font-size:2em;text-align:center;width:8em;"
                  v-if="askOTP"
                  class="q-mx-auto" />
            </q-form>
          </q-card-section>
          <q-card-actions class="q-px-md">
            <q-btn @click="onSubmit()"
                unelevated color="light-green-7" size="lg" class="q-mx-auto"
                :label="btnLabel" />
          </q-card-actions>
       </q-card>
 </q-page>
</template>

<script>
import SysMsg from 'components/SysMsg.vue'
import { LocalStorage } from 'quasar'

export default {
    name: 'Login',
    created() {
        if (LocalStorage.has('loginKey')) {
            this.loginKey = LocalStorage.getItem('loginKey')
        }
    },
    data() {
        return {
            loginKey: "",
            otp: "",
            askOTP: false,
            btnLabel: "Request OTP",
            requestPK: "",
            title: "",
            token: "",
            userPK: ""
        }
    },
    methods: {
        requestOtp() {
          //  SysMsg.Info("Sending data...")
            this.$axios.post("https://app.wtctrl.com/reqotp", {"key": this.loginKey})
                    .then((response) => {
      //                  SysMsg.Info("Done.")
                        this.requestPK = response.data.request_pk
                        this.btnLabel = 'Login'
                        this.askOTP = true
                    })

        },
        sendOtp() {
        //    SysMsg.Info("Sending data...")
            this.$axios.post("https://app.wtctrl.com/login", {"request_pk": this.requestPK, "otp": this.otp})
            .then((response) => {
                if (response.data.ok) {
                    this.title = response.data.title
                    this.token = response.data.token
                    this.userPK = response.data.user_pk
                    //this.$refs.drawer.open()
                } else {
                    console.log(response.data)
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
                 LocalStorage.set('loginKey', this.loginKey)
                 this.requestOtp()
            }
       }
    }
}
</script>

<style scoped>
.otp-input {
    font-size: 4em
}

</style>
