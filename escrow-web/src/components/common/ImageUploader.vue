<template>
  <div class="image-uploader">
    <!-- 已上传图片预览网格（多图模式） -->
    <div v-if="isMulti && previewUrls.length" class="image-grid">
      <div v-for="(url, i) in previewUrls" :key="i" class="preview-item">
        <img :src="url" alt="预览" class="preview-img" @click="openPreview(url)" />
        <q-btn
          round dense flat size="sm" icon="close" class="remove-btn"
          color="negative" @click="handleRemove(i)"
        />
      </div>
    </div>

    <!-- 单图预览模式 -->
    <div v-else-if="!isMulti && previewUrl" class="image-preview">
      <img :src="previewUrl" alt="预览" class="preview-img" @click="openPreview(previewUrl)" />
      <q-btn
        round dense flat size="sm" icon="close" class="remove-btn"
        color="negative" @click="handleRemove(0)"
      />
    </div>

    <!-- 上传区域 -->
    <div
      v-if="canUpload"
      class="upload-area"
      :class="{ 'upload-area--compact': isMulti }"
      @click="triggerUpload"
    >
      <q-icon name="add_photo_alternate" size="40px" color="grey-5" />
      <span class="upload-hint">{{ isMulti ? `上传图片（${previewUrls.length}/${maxFiles}）` : '点击上传图片' }}</span>
      <span class="upload-tip">JPG/PNG/GIF/WEBP，最大 10MB</span>
    </div>

    <!-- 隐藏的文件输入 -->
    <input
      ref="fileInput"
      type="file"
      accept="image/jpeg,image/png,image/gif,image/webp"
      class="hidden-input"
      :multiple="isMulti"
      @change="handleFileChange"
    />

    <!-- 上传中遮罩 -->
    <div v-if="uploading" class="uploading-overlay">
      <q-spinner-dots color="amber" size="32px" />
      <span>上传中...</span>
    </div>

    <!-- 全屏预览弹窗 -->
    <q-dialog v-model="previewVisible">
      <q-img :src="previewSrc" style="max-width: 90vw; max-height: 90vh;" />
    </q-dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useQuasar } from 'quasar'
import { uploadImage, deleteImage, imageUrl } from 'src/api/upload'

const props = defineProps({
  modelValue: { type: Array, default: () => [] }, // 相对路径数组 v-model
  uploadUrl: { type: String, default: '/api/v1/upload' },
  maxFiles: { type: Number, default: 1 },
  entityType: { type: String, required: true }, // avatar | bounty | comment
  entityId: { type: [Number, String], required: true },
})
const emit = defineEmits(['update:modelValue', 'upload-success', 'update:uploading'])

const $q = useQuasar()
const fileInput = ref(null)
const uploading = ref(false)
const previewVisible = ref(false)
const previewSrc = ref('')

const isMulti = computed(() => props.maxFiles > 1)

const currentPaths = ref([...props.modelValue])

const previewUrls = computed(() => currentPaths.value.map(p => imageUrl(p)))
const previewUrl = computed(() => previewUrls.value[0] || '')

const canUpload = computed(() => !isMulti.value || currentPaths.value.length < props.maxFiles)

watch(() => props.modelValue, (val) => {
  currentPaths.value = [...val]
})

function triggerUpload() {
  fileInput.value?.click()
}

async function handleFileChange(e) {
  const files = Array.from(e.target.files || [])
  console.log('[ImageUploader] handleFileChange called, files:', files.length)
  console.log('[ImageUploader] props:', { entityType: props.entityType, entityId: props.entityId, maxFiles: props.maxFiles, isMulti: isMulti.value })
  if (!files.length) return

  const remaining = isMulti.value ? props.maxFiles - currentPaths.value.length : 1
  const toUpload = files.slice(0, remaining)
  if (files.length > remaining) {
    $q.notify({ type: 'warning', message: `最多上传 ${props.maxFiles} 张图片` })
  }

  emit('update:uploading', true)
  uploading.value = true

  try {
    for (const file of toUpload) {
      console.log('[ImageUploader] uploading file:', file.name, file.size, file.type)
      if (!validateFile(file)) continue
      const data = await uploadImage({
        file,
        entityType: props.entityType,
        entityId: props.entityId,
      })
      console.log('[ImageUploader] upload success, data:', data)
      emit('upload-success', data) // { id, relative_path, original_name, file_size, mime_type }
      if (isMulti.value) {
        currentPaths.value.push(data.relative_path)
      } else {
        currentPaths.value = [data.relative_path]
      }
    }
    emit('update:modelValue', [...currentPaths.value])
  } catch (err) {
    console.error('[ImageUploader] upload error:', err)
    $q.notify({ type: 'negative', message: err.message || '上传失败' })
  } finally {
    uploading.value = false
    emit('update:uploading', false)
    if (fileInput.value) fileInput.value.value = ''
  }
}

function validateFile(file) {
  if (file.size > 10 * 1024 * 1024) {
    $q.notify({ type: 'negative', message: '文件大小不能超过 10MB' })
    return false
  }
  const validTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
  if (!validTypes.includes(file.type)) {
    $q.notify({ type: 'negative', message: '仅支持 JPG/PNG/GIF/WEBP 格式' })
    return false
  }
  return true
}

async function handleRemove(index) {
  const path = currentPaths.value[index]
  currentPaths.value.splice(index, 1)
  emit('update:modelValue', [...currentPaths.value])
  try {
    await deleteImage(path)
  } catch {
    // ignore: 即使删除 API 失败，也清掉本地状态
  }
}

function openPreview(url) {
  previewSrc.value = url
  previewVisible.value = true
}
</script>

<style scoped lang="scss">
.image-uploader {
  position: relative;
  display: inline-block;
}

.hidden-input {
  display: none;
}

.upload-area {
  width: 120px;
  height: 120px;
  border: 2px dashed var(--color-border);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  cursor: pointer;
  transition: border-color 0.2s;

  &:hover {
    border-color: var(--color-accent-gold);
  }

  .upload-hint {
    font-size: 0.8rem;
    color: var(--color-text-secondary);
  }

  .upload-tip {
    font-size: 0.65rem;
    color: var(--color-text-muted);
    text-align: center;
    padding: 0 8px;
  }

  &--compact {
    width: 80px;
    height: 80px;

    .upload-hint { font-size: 0.7rem; }
    .upload-tip { display: none; }
    .q-icon { font-size: 28px; }
  }
}

.image-preview {
  position: relative;
  width: 120px;
  height: 120px;
  border-radius: 8px;
  overflow: hidden;

  .preview-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: 8px;
    cursor: pointer;
  }

  .remove-btn {
    position: absolute;
    top: 4px;
    right: 4px;
    background: rgba(0, 0, 0, 0.5);
  }
}

.image-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.preview-item {
  position: relative;
  width: 80px;
  height: 80px;
  border-radius: 6px;
  overflow: hidden;

  .preview-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: 6px;
    cursor: pointer;
  }

  .remove-btn {
    position: absolute;
    top: 2px;
    right: 2px;
    background: rgba(0, 0, 0, 0.5);
  }
}

.uploading-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #fff;
  font-size: 0.8rem;
}
</style>
