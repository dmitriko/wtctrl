
export function OPEN_DRAWER (state) {
    state.is_drawer_open = true
}

export function CLOSE_DRAWER (state) {
    state.is_drawer_open = false
}

export function SET_DRAWER (state, val) {
    state.is_drawer_open = val
}

export function SET_SYS_MSG_ERR (state, val) {
    state.sys_msg_error = val
}

export function SET_SYS_MSG (state, val) {
    state.sys_msg = val
}
