class WSConn {
    constructor(store) {
        this.store = store
        this.connection = ""
        this.url = ""
    }
    connect(url, pinger=true) {
        this.url = url
        this.connection = new WebSocket(url)
        this.connection.onopen = this.eventHandler()
        this.connection.onclose = this.eventHandler()
        this.connection.onerror = this.eventHandler()
        this.connection.onmessage = this.eventHandler()
        if (pinger) {
            setInterval(this.pinger(), 3 * 60 * 1000)
        }
    }
    reconnect() {
        if (this.url !== ""){
            this.connect(this.url, false)
        }
    }
    close() {
        if (this.connection !== "") {
            this.connection.close()
            this.connection = ""
        }
    }
    pinger() {
        return () => {
            this.connection.send(JSON.stringify({"name": "ping"}))
        }
    }
    eventHandler() {
        return (event) => {
            this.store.dispatch("ws/handleEvent", event)
        }
    }
    send(obj) {
        console.log(obj)
        if (this.connection !== "") {
            this.connection.send(JSON.stringify(obj))
        }
    }
}


export default  ({Vue, store}) => {
    Vue.prototype.$wsconn = new WSConn(store)
}
