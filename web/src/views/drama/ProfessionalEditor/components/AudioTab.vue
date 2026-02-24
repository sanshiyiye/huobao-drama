<template>
  <div class="audio-tab">
    <el-alert
      title="音频功能开发中"
      type="info"
      description="音效与配乐功能正在开发中。视频合成无需等待音频生成，您可以直接进行视频合成操作。"
      show-icon
      :closable="false"
      style="margin-bottom: 20px"
    />
    <el-empty :description="$t('video.soundMusicInDev')" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";

const { t: $t } = useI18n();
</script>

<style scoped>
.audio-tab {
  height: 100%;
  padding: 20px;
}
</style>
