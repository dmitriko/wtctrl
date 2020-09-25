
const routes = [
  {
      path: '/', name:"main",
    component: () => import('layouts/MainLayout.vue'),
    children: [
      { path: '/:folder', name:'msg', component: () => import('pages/MsgList.vue') },
      { path: '/profile', name:'profile', component: () => import('pages/Profile.vue') }
    ]
  },

  // Always leave this as last one,
  // but you can also remove it
  {
    path: '*',
    component: () => import('pages/Error404.vue')
  }
]

export default routes
