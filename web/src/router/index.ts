import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import DownloadView from '../views/DownloadView.vue'
import LibraryView from '../views/LibraryView.vue'
import PlaylistsView from '../views/PlaylistsView.vue'

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'download', component: DownloadView },
  { path: '/library', name: 'library', component: LibraryView },
  { path: '/playlists', name: 'playlists', component: PlaylistsView },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
