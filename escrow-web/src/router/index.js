import { defineRouter } from '#q-app/wrappers'
import {
  createRouter,
  createMemoryHistory,
  createWebHistory,
  createWebHashHistory,
} from 'vue-router'
import routes from './routes'

export default defineRouter((/* { store, ssrContext } */) => {
  const createHistory = process.env.SERVER
    ? createMemoryHistory
    : process.env.VUE_ROUTER_MODE === 'history'
      ? createWebHistory
      : createWebHashHistory

  const Router = createRouter({
    scrollBehavior: () => ({ left: 0, top: 0 }),
    routes,
    history: createHistory(process.env.VUE_ROUTER_BASE),
  })

  // 导航守卫：检查登录状态
  Router.beforeEach((to) => {
    if (to.meta.requiresAuth) {
      const token = localStorage.getItem('token')
      if (!token) {
        return { name: 'login', query: { redirect: to.fullPath } }
      }
    }
  })

  return Router
})
