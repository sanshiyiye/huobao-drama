<template>
  <div class="audio-tab">
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
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
