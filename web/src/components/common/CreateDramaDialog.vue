<template>
  <!-- Create Drama Dialog / 创建短剧弹窗 -->
  <el-dialog
    v-model="visible"
    :title="$t('drama.createNew')"
    width="520px"
    :close-on-click-modal="false"
    class="create-dialog"
    @closed="handleClosed"
  >
    <div class="dialog-desc">{{ $t("drama.createDesc") }}</div>

    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="top"
      class="create-form"
      @submit.prevent="handleSubmit"
    >
      <el-form-item :label="$t('drama.projectName')" prop="title" required>
        <el-input
          v-model="form.title"
          :placeholder="$t('drama.projectNamePlaceholder')"
          size="large"
          maxlength="100"
          show-word-limit
        />
      </el-form-item>

      <el-form-item :label="$t('drama.projectDesc')" prop="description">
        <el-input
          v-model="form.description"
          type="textarea"
          :rows="4"
          :placeholder="$t('drama.projectDescPlaceholder')"
          maxlength="500"
          show-word-limit
          resize="none"
        />
      </el-form-item>

      <el-form-item :label="$t('drama.style')" prop="style" required>
        <div class="style-selector">
          <div class="style-presets">
            <button
              v-for="item in stylePresets"
              :key="item.value"
              type="button"
              class="style-preset-btn"
              :class="{ active: form.style === item.value }"
              @click="selectPresetStyle(item.value)"
            >
              {{ $t(item.labelKey) }}
            </button>
          </div>
          <div class="style-custom">
            <el-input
              :model-value="customStyleInput"
              :placeholder="$t('drama.styleCustomPlaceholder')"
              size="default"
              clearable
              maxlength="50"
              show-word-limit
              @update:model-value="onCustomStyleInput"
            />
          </div>
        </div>
      </el-form-item>

      <el-form-item label="短剧类型" prop="drama_mode">
        <el-select v-model="form.drama_mode" placeholder="请选择短剧类型" size="large" style="width: 100%">
          <el-option label="VO 主导模式" value="VO主导模式" />
          <el-option label="对话主导模式" value="对话主导模式" />
          <el-option label="混合模式" value="混合模式" />
          <el-option label="动作/视觉模式" value="动作/视觉模式" />
        </el-select>
        <div class="help-text">
          <p>• VO 主导模式：旁白 &gt; 50%，旁白不切碎，微动为主</p>
          <p>• 对话主导模式：对话 &gt; 50%，每句对话可单独成镜</p>
          <p>• 混合模式（默认）：旁白和对话各半</p>
          <p>• 动作/视觉模式：旁白极少，靠画面讲故事</p>
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button size="large" @click="handleClose">
          {{ $t("common.cancel") }}
        </el-button>
        <el-button
          type="primary"
          size="large"
          :loading="loading"
          @click="handleSubmit"
        >
          <el-icon v-if="!loading"><Plus /></el-icon>
          {{ $t("drama.createNew") }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, watch, computed } from "vue";
import { useRouter } from "vue-router";
import { ElMessage, type FormInstance, type FormRules } from "element-plus";
import { Plus } from "@element-plus/icons-vue";
import { dramaAPI } from "@/api/drama";
import type { CreateDramaRequest } from "@/types/drama";

const PRESET_STYLE_VALUES = [
  "ghibli",
  "guoman",
  "wasteland",
  "nostalgia",
  "pixel",
  "voxel",
  "urban",
  "guoman3d",
  "chibi3d",
];

const stylePresets = [
  { value: "ghibli", labelKey: "drama.styles.ghibli" },
  { value: "guoman", labelKey: "drama.styles.guoman" },
  { value: "wasteland", labelKey: "drama.styles.wasteland" },
  { value: "nostalgia", labelKey: "drama.styles.nostalgia" },
  { value: "pixel", labelKey: "drama.styles.pixel" },
  { value: "voxel", labelKey: "drama.styles.voxel" },
  { value: "urban", labelKey: "drama.styles.urban" },
  { value: "guoman3d", labelKey: "drama.styles.guoman3d" },
  { value: "chibi3d", labelKey: "drama.styles.chibi3d" },
];

/**
 * CreateDramaDialog - Reusable dialog for creating new drama projects
 * 创建短剧弹窗 - 可复用的创建短剧项目弹窗
 */
const props = defineProps<{
  modelValue: boolean;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  created: [id: string];
}>();

const router = useRouter();
const formRef = ref<FormInstance>();
const loading = ref(false);

const customStyleInput = computed({
  get() {
    return PRESET_STYLE_VALUES.includes(form.style) ? "" : form.style;
  },
  set(v: string) {
    form.style = v || "ghibli";
  },
});

function selectPresetStyle(value: string) {
  form.style = value;
}

function onCustomStyleInput(value: string) {
  form.style = value?.trim() || "ghibli";
}

// v-model binding / 双向绑定
const visible = ref(props.modelValue);
watch(
  () => props.modelValue,
  (val) => {
    visible.value = val;
  },
);
watch(visible, (val) => {
  emit("update:modelValue", val);
});

// Form data / 表单数据
const form = reactive<CreateDramaRequest>({
  title: "",
  description: "",
  style: "ghibli",
  drama_mode: "混合模式",
});

// Validation rules / 验证规则
const rules: FormRules = {
  title: [
    { required: true, message: "请输入项目标题", trigger: "blur" },
    {
      min: 1,
      max: 100,
      message: "标题长度在 1 到 100 个字符",
      trigger: "blur",
    },
  ],
  style: [
    { required: true, message: "请选择或输入风格", trigger: "change" },
    { min: 1, message: "风格不能为空", trigger: "change" },
  ],
};

// Reset form when dialog closes / 关闭时重置表单
const handleClosed = () => {
  form.title = "";
  form.description = "";
  form.drama_mode = "混合模式";
  formRef.value?.resetFields();
};

// Close dialog / 关闭弹窗
const handleClose = () => {
  visible.value = false;
};

// Submit form / 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return;

  await formRef.value.validate(async (valid) => {
    if (valid) {
      loading.value = true;
      try {
        const drama = await dramaAPI.create(form);
        ElMessage.success("创建成功");
        visible.value = false;
        emit("created", drama.id);
        // Navigate to drama detail page / 跳转到短剧详情页
        router.push(`/dramas/${drama.id}`);
      } catch (error: any) {
        ElMessage.error(error.message || "创建失败");
      } finally {
        loading.value = false;
      }
    }
  });
};
</script>

<style scoped>
/* ========================================
   Dialog Styles / 弹窗样式
   ======================================== */
.create-dialog :deep(.el-dialog) {
  border-radius: var(--radius-xl);
}

.create-dialog :deep(.el-dialog__header) {
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border-primary);
  margin-right: 0;
}

.create-dialog :deep(.el-dialog__title) {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-primary);
}

.create-dialog :deep(.el-dialog__body) {
  padding: 1.5rem;
}

.dialog-desc {
  margin-bottom: 1.5rem;
  font-size: 0.875rem;
  color: var(--text-secondary);
}

/* ========================================
   Form Styles / 表单样式
   ======================================== */
.create-form :deep(.el-form-item) {
  margin-bottom: 1.25rem;
}

.create-form :deep(.el-form-item__label) {
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 0.5rem;
}

.create-form :deep(.el-input__wrapper),
.create-form :deep(.el-textarea__inner) {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  box-shadow: 0 0 0 1px var(--border-primary) inset;
  transition: all var(--transition-fast);
}

.create-form :deep(.el-input__wrapper:hover),
.create-form :deep(.el-textarea__inner:hover) {
  box-shadow: 0 0 0 1px var(--border-secondary) inset;
}

.create-form :deep(.el-input__wrapper.is-focus),
.create-form :deep(.el-textarea__inner:focus) {
  box-shadow: 0 0 0 2px var(--accent) inset;
}

.create-form :deep(.el-input__inner),
.create-form :deep(.el-textarea__inner) {
  color: var(--text-primary);
}

.create-form :deep(.el-input__inner::placeholder),
.create-form :deep(.el-textarea__inner::placeholder) {
  color: var(--text-muted);
}

.create-form :deep(.el-input__count) {
  color: var(--text-muted);
  background: transparent;
}

.style-selector {
  width: 100%;
}

.style-presets {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.style-preset-btn {
  padding: 6px 14px;
  font-size: 13px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-primary);
  background: var(--bg-secondary);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.style-preset-btn:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.style-preset-btn.active {
  border-color: var(--accent);
  background: var(--accent-light);
  color: var(--accent);
}

.style-custom :deep(.el-input__wrapper) {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  box-shadow: 0 0 0 1px var(--border-primary) inset;
}

/* ========================================
   Footer Styles / 底部样式
   ======================================== */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

.dialog-footer .el-button {
  min-width: 100px;
}

.help-text {
  margin-top: 8px;
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
}

.help-text p {
  margin: 4px 0;
}
</style>
