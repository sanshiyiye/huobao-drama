<template>
  <div class="image-generation-tab">
    <!-- 帧类型选择 -->
    <div class="frame-type-selector">
      <el-radio-group v-model="internalSelectedFrameType" size="small" @change="handleFrameTypeChange">
        <el-radio-button label="first">{{ $t("editor.firstFrame") }}</el-radio-button>
        <el-radio-button label="key">{{ $t("editor.keyFrame") }}</el-radio-button>
        <el-radio-button label="last">{{ $t("editor.lastFrame") }}</el-radio-button>
        <el-radio-button label="panel">{{ $t("editor.panelFrame") }}</el-radio-button>
        <el-radio-button label="action">{{ $t("editor.actionSequence") }}</el-radio-button>
      </el-radio-group>
    </div>

    <!-- 提示词输入 -->
    <div class="prompt-input-section">
      <div class="section-header">
        <span>{{ $t('editor.prompt') }}</span>
        <el-button
          size="small"
          :loading="isGeneratingPrompt"
          @click="handleExtractPrompt"
          :disabled="!currentStoryboard"
        >
          {{ $t('editor.extractPrompt') }}
        </el-button>
      </div>
      <el-input
        v-model="internalCurrentFramePrompt"
        type="textarea"
        :rows="5"
        :placeholder="$t('editor.promptPlaceholder')"
      />
    </div>

    <!-- 图片生成控制 -->
    <div class="generate-controls">
      <el-button
        type="primary"
        :loading="generatingImage"
        @click="handleGenerateImage"
        :disabled="!currentStoryboard || !internalCurrentFramePrompt"
      >
        {{ generatingImage ? $t('editor.generating') : $t('editor.generateImage') }}
      </el-button>
      <el-button
        size="small"
        @click="handleUploadImage"
        :disabled="!currentStoryboard"
      >
        {{ $t('editor.uploadImage') }}
      </el-button>
    </div>

    <!-- 图片列表 -->
    <div class="images-list" v-if="filteredImages.length > 0">
      <div class="section-header">
        <span>{{ $t('editor.generationResult') }} ({{ filteredImages.length }})</span>
        <el-button size="small" link @click="handleRefreshImages">
          <el-icon><Refresh /></el-icon>
          {{ $t('editor.refresh') }}
        </el-button>
      </div>
      <el-scrollbar wrap-class="images-container">
        <div class="image-items">
          <div
            v-for="(image, index) in filteredImages"
            :key="image.id || index"
            class="image-item"
            @click="handleSelectImage(image)"
          >
            <img
              v-if="hasImage(image)"
              :src="getImageUrl(image)"
              :alt="image.prompt"
              @error="handleImageError"
            />
            <div v-else class="image-placeholder">
              {{ getStatusText(image.status) }}
            </div>
            <div class="image-status" v-if="image.status !== 'completed'">
              {{ getStatusText(image.status) }}
            </div>
            <div class="image-actions">
              <el-button
                size="small"
                link
                @click.stop="handlePreviewImage(image)"
              >
                {{ $t('editor.preview') }}
              </el-button>
              <el-button
                size="small"
                link
                type="danger"
                @click.stop="handleDeleteImage(image)"
              >
                {{ $t('editor.delete') }}
              </el-button>
            </div>
          </div>
        </div>
      </el-scrollbar>
    </div>

    <!-- 图片预览对话框 -->
    <el-dialog
      v-model="showPreview"
      title="图片预览"
      width="80%"
      top="5vh"
    >
      <div class="preview-container">
        <img v-if="previewImageUrl" :src="previewImageUrl" :alt="previewPrompt" />
        <div v-else class="preview-placeholder">{{ $t('editor.previewLoading') }}</div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'

interface ImageGenerationTabProps {
  currentStoryboard?: any
  generatedImages?: any[]
  loadingImages?: boolean
  isGeneratingPrompt?: boolean
  generatingImage?: boolean
  selectedFrameType?: string
  currentFramePrompt?: string
}

interface ImageGenerationTabEmits {
  (e: 'update:currentFramePrompt', value: string): void
  (e: 'update:selectedFrameType', value: string): void
  (e: 'extractPrompt'): void
  (e: 'generateImage'): void
  (e: 'uploadImage'): void
  (e: 'refreshImages'): void
  (e: 'selectImage', image: any): void
  (e: 'deleteImage', image: any): void
}

const props = defineProps<ImageGenerationTabProps>()
const emit = defineEmits<ImageGenerationTabEmits>()
const { t } = useI18n()

// 组件内部状态
const selectedImageId = ref<string | number | null>(null)
const showPreview = ref(false)
const previewImageUrl = ref('')
const previewPrompt = ref('')

// 内部状态与props同步
const internalSelectedFrameType = ref(props.selectedFrameType || 'first')
const internalCurrentFramePrompt = ref(props.currentFramePrompt || '')
const isGeneratingPrompt = ref(props.isGeneratingPrompt || false)
const generatingImage = ref(props.generatingImage || false)
const generatedImages = ref(props.generatedImages || [])

// 根据帧类型过滤图片
const filteredImages = computed(() => {
  return generatedImages.value.filter(image => image.frame_type === internalSelectedFrameType.value)
})

// 监听props变化
watch(() => props.currentFramePrompt, (newVal) => {
  if (newVal !== undefined) {
    internalCurrentFramePrompt.value = newVal
  }
})

watch(() => props.selectedFrameType, (newVal) => {
  if (newVal !== undefined) {
    internalSelectedFrameType.value = newVal
  }
})

watch(() => props.isGeneratingPrompt, (newVal) => {
  isGeneratingPrompt.value = newVal || false
})

watch(() => props.generatingImage, (newVal) => {
  generatingImage.value = newVal || false
})

watch(() => props.generatedImages, (newVal) => {
  if (newVal !== undefined) {
    generatedImages.value = newVal
  }
})

// 方法
const handleFrameTypeChange = (newType: string) => {
  internalSelectedFrameType.value = newType
  emit('update:selectedFrameType', newType)
}

const handleExtractPrompt = () => {
  emit('extractPrompt')
}

const handleGenerateImage = () => {
  if (!internalCurrentFramePrompt.value.trim()) {
    ElMessage.warning(t('editor.promptRequired'))
    return
  }
  emit('generateImage')
}

const handleUploadImage = () => {
  emit('uploadImage')
}

const handleRefreshImages = () => {
  emit('refreshImages')
}

const handleSelectImage = (image: any) => {
  selectedImageId.value = image.id
  emit('selectImage', image)
}

const handlePreviewImage = (image: any) => {
  if (hasImage(image)) {
    previewImageUrl.value = getImageUrl(image)
    previewPrompt.value = image.prompt || ''
    showPreview.value = true
  } else {
    ElMessage.warning(t('editor.imageNotAvailable'))
  }
}

const handleDeleteImage = async (image: any) => {
  try {
    await ElMessageBox.confirm(
      t('editor.confirmDeleteImage'),
      t('common.confirm'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )
    emit('deleteImage', image)
  } catch (error) {
    // 用户取消删除
  }
}

const handleImageError = (event: any) => {
  console.error('图片加载失败:', event)
}

const hasImage = (image: any) => {
  return image.image_url || image.local_path
}

const getImageUrl = (image: any) => {
  return image.image_url || (image.local_path ? `/api/files/${image.local_path}` : '')
}

const getStatusText = (status: string) => {
  const statusMap: Record<string, string> = {
    pending: t('editor.pending'),
    processing: t('editor.processing'),
    completed: t('editor.completed'),
    failed: t('editor.failed')
  }
  return statusMap[status] || status
}
</script>

<style scoped>
.image-generation-tab {
  padding: 20px;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.frame-type-selector {
  margin-bottom: 10px;
}

.prompt-input-section {
  flex: 1;
  min-height: 200px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  font-weight: 500;
}

.generate-controls {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}

.images-list {
  flex: 1;
  min-height: 200px;
}

.images-container {
  max-height: 400px;
  overflow-y: auto;
}

.image-items {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 15px;
}

.image-item {
  position: relative;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.3s;
}

.image-item:hover {
  border-color: #409eff;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.image-item img {
  width: 100%;
  height: 120px;
  object-fit: cover;
}

.image-placeholder {
  width: 100%;
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #f5f7fa;
  color: #909399;
  font-size: 12px;
}

.image-status {
  position: absolute;
  top: 10px;
  left: 10px;
  background-color: rgba(0, 0, 0, 0.6);
  color: white;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
}

.image-actions {
  display: flex;
  justify-content: space-between;
  padding: 8px;
  background-color: #f8f9fa;
}

.image-actions .el-button {
  padding: 0;
}

.preview-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}

.preview-container img {
  max-width: 100%;
  max-height: 60vh;
}

.preview-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  color: #909399;
}
</style>
