import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/login',
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/auth/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/p/:slug',
      name: 'public-profile',
      component: () => import('@/views/public/ProfilePublicView.vue'),
      meta: { public: true },
    },
    {
      path: '/loyalty/:token',
      name: 'loyalty-card',
      component: () => import('@/views/public/LoyaltyCardView.vue'),
      meta: { public: true },
    },

    // --- Client editor (full-screen, no sidebar) ---
    {
      path: '/client/editor',
      name: 'client-editor',
      component: () => import('@/views/client/ProfileEditorView.vue'),
      meta: { requiresRole: 'CLIENT' },
    },

    // --- Client panel (with sidebar shell) ---
    {
      path: '/client',
      component: () => import('@/layouts/ClientLayout.vue'),
      meta: { requiresRole: 'CLIENT' },
      children: [
        { path: '', name: 'client-dashboard', component: () => import('@/views/client/DashboardView.vue') },
        { path: 'license', name: 'client-license', component: () => import('@/views/client/LicenseView.vue') },
        { path: 'stats', redirect: { name: 'client-dashboard' } },
        { path: 'loyalty', name: 'client-loyalty', component: () => import('@/views/client/LoyaltyView.vue') },
      ],
    },

    // --- Admin panel (with sidebar shell) ---
    {
      path: '/admin',
      component: () => import('@/layouts/AdminLayout.vue'),
      meta: { requiresRole: 'ADMIN' },
      children: [
        { path: '', name: 'admin-dashboard', component: () => import('@/views/admin/DashboardView.vue') },
        { path: 'clients', name: 'admin-clients', component: () => import('@/views/admin/ClientsView.vue') },
        { path: 'clients/:id', name: 'admin-client-detail', component: () => import('@/views/admin/ClientDetailView.vue') },
        { path: 'clients/:id/print-cards', name: 'admin-print-cards', component: () => import('@/views/admin/PrintCardsView.vue') },
        { path: 'licenses', name: 'admin-licenses', component: () => import('@/views/admin/LicensesView.vue') },
        { path: 'templates', name: 'admin-templates', component: () => import('@/views/admin/TemplatesView.vue') },
        { path: 'audit-logs', name: 'admin-audit', component: () => import('@/views/admin/AuditLogsView.vue') },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true

  const auth = useAuthStore()
  if (!auth.isAuthenticated) {
    return { name: 'login' }
  }

  if (!auth.user) {
    try {
      await auth.fetchCurrentUser()
    } catch {
      auth.clearSession()
      return { name: 'login' }
    }
  }

  const requiredRole = to.meta.requiresRole as string | undefined
  if (requiredRole && auth.user?.role !== requiredRole) {
    return auth.isAdmin ? { name: 'admin-dashboard' } : { name: 'client-dashboard' }
  }

  return true
})

export default router
