
export function openDrawer ({commit}) {
    commit("OPEN_DRAWER")
}

export function closeDrawer ({commit}) {
    commit("CLOSE_DRAWER")
}

export function SysMsgInfo({commit}, msg) {
    commit('SET_SYS_MSG_ERR', false)
    commit('SET_SYS_MSG', msg)
}

export function SysMsgError({commit}, msg) {
    commit('SET_SYS_MSG_ERR', true)
    commit('SET_SYS_MSG', msg)
}


