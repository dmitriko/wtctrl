class WSConn {
    constructor(store) {
        this.store = store
        this.connection = ""
        this.url = ""
    }
    connect(url) {
        this.url = url
        this.connection = new WebSocket(url)
        this.connection.onopen = this.eventHandler()
        this.connection.onclose = this.eventHandler()
        this.connection.onerror = this.eventHandler()
        this.connection.onmessage = this.eventHandler()
    }
    eventHandler() {
        return (event) => {
            this.store.dispatch("ws/handleEvent", event)
        }
    }
    send(obj) {
        if (this.connection !== "") {
            this.connection.send(JSON.stringify(obj))
        }
    }
}


export default  ({Vue, store}) => {
    Vue.prototype.$wsconn = new WSConn(store)
}