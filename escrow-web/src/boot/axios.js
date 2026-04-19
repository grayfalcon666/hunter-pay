import { boot } from 'quasar/wrappers'
import apiClient from 'src/api/client'

export default boot(({ app }) => {
  app.config.globalProperties.$api = apiClient
})

export { apiClient }
