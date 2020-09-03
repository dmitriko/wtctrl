<template>
  <q-page class="bg-secondary window-height window-width row justify-center items-start">
       <q-card square bordered class="q-ma-lg q-pa-lg shadow-1 window-width">
          <q-card-section>
            <q-form class="q-gutter-md">
              <q-input square filled clearable
                  label="email or phone"
                  v-model="loginKey"
                  class="q-ml-xl q-mr-xl" />
            </q-form>
          </q-card-section>
          <q-card-actions class="q-px-md">
            <q-btn unelevated color="light-green-7" size="lg" class="q-mx-auto"  label="Login" />
          </q-card-actions>
       </q-card>
 </q-page>
</template>

<script>
export default {
    name: 'Login',
    data() {
        return {
            loginKey: ""
        }
    },
    methods: {
        requestOtp() {
        },
        sendOtp() {
        }
    }
}
</script>
