import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import DownloadView from '../views/DownloadView.vue'
import LibraryView from '../views/LibraryView.vue'
import GoogleView from '../views/GoogleView.vue'
import ScheduleView from '../views/ScheduleView.vue'

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'download', component: DownloadView },
  { path: '/library', name: 'library', component: LibraryView },
  { path: '/google', name: 'google', component: GoogleView },
  { path: '/schedule', name: 'schedule', component: ScheduleView },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
