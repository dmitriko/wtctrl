export function setLoggedUser ({commit}, data) {
    commit('SET_LOGGED_USER', data)
}

export function setLoggedOut ({commit}, dummy) {
    commit('SET_LOGGED_OUT')
}
