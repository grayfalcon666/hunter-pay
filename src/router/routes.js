const routes = [
  {
    path: '/',
    component: () => import('layouts/MainLayout.vue'),
    children: [
      { path: '', name: 'home', component: () => import('pages/IndexPage.vue') },
      { path: 'login', name: 'login', component: () => import('pages/LoginPage.vue') },
      { path: 'register', name: 'register', component: () => import('pages/RegisterPage.vue') },
      { path: 'user-status', name: 'user-status', component: () => import('pages/UserStatusPage.vue') },
      { path: 'onboarding', name: 'onboarding', component: () => import('pages/OnboardingWizard.vue'), meta: { requiresAuth: true } },
      { path: 'bounty/create', name: 'create-bounty', component: () => import('pages/CreateBountyPage.vue'), meta: { requiresAuth: true } },
      { path: 'bounty/:id', name: 'bounty-detail', component: () => import('pages/BountyDetailPage.vue'), props: true },
      { path: 'profile/:username', name: 'profile', component: () => import('pages/ProfilePage.vue'), props: true },
      { path: 'profile/edit', name: 'edit-profile', component: () => import('pages/EditProfilePage.vue'), meta: { requiresAuth: true } },
      { path: 'reviews/:username', name: 'reviews', component: () => import('pages/ReviewsPage.vue'), props: true },
      { path: 'my-bounties', name: 'my-bounties', component: () => import('pages/MyBountiesPage.vue'), meta: { requiresAuth: true } },
      { path: 'my-applications', name: 'my-applications', component: () => import('pages/MyApplicationsPage.vue'), meta: { requiresAuth: true } },
      { path: 'my/comments', name: 'my-comments', component: () => import('pages/MyCommentsPage.vue'), meta: { requiresAuth: true } },
      { path: 'my-tasks', name: 'my-tasks', component: () => import('pages/MyTasksPage.vue'), meta: { requiresAuth: true } },
      { path: 'hunters', name: 'hunters', component: () => import('pages/HunterGalleryPage.vue'), meta: { requiresAuth: true } },
      { path: 'wallet', name: 'wallet', component: () => import('pages/WalletPage.vue'), meta: { requiresAuth: true } },
      { path: 'wallet/history', name: 'wallet-history', component: () => import('pages/TransactionHistoryPage.vue'), meta: { requiresAuth: true } },
    ],
  },

  {
    path: '/:catchAll(.*)*',
    component: () => import('pages/ErrorNotFound.vue'),
  },
]

export default routes
