
const routes = [
  {
    path: '/:folder?',
    component: () => import('layouts/MainLayout.vue'),
    children: [
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
