import Vue from 'vue'

export function handleOpen(context, event){
    context.commit('SOCKET_ONOPEN')
    context.dispatch('ui/SysMsgInfo', 'Connected to server!', {root: true})
}

export function handleClose(context, event) {
    context.commit('SOCKET_ONCLOSE')
    context.dispatch('ui/SysMsgInfo', 'Connection to server closed!', {root: true})
}

export function handleError(context, event) {
    context.commit('SOCKET_ONERROR')
    context.dispatch('ui/SysMsgError', 'Error connecting to server.', {root: true})
}

export function handleMessage(context, event) {
    context.commit('SOCKET_ONMESSAGE', event.data)
}

export function handleEvent (context, event) {
    console.log(event)
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

