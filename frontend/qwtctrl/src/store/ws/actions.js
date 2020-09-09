import Vue from 'vue'
import {LocalStorage} from 'quasar'

export function handleOpen(context, event){
    context.commit('SOCKET_ONOPEN')
    context.dispatch('ui/SysMsgInfo', 'Connected to server!', {root: true})
}

export function handleClose(context, event) {
    context.dispatch('ui/SysMsgInfo', 'Connection to server closed!', {root: true})
    if (!navigator.onLine) {
        return
    }
    let isExpired = ((~~(Date.now() / 1000) - context.rootState.login.created) > 24*60*59)
    if (isExpired) {
        context.dispatch('login/setLoggedOut', 'dummy', {root: true})
        if (LocalStorage.has('loginUser')) {
            LocalStorage.remove('loginUser')
         }
    }
    if (contest.state.reconnectAttempts < 6) {
        context.commit('RECONNECT_ATTEMPT')
        context.dispatch('ui/SysMsgInfo', 'Reconnecting...', {root: true})
        Vue.prototype.$wsconn.reconnect()
    }
    context.commit('SOCKET_ONCLOSE')
}

export function handleError(context, event) {
    console.log("websocket error")
    console.log(event)
    context.commit('SOCKET_ONERROR')
    context.dispatch('ui/SysMsgError', 'Error connecting to server.', {root: true})
}

export function handleMessage(context, event) {
    let msg = JSON.parse(event.data)
    context.commit('SOCKET_ONMESSAGE', msg)
}

export function handleEvent (context, event) {
    if (event.type === "open") {
        return handleOpen(context, event)
    }
    if (event.type === "error") {
        return handleError(context, event)
    }
    if (event.type === "close") {
        return handleClose(context, event)
    }

    if (event.type === "message") {
        return handleMessage(context, event)
    }
}

