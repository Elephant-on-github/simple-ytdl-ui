import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import { useModeStore } from './stores/mode'

import './styles/fonts.css'
import './styles/theme.css'
import './styles/base.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)

// Paint the correct accent identity before the first frame.
document.documentElement.dataset.mode = useModeStore().mode

app.mount('#app')
