<template>
  <div class="video-management">
    <div class="management-header">
      <h2>{{ $t('drama.management.videoManagement') }}</h2>
      <el-button type="primary" @click="refreshVideos">
        <el-icon><Refresh /></el-icon>
        {{ $t('common.refresh') }}
      </el-button>
    </div>

    <el-card class="management-card">
      <template #header>
        <div class="card-header">
          <span>{{ $t('drama.management.videoList') }}</span>
          <div class="header-actions">
            <el-input
              v-model="searchQuery"
              :placeholder="$t('common.search')"
              clearable
              style="width: 200px"
              @clear="onSearch"
              @input="onSearch"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
            <el-button
              type="danger"
              size="small"
              @click="deleteSelectedVideos"
              :disabled="selectedVideos.length === 0"
            >
              <el-icon><Delete /></el-icon>
              {{ $t('common.deleteSelected') }}
            </el-button>
          </div>
        </div>
      </template>

      <el-table
        v-loading="loading"
        :data="filteredVideos"
        @selection-change="handleSelectionChange"
        stripe
        style="width: 100%"
      >
        <el-table-column type="selection" width="50" />
        <el-table-column type="index" :label="$t('common.index')" width="60" />
        <el-table-column
          prop="filename"
          :label="$t('drama.management.filename')"
          min-width="200"
          show-overflow-tooltip
        />
        <el-table-column
          prop="isAssociated"
          :label="$t('drama.management.databaseAssociation')"
          width="150"
        >
          <template #default="{ row }">
            <el-tag :type="row.isAssociated ? 'success' : 'warning'">
              {{ row.isAssociated ? $t('drama.management.associated') : $t('drama.management.notAssociated') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column
          prop="created_at"
          :label="$t('common.createdAt')"
          width="180"
          :formatter="formatDateTime"
        />
        <el-table-column
          :label="$t('common.actions')"
          width="120"
          fixed="right"
        >
          <template #default="{ row }">
            <el-button
              size="small"
              link
              @click="viewVideo(row)"
            >
              {{ $t('common.view') }}
            </el-button>
            <el-button
              size="small"
              link
              type="danger"
              @click="deleteVideo(row)"
            >
              {{ $t('common.delete') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-if="filteredVideos.length > 0"
        class="pagination"
        background
        layout="total, sizes, prev, pager, next, jumper"
        :total="filteredVideos.length"
        :page-sizes="[10, 20, 50, 100]"
        v-model:page-size="pageSize"
        v-model:current-page="currentPage"
      />
    </el-card>

    <!-- 视频预览对话框 -->
    <el-dialog
      v-model="previewDialogVisible"
      :title="$t('drama.management.previewVideo')"
      width="80%"
      :before-close="handlePreviewClose"
    >
      <div class="preview-container" v-if="previewVideo">
        <video
          :src="previewVideo.url"
          :alt="previewVideo.filename"
          class="preview-video"
          controls
          autoplay
        />
        <div class="preview-info">
          <h3>{{ previewVideo.filename }}</h3>
          <p>{{ $t('drama.management.filePath') }}: {{ previewVideo.path }}</p>
          <p>{{ $t('drama.management.fileSize') }}: {{ formatFileSize(previewVideo.size) }}</p>
          <p>{{ $t('drama.management.associationStatus') }}: {{ formatAssociationStatusText(previewVideo.isAssociated) }}</p>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { ElMessage, ElMessageBox } from 'element-plus';
import { Refresh, Search, Delete } from '@element-plus/icons-vue';
import { listVideos, deleteVideo as deleteVideoFile, FileInfo } from '@/api/fileManagement';

const { t: $t } = useI18n();

// 响应式数据
const loading = ref(false);
const searchQuery = ref('');
const selectedVideos = ref<FileInfo[]>([]);
const currentPage = ref(1);
const pageSize = ref(10);

// 视频数据
const videos = ref<FileInfo[]>([]);

// 预览对话框
const previewDialogVisible = ref(false);
const previewVideo = ref<any>(null);

// 计算属性
const filteredVideos = computed(() => {
  if (!searchQuery.value) {
    return videos.value;
  }
  return videos.value.filter(video =>
    video.filename.toLowerCase().includes(searchQuery.value.toLowerCase())
  );
});

// 方法
const refreshVideos = async () => {
  loading.value = true;
  try {
    const data = await listVideos();
    videos.value = data.map((item: any) => ({
      ...item,
      isAssociated: item.is_associated ?? item.isAssociated,
      createdAt: item.created_at ?? item.createdAt,
    }));
    ElMessage.success($t('drama.management.videosRefreshed'));
  } catch (error) {
    ElMessage.error($t('drama.management.failedToRefreshVideos'));
    console.error(error);
  } finally {
    loading.value = false;
  }
};

const onSearch = () => {
  currentPage.value = 1;
};

const handleSelectionChange = (selection: any[]) => {
  selectedVideos.value = selection;
};

const formatAssociationStatus = (row: any) => {
  return row.isAssociated ? $t('drama.management.associated') : $t('drama.management.notAssociated');
};

const formatAssociationStatusText = (isAssociated: boolean) => {
  return isAssociated ? $t('drama.management.associated') : $t('drama.management.notAssociated');
};

const formatDateTime = (row: any) => {
  const raw = row?.createdAt ?? row?.created_at;
  if (raw == null || raw === '') return '—';
  const date = new Date(raw);
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString();
};

const formatFileSize = (bytes: number) => {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

const viewVideo = (row: any) => {
  previewVideo.value = row;
  previewDialogVisible.value = true;
};

const handlePreviewClose = () => {
  previewDialogVisible.value = false;
  previewVideo.value = null;
};

const deleteVideo = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      $t('drama.management.confirmDeleteVideo'),
      $t('common.confirm'),
      {
        confirmButtonText: $t('common.confirm'),
        cancelButtonText: $t('common.cancel'),
        type: 'warning'
      }
    );

    await deleteVideoFile(row.filename);
    const index = videos.value.findIndex(video => video.filename === row.filename);
    if (index !== -1) {
      videos.value.splice(index, 1);
    }

    ElMessage.success($t('drama.management.videoDeleted'));
  } catch (error) {
    // 用户取消删除
    if (error !== 'cancel') {
      ElMessage.error($t('drama.management.failedToDeleteVideo'));
      console.error(error);
    }
  }
};

const deleteSelectedVideos = async () => {
  if (selectedVideos.value.length === 0) {
    ElMessage.warning($t('drama.management.selectVideosToDelete'));
    return;
  }

  try {
    await ElMessageBox.confirm(
      $t('drama.management.confirmDeleteSelectedVideos', { count: selectedVideos.value.length }),
      $t('common.confirm'),
      {
        confirmButtonText: $t('common.confirm'),
        cancelButtonText: $t('common.cancel'),
        type: 'warning'
      }
    );

    for (const video of selectedVideos.value) {
      await deleteVideoFile(video.filename);
    }

    selectedVideos.value.forEach(video => {
      const index = videos.value.findIndex(vid => vid.filename === video.filename);
      if (index !== -1) {
        videos.value.splice(index, 1);
      }
    });

    selectedVideos.value = [];
    ElMessage.success($t('drama.management.videosDeleted', { count: selectedVideos.value.length }));
  } catch (error) {
    // 用户取消删除
    if (error !== 'cancel') {
      ElMessage.error($t('drama.management.failedToDeleteVideos'));
      console.error(error);
    }
  }
};

// 生命周期钩子
onMounted(() => {
  refreshVideos();
});
</script>

<style scoped>
.video-management {
  padding: 20px;
}

.management-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.management-header h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
}

.management-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.pagination {
  margin-top: 20px;
  text-align: right;
}

.preview-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px;
}

.preview-video {
  max-width: 100%;
  max-height: 60vh;
  margin-bottom: 20px;
  border-radius: 8px;
}

.preview-info {
  text-align: center;
  max-width: 800px;
}

.preview-info h3 {
  margin-bottom: 10px;
  font-size: 18px;
  font-weight: 600;
}

.preview-info p {
  margin: 5px 0;
  color: #666;
}
</style>
