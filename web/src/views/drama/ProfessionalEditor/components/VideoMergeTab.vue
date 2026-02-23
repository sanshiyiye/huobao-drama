<template>
  <div class="video-merge-tab">
    <div class="merges-list" v-loading="loadingMerged">
      <el-empty
        v-if="videoMerged.length === 0"
        :description="$t('video.noMergeRecords')"
        :image-size="120"
      >
        <template #bottom>
          <el-button type="primary" @click="$emit('createMerge')">
            {{ $t('video.createMerge') }}
          </el-button>
        </template>
      </el-empty>

      <el-table
        v-else
        :data="videoMerged"
        style="width: 100%"
        :default-sort="{ prop: 'created_at', order: 'descending' }"
      >
        <el-table-column
          prop="id"
          label="ID"
          width="80"
          sortable
        />
        <el-table-column
          prop="title"
          label="标题"
          min-width="200"
          show-overflow-tooltip
        />
        <el-table-column
          prop="status"
          label="状态"
          width="100"
          sortable
          :formatter="formatStatus"
        />
        <el-table-column
          prop="progress"
          label="进度"
          width="150"
        >
          <template #default="{ row }">
            <el-progress
              v-if="row.status !== 'completed' && row.status !== 'failed'"
              :percentage="row.progress || 0"
              :stroke-width="6"
            />
            <span v-else>{{ formatProgress(row.status) }}</span>
          </template>
        </el-table-column>
        <el-table-column
          prop="created_at"
          label="创建时间"
          width="180"
          sortable
          :formatter="formatDateTime"
        />
        <el-table-column
          prop="duration"
          label="时长(秒)"
          width="100"
          sortable
        />
        <el-table-column
          label="操作"
          width="180"
          fixed="right"
        >
          <template #default="{ row }">
            <el-button
              size="small"
              link
              @click="$emit('preview', row)"
              v-if="row.status === 'completed' && row.video_url"
            >
              {{ $t('video.preview') }}
            </el-button>
            <el-button
              size="small"
              link
              @click="$emit('download', row)"
              v-if="row.status === 'completed' && row.video_url"
            >
              {{ $t('video.download') }}
            </el-button>
            <el-button
              size="small"
              link
              type="danger"
              @click="$emit('delete', row)"
            >
              {{ $t('video.delete') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import type { VideoMerge } from "@/types/video";

interface Props {
  videoMerged: VideoMerge[];
  loadingMerged: boolean;
}

interface Emits {
  (e: "preview", merge: VideoMerge): void;
  (e: "download", merge: VideoMerge): void;
  (e: "delete", merge: VideoMerge): void;
  (e: "createMerge"): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const { t: $t } = useI18n();

const formatStatus = (row: VideoMerge) => {
  const statusMap: Record<string, string> = {
    pending: $t('video.pending'),
    processing: $t('video.processing'),
    completed: $t('video.completed'),
    failed: $t('video.failed')
  };
  return statusMap[row.status] || row.status;
};

const formatProgress = (status: string) => {
  if (status === 'completed') return '100%';
  if (status === 'failed') return '0%';
  return '0%';
};

const formatDateTime = (row: VideoMerge) => {
  return new Date(row.created_at).toLocaleString('zh-CN');
};
</script>

<style scoped>
.video-merge-tab {
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.merges-list {
  flex: 1;
  overflow: auto;
  padding: 8px;
}
</style>
