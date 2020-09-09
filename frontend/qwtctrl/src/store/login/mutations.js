export function SET_LOGGED_OUT (state) {
    state.isLoggedIn = false
    state.title = ""
    state.userPK = ""
    state.token = ""
    state.created = 0
}


export function SET_LOGGED_USER (state, data) {
    state.isLoggedIn = true
    state.title = data.title
    state.userPK = data.user_pk
    state.token = data.token
    state.created = data.created
}
