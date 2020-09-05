import Vue from 'vue'
import Vuex from 'vuex'

// import example from './module-example'

import login from './login'
import ui from './ui'

Vue.use(Vuex)

/*
 * If not building with SSR mode, you can
 * directly export the Store instantiation;
 *
 * The function below can be async too; either use
 * async/await or return a Promise which resolves
 * with the Store instance.
 */

export default function (/* { ssrContext } */) {
  const Store = new Vuex.Store({
    modules: {
        login,
        ui
    },

    // enable strict mode (adds overhead!)
    // for dev mode only
    strict: process.env.DEV
  })

  if (process.env.DEV && module.hot) {
     module.hot.accept(['./ui'], () => {
      const newUI = require('./ui').default
      Store.hotUpdate({ modules: { ui: newUI } })
    })
    module.hot.accept(['./login'], () => {
      const newLogin = require('./login').default
      Store.hotUpdate({ modules: { login: newLogin } })
    })
  }

  return Store
}
