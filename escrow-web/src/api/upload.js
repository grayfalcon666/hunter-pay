import axios from 'axios'

const IMAGE_BASE_URL = import.meta.env.VITE_IMAGE_BASE_URL || ''

// 上传图片，返回 relative_path
export async function uploadImage({ file, entityType, entityId }) {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('entity_type', entityType)
  formData.append('entity_id', String(entityId))

  const token = localStorage.getItem('token')
  const headers = { Authorization: `Bearer ${token}` }

  const url = `${IMAGE_BASE_URL}/api/v1/upload`
  console.log('[uploadImage] POST', url, { entityType, entityId, fileName: file.name })
  const res = await axios.post(url, formData, { headers })
  console.log('[uploadImage] response:', res.status, res.data)
  return res.data // { id, relative_path, original_name, file_size, mime_type }
}

// 删除图片
export async function deleteImage(imageId) {
  const token = localStorage.getItem('token')
  const headers = { Authorization: `Bearer ${token}` }
  await axios.delete(`${IMAGE_BASE_URL}/api/v1/upload/${imageId}`, { headers })
}

// 拼接完整图片 URL
export function imageUrl(relativePath) {
  if (!relativePath) return ''
  if (relativePath.startsWith('http://') || relativePath.startsWith('https://')) {
    return relativePath
  }
  return `${IMAGE_BASE_URL}/uploads/${relativePath}`
}
