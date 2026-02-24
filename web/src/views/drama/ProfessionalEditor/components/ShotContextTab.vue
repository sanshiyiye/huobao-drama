<template>
  <div class="shot-context-tab">
    <!-- 场景/背景图 -->
    <div class="context-section">
      <div class="section-label">
        <span class="section-title">{{ $t("storyboard.scene") }}</span>
        <el-button
          size="small"
          text
          class="section-action"
          @click="handleShowSceneSelector"
        >
          {{ $t("storyboard.selectScene") }}
        </el-button>
      </div>
      <div
        class="context-block scene-preview"
        v-if="hasImage(currentStoryboard?.background)"
        @click="showSceneImage"
      >
        <img
          :src="getImageUrl(currentStoryboard?.background)"
          alt="场景"
          style="cursor: pointer"
        />
        <div class="scene-info">
          <div class="scene-meta">
            {{ currentStoryboard?.background?.location }} ·
            {{ currentStoryboard?.background?.time }}
          </div>
          <div class="scene-id">
            {{ $t("editor.sceneId") }}:
            {{ currentStoryboard?.scene_id || "N/A" }}
          </div>
        </div>
      </div>
      <div class="context-block scene-preview-empty" v-else>
        <el-icon :size="48" class="empty-icon">
          <Picture />
        </el-icon>
        <div class="empty-text">
          {{
            currentStoryboard?.background
              ? $t("editor.sceneGenerating")
              : $t("editor.noBackground")
          }}
        </div>
      </div>
    </div>

    <!-- 登场角色 -->
    <div class="context-section">
      <div class="section-label">
        <span class="section-title">{{ $t("editor.cast") }}</span>
        <el-button
          size="small"
          text
          class="section-action"
          :icon="Plus"
          @click="handleShowCharacterSelector"
        >
          {{ $t("editor.addCharacter") }}
        </el-button>
      </div>
      <div class="cast-list">
        <div
          v-for="char in currentStoryboardCharacters"
          :key="char.id"
          class="context-block cast-item"
        >
          <div class="cast-avatar" @click="showCharacterImage(char)">
            <img
              v-if="hasImage(char)"
              :src="getImageUrl(char)"
              :alt="char.name"
            />
            <span v-else>{{ char.name?.[0] || "?" }}</span>
          </div>
          <div class="cast-name">{{ char.name }}</div>
          <div
            class="cast-remove"
            @click.stop="toggleCharacterInShot(char.id)"
            :title="$t('editor.removeCharacter')"
          >
            <el-icon :size="14">
              <Close />
            </el-icon>
          </div>
        </div>
        <div
          v-if="
            !currentStoryboard?.characters ||
            currentStoryboard.characters.length === 0
          "
          class="context-block cast-empty"
        >
          {{ $t("editor.noCharacters") }}
        </div>
      </div>
    </div>

    <!-- 道具 -->
    <div class="context-section">
      <div class="section-label">
        <span class="section-title">{{ $t("editor.props") }}</span>
        <el-button
          size="small"
          text
          class="section-action"
          :icon="Plus"
          @click="handleShowPropSelector"
        >
          {{ $t("editor.addProp") }}
        </el-button>
      </div>
      <div class="cast-list">
        <div
          v-for="prop in currentStoryboardProps"
          :key="prop.id"
          class="context-block cast-item"
        >
          <div class="cast-avatar">
            <img
              v-if="hasImage(prop)"
              :src="getImageUrl(prop)"
              :alt="prop.name"
            />
            <el-icon v-else>
              <Box />
            </el-icon>
          </div>
          <div class="cast-name">{{ prop.name }}</div>
          <div
            class="cast-remove"
            @click.stop="togglePropInShot(prop.id)"
            title="移除道具"
          >
            <el-icon :size="14">
              <Close />
            </el-icon>
          </div>
        </div>
        <div
          v-if="
            !currentStoryboardProps ||
            currentStoryboardProps.length === 0
          "
          class="context-block cast-empty"
        >
          {{ $t("editor.noProps") }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, Close, Picture, Box } from '@element-plus/icons-vue'
import { getImageUrl, hasImage } from '@/utils/image'

interface ShotContextTabProps {
  currentStoryboard?: any
  characters?: any[]
  props?: any[]
}

interface ShotContextTabEmits {
  (e: 'toggleCharacter', charId: number): void
  (e: 'toggleProp', propId: number): void
  (e: 'showSceneSelector'): void
  (e: 'showSceneImage'): void
  (e: 'showCharacterSelector'): void
  (e: 'showCharacterImage', char: any): void
  (e: 'showPropSelector'): void
}

const props = defineProps<ShotContextTabProps>()
const emit = defineEmits<ShotContextTabEmits>()
const { t } = useI18n()

// 组件内部状态
const showSceneSelector = ref(false)
const showCharacterSelector = ref(false)
const showPropSelector = ref(false)

// 计算属性
const currentStoryboardCharacters = computed(() => {
  if (!props.currentStoryboard?.characters) {
    return []
  }

  // 检查 characters 数组中的元素是对象还是 ID
  if (props.currentStoryboard.characters.length > 0 && typeof props.currentStoryboard.characters[0] === 'object') {
    // 如果是对象，直接返回
    return props.currentStoryboard.characters
  } else {
    // 如果是 ID，通过 ID 查找对应的对象
    return props.currentStoryboard.characters
      .map(characterId => props.characters?.find(char => char.id === characterId))
      .filter(char => char !== undefined)
  }
})

const currentStoryboardProps = computed(() => {
  if (!props.currentStoryboard?.props) {
    return []
  }

  // 检查 props 数组中的元素是对象还是 ID
  if (props.currentStoryboard.props.length > 0 && typeof props.currentStoryboard.props[0] === 'object') {
    // 如果是对象，直接返回
    return props.currentStoryboard.props
  } else {
    // 如果是 ID，通过 ID 查找对应的对象
    return props.currentStoryboard.props
      .map(propId => props.props?.find(prop => prop.id === propId))
      .filter(prop => prop !== undefined)
  }
})

// 方法
const handleShowSceneSelector = () => {
  emit('showSceneSelector')
}

const handleShowCharacterSelector = () => {
  emit('showCharacterSelector')
}

const handleShowPropSelector = () => {
  emit('showPropSelector')
}

const toggleCharacterInShot = (charId: number) => {
  emit('toggleCharacter', charId)
}

const togglePropInShot = (propId: number) => {
  emit('toggleProp', propId)
}

const showSceneImage = () => {
  emit('showSceneImage')
}

const showCharacterImage = (char: any) => {
  emit('showCharacterImage', char)
}
</script>

<style scoped>
.shot-context-tab {
  padding: 0;
}

/* 统一区块：场景、登场角色、道具 */
.context-section {
  margin-bottom: 20px;
}

.section-label {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  gap: 8px;
}

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.section-action {
  color: var(--text-secondary);
}
.section-action:hover {
  color: var(--accent);
}

/* 统一内容块：背景图预览、角色卡片、空状态 */
.context-block {
  background: var(--bg-card);
  border: 1px solid var(--border-primary);
  border-radius: 8px;
  padding: 12px;
}

/* 场景/背景图 */
.scene-preview {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  transition: background 0.2s, border-color 0.2s;
}
.scene-preview:hover {
  background: var(--bg-card-hover);
  border-color: var(--border-secondary);
}

.scene-preview img {
  width: 80px;
  height: 45px;
  object-fit: cover;
  border-radius: 6px;
  flex-shrink: 0;
}

.scene-info {
  flex: 1;
  min-width: 0;
}

.scene-meta {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 4px;
}

.scene-id {
  font-size: 12px;
  color: var(--text-muted);
}

.scene-preview-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px;
  min-height: 80px;
}

.scene-preview-empty .empty-icon {
  color: var(--text-muted);
}

.scene-preview-empty .empty-text {
  margin-top: 10px;
  font-size: 13px;
  color: var(--text-muted);
}

/* 登场角色 / 道具 列表 */
.cast-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.cast-item {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 88px;
  transition: background 0.2s, border-color 0.2s;
}
.cast-item:hover {
  background: var(--bg-card-hover);
  border-color: var(--border-secondary);
}

.cast-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  overflow: hidden;
  margin-bottom: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
}

.cast-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.cast-avatar span {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-secondary);
}

.cast-name {
  font-size: 12px;
  color: var(--text-primary);
  text-align: center;
  line-height: 1.3;
}

.cast-remove {
  position: absolute;
  top: 4px;
  right: 4px;
  cursor: pointer;
  padding: 4px;
  border-radius: 50%;
  color: var(--text-secondary);
  background: var(--bg-secondary);
}
.cast-remove:hover {
  color: var(--text-primary);
  background: var(--bg-card-hover);
}

.cast-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
  width: 100%;
  min-height: 72px;
  color: var(--text-muted);
  font-size: 13px;
}
</style>
