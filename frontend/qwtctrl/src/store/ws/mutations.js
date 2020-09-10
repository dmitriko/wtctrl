import Vue from 'vue'
export function RECONNECT_ATTEMPT (state) {
    state.reconnectCount = state.reconnectCount + 1
}

export function CLEAR_RECONNECT_COUNT (state) {
    state.reconnectCount = 0
}

export function SOCKET_ONOPEN (state)  {
      state.isConnected = true
      state.reconnectCount = 0
}

export function SOCKET_ONCLOSE (state, event)  {
      state.isConnected = false
}

export function SOCKET_ONERROR (state, event)  {
      console.error(state, event)
}

export function SOCKET_ONMESSAGE (state, message)  {
//      Vue.set(state, 'message', message)
        console.log(message)
        state.message = message
}


