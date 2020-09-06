import Vue from 'vue'

export function SOCKET_ONOPEN (state)  {
      state.isConnected = true
}

export function SOCKET_ONCLOSE (state, event)  {
      state.socket.isConnected = false
}

export function SOCKET_ONERROR (state, event)  {
      console.error(state, event)
}

export function SOCKET_ONMESSAGE (state, message)  {
      state.socket.message = message
}

export function SOCKET_RECONNECT(state, count) {
      console.info(state, count)
}

export function SOCKET_RECONNECT_ERROR(state) {
      state.socket.reconnectError = true;
}
