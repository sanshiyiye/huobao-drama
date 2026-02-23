<template>
  <div class="storyboard-panel">
    <div class="panel-header">
      <h3>{{ $t("storyboard.scriptStructure") }}</h3>
      <el-button text :icon="Plus" @click="handleAddStoryboard">
        {{ $t("storyboard.add") }}
      </el-button>
    </div>

    <div class="storyboard-list">
      <div
        v-for="(shot, index) in storyboards"
        :key="shot.id"
        class="storyboard-item"
        :class="{ active: currentStoryboardId === shot.id }"
        @click="selectStoryboard(shot.id)"
      >
        <div class="shot-content">
          <div class="shot-header">
            <div class="shot-title-row">
              <span class="shot-number">{{
                $t("storyboard.shotNumber", {
                  number: shot.storyboard_number,
                })
              }}</span>
              <span class="shot-title">{{
                shot.title || $t("storyboard.untitled")
              }}</span>
            </div>
            <div class="shot-actions">
              <span class="shot-duration">{{ shot.duration }}s</span>
              <el-button
                link
                type="danger"
                :icon="Delete"
                @click.stop="handleDeleteStoryboard(shot)"
                class="delete-btn"
              />
            </div>
          </div>
          <div class="shot-action" v-if="shot.action">
            {{ shot.action }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus, Delete } from "@element-plus/icons-vue";
import { useI18n } from "vue-i18n";
import type { Storyboard } from "@/types/drama";

interface Props {
  storyboards: Storyboard[];
  currentStoryboardId: number;
  onSelectStoryboard: (id: number) => void;
  onAddStoryboard: () => void;
  onDeleteStoryboard: (storyboard: Storyboard) => void;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  select: [id: number];
  add: [];
  delete: [storyboard: Storyboard];
}>();

const { t: $t } = useI18n();

const selectStoryboard = (id: number) => {
  emit("select", id);
};

const handleAddStoryboard = () => {
  emit("add");
};

const handleDeleteStoryboard = (storyboard: Storyboard) => {
  emit("delete", storyboard);
};
</script>

<style scoped>
.storyboard-panel {
  width: 280px;
  height: 100%;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-primary);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-header {
  padding: 16px 20px;
  background: var(--bg-card);
  border-bottom: 1px solid var(--border-primary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.panel-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.storyboard-list {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.storyboard-item {
  background: var(--bg-card);
  border: 1px solid var(--border-primary);
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.storyboard-item:hover {
  background: var(--bg-card-hover);
  border-color: var(--border-secondary);
}

.storyboard-item.active {
  border-color: var(--accent);
  background: var(--accent);
}

.storyboard-item.active .shot-number,
.storyboard-item.active .shot-title,
.storyboard-item.active .shot-duration,
.storyboard-item.active .shot-action {
  color: var(--text-inverse) !important;
}

.shot-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.shot-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.shot-title-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.shot-number {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 500;
}

.shot-title {
  font-size: 14px;
  color: var(--text-primary);
  font-weight: 500;
  line-height: 1.4;
}

.shot-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.shot-duration {
  font-size: 12px;
  color: var(--text-secondary);
}

.shot-action {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.delete-btn {
  padding: 4px;
}

.delete-btn:hover {
  background: rgba(255, 255, 255, 0.1);
}

/* 滚动条样式 */
.storyboard-list::-webkit-scrollbar {
  width: 6px;
}

.storyboard-list::-webkit-scrollbar-track {
  background: var(--bg-secondary);
  border-radius: 3px;
}

.storyboard-list::-webkit-scrollbar-thumb {
  background: var(--border-secondary);
  border-radius: 3px;
}

.storyboard-list::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted);
}
</style>
