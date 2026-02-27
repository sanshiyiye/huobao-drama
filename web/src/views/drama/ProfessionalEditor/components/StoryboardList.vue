<template>
  <div class="storyboard-panel">
    <div class="panel-header">
      <h3 class="panel-title">{{ $t("storyboard.shotList") }}</h3>
      <div class="panel-header-right">
        <el-tooltip content="分镜列表：按顺序编辑各镜头内容" placement="bottom">
          <el-icon class="header-info-icon"><InfoFilled /></el-icon>
        </el-tooltip>
        <el-button text :icon="Plus" @click="handleAddStoryboard">
          {{ $t("storyboard.add") }}
        </el-button>
      </div>
    </div>

    <div
      class="storyboard-list"
      @dragover.prevent
      @drop.prevent="onListDrop"
    >
      <div
        v-for="(shot, index) in orderedList"
        :key="shot.id"
        class="storyboard-item"
        :class="{ active: String(currentStoryboardId) === String(shot.id), 'drag-over': dragOverIndex === index }"
        draggable="true"
        @click="!isDragging && selectStoryboard(shot.id)"
        @dragstart="onDragStart($event, index)"
        @dragend="onDragEnd"
        @dragover.prevent="dragOverIndex = index"
        @dragleave="dragOverIndex = -1"
        @drop.prevent="onDrop($event, index)"
      >
        <div class="shot-content">
          <div class="shot-header">
            <span class="shot-title-text">
              {{ $t("storyboard.shotNumber", { number: shot.storyboard_number }) }} {{ shot.title || $t("storyboard.untitled") }}
            </span>
            <span class="shot-duration">{{ shot.duration }}s</span>
          </div>
          <div class="shot-desc" v-if="shot.action || shot.description">
            {{ shot.action || shot.description }}
          </div>
          <div class="shot-content-actions">
            <el-button
              link
              type="primary"
              :icon="Edit"
              class="edit-btn"
              @click.stop="handleEditStoryboard(shot)"
            >
              {{ $t("workflow.editShot") }}
            </el-button>
            <el-button
              link
              type="danger"
              :icon="Delete"
              @click.stop="handleDeleteStoryboard(shot)"
              class="delete-btn"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { Plus, Delete, InfoFilled } from "@element-plus/icons-vue";
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
  reorder: [storyboardIds: number[]];
  edit: [storyboard: Storyboard];
}>();

const { t: $t } = useI18n();

const orderedList = ref<Storyboard[]>([]);
watch(
  () => props.storyboards,
  (v) => {
    orderedList.value = v && v.length ? [...v] : [];
  },
  { immediate: true, deep: true }
);

const dragFromIndex = ref(-1);
const dragOverIndex = ref(-1);
const isDragging = ref(false);

const selectStoryboard = (id: number) => {
  emit("select", id);
};

const handleAddStoryboard = () => {
  emit("add");
};

const handleEditStoryboard = (storyboard: Storyboard) => {
  emit("edit", storyboard);
};

const handleDeleteStoryboard = (storyboard: Storyboard) => {
  emit("delete", storyboard);
};

const onDragStart = (_e: DragEvent, index: number) => {
  dragFromIndex.value = index;
  isDragging.value = true;
};

const onDragEnd = () => {
  isDragging.value = false;
  dragFromIndex.value = -1;
  dragOverIndex.value = -1;
};

const onDrop = (_e: DragEvent, toIndex: number) => {
  const from = dragFromIndex.value;
  if (from < 0 || from === toIndex) return;
  const list = orderedList.value;
  const [item] = list.splice(from, 1);
  list.splice(toIndex, 0, item);
  emit("reorder", list.map((s) => s.id));
  dragOverIndex.value = -1;
};

const onListDrop = (e: DragEvent) => {
  e.preventDefault();
  dragOverIndex.value = -1;
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
  padding: 14px 16px;
  background: var(--bg-card);
  border-bottom: 1px solid var(--border-primary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.panel-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.panel-header-right {
  display: flex;
  align-items: center;
  gap: 4px;
}

.header-info-icon {
  font-size: 14px;
  color: var(--text-muted);
  cursor: help;
}

.storyboard-list {
  flex: 1;
  overflow-y: auto;
  padding: 10px 12px;
}

.storyboard-item {
  position: relative;
  background: var(--bg-card);
  border: 1px solid var(--border-primary);
  border-radius: 8px;
  padding: 12px 14px;
  margin-bottom: 8px;
  cursor: pointer;
  transition: background 0.2s, border-color 0.2s;
}

.storyboard-item:hover {
  background: var(--bg-card-hover);
  border-color: var(--border-secondary);
}

.storyboard-item:hover .delete-btn {
  opacity: 1;
}

.storyboard-item.active {
  background: var(--bg-card-hover);
  border-color: var(--border-secondary);
  box-shadow: 0 0 0 1px var(--border-secondary);
}

.storyboard-item.active .shot-title-text,
.storyboard-item.active .shot-duration {
  color: var(--text-primary);
}

.shot-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-right: 80px;
}

.shot-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.shot-title-text {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  line-height: 1.35;
  flex: 1;
  min-width: 0;
}

.shot-duration {
  font-size: 12px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.shot-desc {
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
}

.storyboard-item.active .shot-desc {
  color: var(--text-secondary);
}

.storyboard-item.drag-over {
  border-color: var(--el-color-primary);
  background: var(--bg-card-hover);
}

.shot-content-actions {
  position: absolute;
  top: 10px;
  right: 10px;
  display: flex;
  align-items: center;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s;
}

.storyboard-item:hover .shot-content-actions {
  opacity: 1;
}

.shot-content-actions .edit-btn,
.shot-content-actions .delete-btn {
  padding: 4px;
}

.shot-content-actions .edit-btn:hover,
.shot-content-actions .delete-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
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
