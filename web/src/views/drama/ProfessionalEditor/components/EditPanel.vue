<template>
  <div class="edit-panel">
    <el-tabs :model-value="activeTab" @update:model-value="$emit('update:activeTab', $event)" class="edit-tabs">
      <!-- 镜头图片 -->
      <el-tab-pane name="image">
        <template #label>
          <span class="edit-tab-label">
            <el-icon><Picture /></el-icon>
            <span>{{ $t('editor.shotImage') }}</span>
          </span>
        </template>
        <slot name="image-tab" />
      </el-tab-pane>

      <!-- 视频生成 -->
      <el-tab-pane name="video">
        <template #label>
          <span class="edit-tab-label">
            <el-icon><VideoPlay /></el-icon>
            <span>{{ $t('video.videoGeneration') }}</span>
          </span>
        </template>
        <slot name="video-tab" />
      </el-tab-pane>

      <!-- 音效与配乐（条件显示） -->
      <el-tab-pane v-if="showAudioTab" :label="$t('video.soundAndMusicTab')" name="audio">
        <slot name="audio-tab" />
      </el-tab-pane>

      <!-- 视频合成 -->
      <el-tab-pane :label="$t('video.videoMerge')" name="merges">
        <slot name="merges-tab" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { Picture, VideoPlay } from "@element-plus/icons-vue";
import { useI18n } from "vue-i18n";

defineProps<{
  activeTab: string;
  /** 是否显示「有声/配乐」标签页（依赖模型能力） */
  showAudioTab?: boolean;
}>();

defineEmits<{
  (e: "update:activeTab", value: string): void;
}>();

const { t: $t } = useI18n();
</script>

<style scoped lang="scss">
.edit-panel {
  width: 520px;
  background: var(--bg-card);
  border-left: 1px solid var(--border-primary);
  overflow: hidden;
  flex-shrink: 0;
}

.edit-tabs {
  height: 100%;

  .edit-tab-label {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }

  :deep(.el-tabs__header) {
    margin: 0;
    background: var(--bg-secondary);
    padding: 0 16px;
    border-bottom: 1px solid var(--border-primary);
  }

  :deep(.el-tabs__content) {
    height: calc(100% - 55px);
    overflow-y: auto;
    background: var(--bg-secondary);
  }

  :deep(.tab-content) {
    padding: 16px;
    background: var(--bg-secondary);
    min-height: 100%;
  }
}
</style>
