<template>
  <div class="image-generation-tab">
    <!-- 当前使用模型（单独一块，在场景上方） -->
    <div v-if="props.currentImageModel" class="current-model-section">
      <span class="section-label">{{ $t('video.model') }}</span>
      <span class="current-model-value">{{ props.currentImageModel }}</span>
      <template v-if="props.currentImageProvider">
        <span class="section-label provider-label">厂商</span>
        <span class="current-model-value">{{ props.currentImageProvider }}</span>
      </template>
    </div>

    <!-- 镜头上下文信息 -->
    <ShotContextTab
      :current-storyboard="currentStoryboard"
      :characters="characters"
      :props="propsData"
      @toggle-character="handleToggleCharacter"
      @toggle-prop="handleToggleProp"
      @show-scene-selector="handleShowSceneSelector"
      @show-scene-image="handleShowSceneImage"
      @show-character-selector="handleShowCharacterSelector"
      @show-character-image="handleShowCharacterImage"
      @show-prop-selector="handleShowPropSelector"
    />

    <!-- 镜头类型 -->
    <div class="prompt-block-section">
      <div class="section-label">
        <span class="section-title">{{ $t('editor.frameTypeLabel') }}</span>
      </div>
      <div class="frame-type-block">
        <el-radio-group v-model="internalSelectedFrameType" size="small" @change="handleFrameTypeChange" class="frame-type-group">
          <el-radio-button label="first">{{ $t("editor.firstFrame") }}</el-radio-button>
          <el-radio-button label="key">{{ $t("editor.keyFrame") }}</el-radio-button>
          <el-radio-button label="last">{{ $t("editor.lastFrame") }}</el-radio-button>
          <el-radio-button label="panel">{{ $t("editor.panelFrame") }}</el-radio-button>
          <el-radio-button label="action">{{ $t("editor.actionSequence") }}</el-radio-button>
        </el-radio-group>
      </div>
    </div>

    <!-- 提示词编辑 -->
    <div class="prompt-block-section prompt-editing-section">
      <div class="prompt-section-header">
        <span class="section-title">{{ $t('editor.promptEditing') }}</span>
        <span v-if="internalCurrentFramePrompt?.trim()" class="prompt-status generated">
          <el-icon><CircleCheckFilled /></el-icon>
          {{ $t('editor.promptGenerated') }}
        </span>
        <el-button
          size="small"
          :loading="isGeneratingPrompt"
          @click="handleExtractPrompt"
          :disabled="!currentStoryboard"
          class="regenerate-btn"
        >
          <el-icon><Refresh /></el-icon>
          {{ internalCurrentFramePrompt?.trim() ? $t('editor.regeneratePrompt') : $t('editor.extractPrompt') }}
        </el-button>
      </div>
      <div class="prompt-text-block">
        <el-input
          v-model="internalCurrentFramePrompt"
          type="textarea"
          :rows="6"
          :placeholder="$t('editor.promptPlaceholder')"
          class="prompt-textarea"
          resize="vertical"
        />
        <div class="prompt-hint">
          <el-icon class="hint-icon"><InfoFilled /></el-icon>
          <span>{{ $t('editor.promptHint') }}</span>
        </div>
      </div>
    </div>

    <!-- 图片生成控制：等大、并排、配色均衡 -->
    <div class="generate-controls">
      <div class="generate-controls-buttons">
        <el-button
          class="gen-btn gen-btn-primary"
          :loading="generatingImage"
          @click="handleGenerateImage"
          :disabled="!currentStoryboard || !internalCurrentFramePrompt"
        >
          <el-icon class="gen-btn-icon"><Picture /></el-icon>
          <span class="gen-btn-text">{{ generatingImage ? $t('editor.generating') : $t('editor.generateImage') }}</span>
        </el-button>
        <el-button
          class="gen-btn gen-btn-secondary"
          @click="handleUploadImage"
          :disabled="!currentStoryboard"
        >
          <el-icon class="gen-btn-icon"><Upload /></el-icon>
          <span class="gen-btn-text">{{ $t('editor.uploadImage') }}</span>
        </el-button>
      </div>
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
import { Refresh, CircleCheckFilled, InfoFilled, Picture, Upload } from '@element-plus/icons-vue'
import ShotContextTab from './ShotContextTab.vue'
import { getImageUrl, hasImage } from '@/utils/image'

interface ImageGenerationTabProps {
  currentStoryboard?: any
  /** 当前使用的图片模型名称（来自后端默认配置，用于展示） */
  currentImageModel?: string
  /** 当前使用的图片厂商（来自后端默认配置，用于展示） */
  currentImageProvider?: string
  generatedImages?: any[]
  loadingImages?: boolean
  isGeneratingPrompt?: boolean
  generatingImage?: boolean
  selectedFrameType?: string
  currentFramePrompt?: string
  characters?: any[]
  props?: any[]
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
  (e: 'toggleCharacter', charId: number): void
  (e: 'toggleProp', propId: number): void
  (e: 'showSceneSelector'): void
  (e: 'showSceneImage'): void
  (e: 'showCharacterSelector'): void
  (e: 'showCharacterImage', char: any): void
  (e: 'showPropSelector'): void
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
const characters = ref(props.characters || [])
const propsData = ref(props.props || [])

// ShotContextTab 事件处理方法
const handleToggleCharacter = (charId: number) => {
  emit('toggleCharacter', charId)
}

const handleToggleProp = (propId: number) => {
  emit('toggleProp', propId)
}

const handleShowSceneSelector = () => {
  emit('showSceneSelector')
}

const handleShowSceneImage = () => {
  emit('showSceneImage')
}

const handleShowCharacterSelector = () => {
  emit('showCharacterSelector')
}

const handleShowCharacterImage = (char: any) => {
  emit('showCharacterImage', char)
}

const handleShowPropSelector = () => {
  emit('showPropSelector')
}
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
  padding: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* 当前使用模型（在场景上方单独一块） */
.current-model-section {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--bg-card, #f5f7fa);
  border: 1px solid var(--border-primary, #e4e7ed);
  border-radius: 8px;
  margin-bottom: 4px;
}
.current-model-section .section-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}
.current-model-section .section-label.provider-label {
  margin-left: 12px;
}
.current-model-section .current-model-value {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* 镜头类型 + 提示词编辑 统一区块 */
.prompt-block-section {
  margin-bottom: 4px;
}

.prompt-block-section .section-label {
  margin-bottom: 10px;
}

.prompt-block-section .section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.frame-type-block {
  background: var(--bg-card);
  border: 1px solid var(--border-primary);
  border-radius: 8px;
  padding: 10px 12px;
}

.frame-type-group {
  width: 100%;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.frame-type-group :deep(.el-radio-button__inner) {
  background: var(--bg-secondary);
  border-color: var(--border-primary);
  color: var(--text-primary);
}
.frame-type-group :deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
  background: var(--accent);
  border-color: var(--accent);
  color: var(--text-inverse);
}
.frame-type-group :deep(.el-radio-button__inner:hover) {
  color: var(--accent);
}

/* 提示词编辑区块 */
.prompt-editing-section {
  flex: 1;
  min-height: 180px;
  display: flex;
  flex-direction: column;
}

.prompt-section-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}

.prompt-section-header .section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.prompt-status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
}
.prompt-status.generated {
  color: var(--success, #67c23a);
}

.prompt-section-header .regenerate-btn {
  margin-left: auto;
  color: var(--text-secondary);
}
.prompt-section-header .regenerate-btn:hover {
  color: var(--accent);
}

.prompt-text-block {
  background: var(--bg-card);
  border: 1px solid var(--border-primary);
  border-radius: 8px;
  padding: 12px;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.prompt-textarea :deep(.el-textarea__inner) {
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: 6px;
  color: var(--text-primary);
}
.prompt-textarea :deep(.el-textarea__inner::placeholder) {
  color: var(--text-muted);
}

.prompt-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-muted);
}
.prompt-hint .hint-icon {
  font-size: 14px;
  color: var(--text-muted);
  flex-shrink: 0;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  font-weight: 500;
  color: var(--text-primary);
}

.generate-controls {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 20px;
}
.generate-controls-buttons {
  display: flex;
  gap: 12px;
}

.gen-btn {
  flex: 1;
  min-width: 0;
  height: 40px;
  padding: 0 14px;
  border-radius: 8px;
  font-size: 13px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: none;
  transition: opacity 0.2s, transform 0.15s;
}
.gen-btn:not(:disabled):hover {
  opacity: 0.92;
}
.gen-btn:not(:disabled):active {
  transform: scale(0.98);
}
.gen-btn-icon {
  font-size: 16px;
  flex-shrink: 0;
}
.gen-btn-text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.gen-btn-primary {
  background: linear-gradient(135deg, var(--accent) 0%, #7c6cf5 100%);
  color: var(--text-inverse, #fff);
}
.gen-btn-primary:disabled {
  background: var(--bg-secondary);
  color: var(--text-muted);
}

.gen-btn-secondary {
  background: linear-gradient(135deg, #6b5bcd 0%, #5a4bb5 100%);
  color: var(--text-inverse, #fff);
  border: 1px solid rgba(255, 255, 255, 0.15);
}
.gen-btn-secondary:disabled {
  background: var(--bg-secondary);
  color: var(--text-muted);
  border-color: var(--border-primary);
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
  border: 1px solid var(--border-primary);
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.image-item:hover {
  border-color: var(--accent);
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.15);
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
  background: var(--bg-secondary);
  color: var(--text-muted);
  font-size: 12px;
}

.image-status {
  position: absolute;
  top: 10px;
  left: 10px;
  background: rgba(0, 0, 0, 0.6);
  color: var(--text-inverse, #fff);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
}

.image-actions {
  display: flex;
  justify-content: space-between;
  padding: 8px;
  background: var(--bg-secondary);
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
  color: var(--text-muted);
}
</style>
