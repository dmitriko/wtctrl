import Vue from 'vue'

export function handleOpen(context, event){
    console.log("handling open ws connection")
    context.commit('SOCKET_ONOPEN')
    context.dispatch('ui/SysMsgInfo', 'Connected to server!', {root: true})
}

export function handleEvent (context, event) {
    console.log("Handling event")
    console.table(event)
    if (event.type === "open") {
        return handleOpen(context, event)
    }
}

