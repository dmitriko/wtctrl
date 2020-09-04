<template>
  <q-page class="bg-secondary window-height window-width row justify-center items-start">
       <q-card square bordered class="q-ma-lg q-pa-lg shadow-1 window-width">
          <q-card-section>
            <q-form class="q-gutter-md" @submit="onSubmit()">
              <q-input square outlined clearable
                  label="email or phone"
                  v-model="loginKey"
                  style="font-size:2em"
                  class="q-ml-xl q-mr-xl" />
              <q-input square outlined
                  label="OTP"
                  v-model="otp"
                  style="font-size:2em;text-align:center;width:50%;"
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
export default {
    name: 'Login',
    data() {
        return {
            loginKey: "",
            otp: "",
            askOTP: false,
            btnLabel: "Request OTP"
        }
    },
    methods: {
        requestOtp() {
        },
        sendOtp() {
            this.$axios.post("https://app.wtctrl.com/login")
            .then((response) => {
                console.log(response.data)
            })
            .catch((err) => {
                console.log("could not POST " + err)
                })
        },
        onSubmit() {
            if (this.loginKey !== "") {
                this.btnLabel = 'Login'
                this.askOTP = true
            }
            if (this.otp !== "") {
                this.sendOtp()
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
