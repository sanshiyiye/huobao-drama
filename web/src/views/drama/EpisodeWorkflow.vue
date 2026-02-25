<template>
  <div class="page-container">
    <div class="content-wrapper animate-fade-in">
      <AppHeader :fixed="false" :show-logo="false">
        <template #left>
          <div class="episode-header-left">
            <el-button text @click="$router.back()" class="back-btn-icon">
              <el-icon><ArrowLeft /></el-icon>
            </el-button>
            <span class="episode-header-title">{{ drama?.title || $t("workflow.episodeProduction", { number: episodeNumber }) }}</span>
            <span class="episode-header-subtitle">{{ $t("workflow.episodeLabel", { number: episodeNumber }) }}</span>
          </div>
        </template>
        <template #center>
          <div class="workflow-steps-strip">
            <div
              v-for="step in workflowStepList"
              :key="step.name"
              class="workflow-step-item"
              :class="{ active: currentStep === step.name, completed: isStepCompleted(step.index) }"
              @click="handleStepClick(step.name)"
            >
              <span class="workflow-step-circle">
                <el-icon v-if="isStepCompleted(step.index)"><Check /></el-icon>
              </span>
              <span class="workflow-step-label">{{ step.label }}</span>
            </div>
          </div>
        </template>
        <template #right>
          <el-button
            :icon="Setting"
            @click="showModelConfigDialog"
            :title="$t('workflow.modelConfig')"
          >
            图文配置
          </el-button>
        </template>
      </AppHeader>

      <div class="content-container">
        <!-- 阶段 0: 章节内容 + 提取角色场景 -->
        <el-card
          v-show="currentStep === '0'"
          shadow="never"
          class="stage-card stage-card-fullscreen"
        >
          <div class="stage-body stage-body-fullscreen">
            <!-- 未保存时显示输入框 -->
            <div v-if="!hasScript" class="generation-form">
              <el-input
                v-model="scriptContent"
                type="textarea"
                :placeholder="$t('workflow.scriptPlaceholder')"
                class="script-textarea script-textarea-fullscreen"
              />

              <div class="action-buttons-inline">
                <el-button
                  type="primary"
                  size="default"
                  @click="saveChapterScript"
                  :disabled="!scriptContent.trim() || generatingScript"
                >
                  <el-icon><Check /></el-icon>
                  <span>{{ $t("workflow.saveChapter") }}</span>
                </el-button>
              </div>
            </div>

            <!-- 已保存时显示内容 -->
            <div v-if="hasScript" class="overview-section">
              <div class="episode-editor-header">
                <div class="episode-editor-title">
                  <h3>
                    {{ $t("workflow.chapterContent", { number: episodeNumber }) }}
                  </h3>
                  <span class="save-status-text">({{ $t("workflow.saved") }})</span>
                </div>
                <div class="episode-editor-actions">
                  <el-button
                    v-if="!isEditing"
                    size="small"
                    class="editor-action-btn"
                    @click="startEdit"
                  >
                    <el-icon><Edit /></el-icon>
                    编辑剧本
                  </el-button>
                  <template v-if="!isEditing">
                    <el-button
                      size="small"
                      class="editor-action-btn extract-action-btn"
                      @click="handleExtractCharactersAndBackgrounds"
                      :loading="extractingCharactersAndBackgrounds"
                      :disabled="!hasScript"
                    >
                      <el-icon><MagicStick /></el-icon>
                      提取角色和场景
                    </el-button>
                    <el-button
                      size="small"
                      class="editor-action-btn extract-action-btn"
                      @click="openExtractStyleDialog"
                      :loading="extractingStyle"
                      :disabled="!hasScript"
                    >
                      <el-icon><MagicStick /></el-icon>
                      提取风格
                    </el-button>
                    <el-button
                      size="small"
                      class="editor-action-btn next-step-btn"
                      @click="nextStep"
                      :disabled="!hasExtractedData"
                    >
                      下一步
                    </el-button>
                  </template>
                  <template v-else>
                    <el-button
                      type="success"
                      size="small"
                      class="editor-action-btn"
                      @click="saveEdit"
                      :loading="saving"
                    >
                      <el-icon><Check /></el-icon>
                      保存
                    </el-button>
                    <el-button
                      size="small"
                      class="editor-action-btn"
                      @click="cancelEdit"
                    >
                      <el-icon><Close /></el-icon>
                      取消
                    </el-button>
                  </template>
                </div>
              </div>

              <div class="script-content-panel">
                <el-input
                  v-if="!isEditing"
                  :model-value="currentEpisode?.script_content || ''"
                  type="textarea"
                  :rows="14"
                  readonly
                  disabled
                  class="script-readonly-input"
                />
                <el-input
                  v-else
                  v-model="displayScriptContent"
                  type="textarea"
                  :rows="14"
                  class="script-editor-input"
                />
              </div>

              <el-divider />

              <!-- 显示已提取的角色和场景 -->
              <div v-if="hasExtractedData" class="extracted-info">
                <el-alert
                  type="success"
                  :closable="false"
                  style="margin-bottom: 16px"
                >
                  <template #title>
                    <div style="display: flex; align-items: center; gap: 16px">
                      <span>✅ {{ $t("workflow.extractedData") }}</span>
                      <el-tag v-if="hasCharacters" type="success"
                        >{{ $t("workflow.characters") }}:
                        {{ charactersCount }}</el-tag
                      >
                      <el-tag v-if="currentEpisode?.scenes" type="success"
                        >{{ $t("workflow.scenes") }}:
                        {{ currentEpisode.scenes.length }}</el-tag
                      >
                      <el-tag v-if="drama?.style" type="success"
                        >{{ $t("workflow.style") }}:
                        {{ drama.style }}</el-tag
                      >
                    </div>
                  </template>
                </el-alert>

                <!-- 角色列表 -->
                <div v-if="hasCharacters" style="margin-bottom: 16px">
                  <h4 class="extracted-title">
                    {{ $t("workflow.extractedCharacters") }}：
                  </h4>
                  <div style="display: flex; flex-wrap: wrap; gap: 8px">
                    <el-tag
                      v-for="char in currentEpisode?.characters"
                      :key="char.id"
                      type="info"
                    >
                      {{ char.name }}
                      <span v-if="char.role" class="secondary-text"
                        >({{ char.role }})</span
                      >
                    </el-tag>
                  </div>
                </div>

                <!-- 场景列表 -->
                <div
                  v-if="
                    currentEpisode?.scenes && currentEpisode.scenes.length > 0
                  "
                >
                  <h4 class="extracted-title">
                    {{ $t("workflow.extractedScenes") }}：
                  </h4>
                  <div style="display: flex; flex-wrap: wrap; gap: 8px">
                    <el-tag
                      v-for="scene in currentEpisode.scenes"
                      :key="scene.id"
                      type="warning"
                    >
                      {{ scene.location }}
                      <span class="secondary-text">· {{ scene.time }}</span>
                    </el-tag>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </el-card>

        <!-- 阶段 1: 生成图片 -->
        <el-card v-show="currentStep === '1'" class="workflow-card">
          <div class="stage-body">
            <!-- 未完成阶段1时显示空内容 -->
            <div v-if="!hasExtractedData" class="empty-stage">
              <el-empty
                description="请先在章节内容阶段完成角色和场景提取"
                :image-size="120"
              >
                <el-button type="primary" @click="currentStep = '0'">
                  <el-icon><ArrowLeft /></el-icon>
                  前往章节内容阶段
                </el-button>
              </el-empty>
            </div>

            <!-- 角色图片生成 -->
            <div v-else>
            <div class="image-gen-section">
              <div class="section-header">
                <div class="section-title">
                  <h3>
                    <el-icon><User /></el-icon>
                    {{ $t("workflow.characterImages") }}
                  </h3>
                  <el-alert type="info" :closable="false" style="margin: 0">
                    {{
                      $t("workflow.characterCount", { count: charactersCount })
                    }}
                  </el-alert>
                </div>
                <div class="section-actions">
                  <el-checkbox
                    v-model="selectAllCharacters"
                    @change="toggleSelectAllCharacters"
                    style="margin-right: 12px"
                  >
                    {{ $t("workflow.selectAll") }}
                  </el-checkbox>
                  <el-button
                    type="primary"
                    @click="batchGenerateCharacterImages"
                    :loading="batchGeneratingCharacters"
                    :disabled="selectedCharacterIds.length === 0"
                    size="default"
                  >
                    {{ $t("workflow.batchGenerate") }} ({{
                      selectedCharacterIds.length
                    }})
                  </el-button>
                </div>
              </div>

              <div class="character-image-list">
                <div
                  v-for="char in currentEpisode?.characters"
                  :key="char.id"
                  class="character-item"
                >
                  <el-card shadow="hover" class="fixed-card">
                    <div class="card-header">
                      <el-checkbox
                        v-model="selectedCharacterIds"
                        :value="char.id"
                        style="margin-right: 8px"
                      />
                      <div class="header-left">
                        <h4>{{ char.name }}</h4>
                        <el-tag size="small">{{ char.role }}</el-tag>
                        <el-tag v-if="char.image_ref" size="small" type="info">{{ char.image_ref }}</el-tag>
                      </div>
                      <el-button
                        type="danger"
                        size="small"
                        :icon="Delete"
                        circle
                        @click="deleteCharacter(char.id)"
                        :title="$t('workflow.deleteCharacter')"
                      />
                    </div>

                    <div class="card-image-container">
                      <div v-if="hasImage(char)" class="char-image">
                        <el-image :src="getImageUrl(char)" fit="cover" />
                      </div>
                      <div
                        v-else-if="
                          char.image_generation_status === 'pending' ||
                          char.image_generation_status === 'processing' ||
                          generatingCharacterImages[char.id]
                        "
                        class="char-placeholder generating"
                      >
                        <el-icon :size="64" class="rotating"
                          ><Loading
                        /></el-icon>
                        <span>{{ $t("common.generating") }}</span>
                        <el-tag
                          type="warning"
                          size="small"
                          style="margin-top: 8px"
                          >{{
                            char.image_generation_status === "pending"
                              ? $t("common.queuing")
                              : $t("common.processing")
                          }}</el-tag
                        >
                      </div>
                      <div
                        v-else-if="char.image_generation_status === 'failed'"
                        class="char-placeholder failed"
                      >
                        <el-icon :size="64"><WarningFilled /></el-icon>
                        <span>{{ $t("common.generateFailed") }}</span>
                        <el-tag
                          type="danger"
                          size="small"
                          style="margin-top: 8px"
                          >{{ $t("common.clickToRegenerate") }}</el-tag
                        >
                      </div>
                      <div v-else class="char-placeholder">
                        <el-icon :size="64"><User /></el-icon>
                        <span>{{ $t("common.notGenerated") }}</span>
                      </div>
                    </div>

                    <div class="card-actions">
                      <el-tooltip
                        :content="$t('workflow.characterDesign')"
                        placement="top"
                      >
                        <el-button
                          size="small"
                          @click="openCharacterDesignDialog(char)"
                          :icon="Edit"
                          circle
                        />
                      </el-tooltip>
                      <el-tooltip
                        :content="$t('tooltip.aiGenerate')"
                        placement="top"
                      >
                        <el-button
                          type="primary"
                          size="small"
                          @click="generateCharacterImage(char.id)"
                          :loading="generatingCharacterImages[char.id]"
                          :icon="MagicStick"
                          circle
                        />
                      </el-tooltip>
                      <el-tooltip
                        :content="$t('tooltip.uploadImage')"
                        placement="top"
                      >
                        <el-button
                          size="small"
                          @click="uploadCharacterImage(char.id)"
                          :icon="Upload"
                          circle
                        />
                      </el-tooltip>
                      <el-tooltip
                        :content="$t('tooltip.selectFromLibrary')"
                        placement="top"
                      >
                        <el-button
                          size="small"
                          @click="selectFromLibrary(char.id)"
                          :icon="Picture"
                          circle
                        />
                      </el-tooltip>
                      <el-tooltip
                        :content="$t('workflow.addToLibrary')"
                        placement="top"
                      >
                        <el-button
                          size="small"
                          @click="addToCharacterLibrary(char)"
                          :icon="FolderAdd"
                          :disabled="!char.image_url"
                          circle
                        />
                      </el-tooltip>
                    </div>
                  </el-card>
                </div>
              </div>
            </div>

            <el-divider />

            <!-- 场景图片生成 -->
            <div class="image-gen-section">
              <div class="section-header">
                <div class="section-title">
                  <h3>
                    <el-icon><Place /></el-icon>
                    {{ $t("workflow.sceneImages") }}
                  </h3>
                  <el-alert type="info" :closable="false" style="margin: 0">
                    {{
                      $t("workflow.sceneCount", {
                        count: currentEpisode?.scenes?.length || 0,
                      })
                    }}
                  </el-alert>
                </div>
                <div class="section-actions">
                  <!-- <el-button
                  :icon="Document"
                  @click="openExtractSceneDialog"
                  size="default"
                >
                  {{ $t("workflow.extractFromScript") }}
                </el-button> -->
                  <el-checkbox
                    v-model="selectAllScenes"
                    @change="toggleSelectAllScenes"
                    style="margin-left: 12px; margin-right: 12px"
                  >
                    {{ $t("workflow.selectAll") }}
                  </el-checkbox>
                  <el-button
                    type="primary"
                    @click="batchGenerateSceneImages"
                    :loading="batchGeneratingScenes"
                    :disabled="selectedSceneIds.length === 0"
                    size="default"
                  >
                    {{ $t("workflow.batchGenerateSelected") }} ({{
                      selectedSceneIds.length
                    }})
                  </el-button>

                  <el-button
                    :icon="Plus"
                    @click="openAddSceneDialog"
                    size="default"
                  >
                    {{ $t("workflow.addScene") }}
                  </el-button>
                </div>
              </div>

              <div class="scene-image-list">
                <div
                  v-for="scene in currentEpisode?.scenes"
                  :key="scene.id"
                  class="scene-item"
                >
                  <el-card shadow="hover" class="fixed-card">
                    <div class="card-header">
                      <el-checkbox
                        v-model="selectedSceneIds"
                        :value="scene.id"
                        style="margin-right: 8px"
                      />
                      <div class="header-left">
                        <h4>{{ scene.location }}</h4>
                        <el-tag size="small">{{ scene.time }}</el-tag>
                        <el-tag v-if="scene.image_ref" size="small" type="info">{{ scene.image_ref }}</el-tag>
                      </div>
                      <el-button
                        type="danger"
                        size="small"
                        :icon="Delete"
                        circle
                        @click="deleteScene(scene.id)"
                        :title="$t('workflow.deleteScene')"
                      />
                    </div>

                    <div class="card-image-container">
                      <div v-if="hasImage(scene)" class="scene-image">
                        <el-image :src="getImageUrl(scene)" fit="cover" />
                      </div>
                      <div
                        v-else-if="
                          scene.image_generation_status === 'pending' ||
                          scene.image_generation_status === 'processing' ||
                          generatingSceneImages[scene.id]
                        "
                        class="scene-placeholder generating"
                      >
                        <el-icon :size="64" class="rotating"
                          ><Loading
                        /></el-icon>
                        <span>{{ $t("common.generating") }}</span>
                        <el-tag
                          type="warning"
                          size="small"
                          style="margin-top: 8px"
                          >{{
                            scene.image_generation_status === "pending"
                              ? $t("common.queuing")
                              : $t("common.processing")
                          }}</el-tag
                        >
                      </div>
                      <div
                        v-else-if="scene.image_generation_status === 'failed'"
                        class="scene-placeholder failed"
                        @click="generateSceneImage(scene.id)"
                        style="cursor: pointer"
                      >
                        <el-icon :size="64"><WarningFilled /></el-icon>
                        <span>{{ $t("common.generateFailed") }}</span>
                        <el-tag
                          type="danger"
                          size="small"
                          style="margin-top: 8px"
                          >{{ $t("common.clickToRegenerate") }}</el-tag
                        >
                      </div>
                      <div v-else class="scene-placeholder">
                        <el-icon :size="64"><Place /></el-icon>
                        <span>{{ $t("common.notGenerated") }}</span>
                      </div>
                    </div>

                    <div class="card-actions">
                      <el-tooltip
                        :content="$t('tooltip.editPrompt')"
                        placement="top"
                      >
                        <el-button
                          size="small"
                          @click="openPromptDialog(scene, 'scene')"
                          :icon="Edit"
                          circle
                        />
                      </el-tooltip>
                      <el-tooltip
                        :content="$t('tooltip.aiGenerate')"
                        placement="top"
                      >
                        <el-button
                          type="primary"
                          size="small"
                          @click="generateSceneImage(scene.id)"
                          :loading="generatingSceneImages[scene.id]"
                          :icon="MagicStick"
                          circle
                        />
                      </el-tooltip>
                      <el-tooltip
                        :content="$t('tooltip.uploadImage')"
                        placement="top"
                      >
                        <el-button
                          size="small"
                          @click="uploadSceneImage(scene.id)"
                          :icon="Upload"
                          circle
                        />
                      </el-tooltip>
                    </div>
                  </el-card>
                </div>
              </div>
            </div>
            </div> <!-- 阶段1内容结束 -->
          </div>
        </el-card>

        <!-- 阶段 2: 拆分分镜 -->
        <el-card v-show="currentStep === '2'" shadow="never" class="stage-card">
          <div class="stage-body">
            <!-- 未完成阶段2时显示空内容 -->
            <div v-if="!allImagesGenerated" class="empty-stage">
              <el-empty
                description="请先生成所有角色和场景图片后再进行分镜拆分"
                :image-size="120"
              >
                <el-button type="primary" @click="currentStep = '1'">
                  <el-icon><ArrowLeft /></el-icon>
                  前往生成图片阶段
                </el-button>
              </el-empty>
            </div>

            <div v-else>
            <!-- 分镜列表（卡片样式：系统提示词可编辑，关联角色/关联场景展示） -->
            <div
              v-if="
                currentEpisode?.storyboards &&
                currentEpisode.storyboards.length > 0
              "
              class="shots-list shots-list-cards"
            >
              <div class="shots-header">
                <h3>{{ $t("workflow.shotList") }}</h3>
                <span class="shots-summary">
                  {{ $t("workflow.shotCount", { count: currentEpisode.storyboards.length }) }}
                  <template v-if="currentEpisode?.characters?.length">
                    | {{ currentEpisode.characters.length }}{{ $t("workflow.charactersUnit") }}
                  </template>
                  <template v-if="currentEpisode?.scenes?.length">
                    | {{ currentEpisode.scenes.length }}{{ $t("workflow.scenesUnit") }}
                  </template>
                </span>
              </div>

              <div class="shot-card-list">
                <div
                  v-for="(shot, index) in currentEpisode.storyboards"
                  :key="shot.id"
                  class="shot-card"
                >
                  <div class="shot-card-left">
                    <span class="shot-number-circle">{{ index + 1 }}</span>
                  </div>
                  <div class="shot-card-main">
                    <div class="shot-card-title-row">
                      <span class="shot-title-text">{{ $t("workflow.shotNumberTitle", { n: index + 1, title: shot.title || $t("workflow.untitledShot") }) }}</span>
                    </div>
                    <div class="shot-card-prompt-row">
                      <span class="shot-char-count">{{ (shot.video_prompt || '').length }}/500</span>
                    </div>
                    <el-input
                      v-model="shot.video_prompt"
                      type="textarea"
                      :rows="3"
                      :maxlength="500"
                      show-word-limit
                      :placeholder="$t('workflow.videoPromptPlaceholder')"
                      class="shot-system-prompt-input"
                      @blur="saveShotSystemPrompt(shot)"
                    />
                    <div class="shot-meta">
                      <div class="shot-meta-row">
                        <span class="shot-label">{{ $t("workflow.associatedCharacters") }}:</span>
                        <template v-if="getShotCharacterNames(shot).length > 0">
                          <el-tag
                            v-for="name in getShotCharacterNames(shot)"
                            :key="name"
                            size="small"
                            class="shot-tag shot-tag-character"
                          >
                            {{ name }}
                          </el-tag>
                        </template>
                        <el-button
                          v-else
                          type="primary"
                          link
                          size="small"
                          @click="openBindCharacter(shot, index)"
                        >
                          + {{ $t("workflow.clickToBindCharacter") }}
                        </el-button>
                      </div>
                      <div class="shot-meta-row">
                        <span class="shot-label">{{ $t("workflow.associatedScenes") }}:</span>
                        <template v-if="getShotScene(shot)">
                          <el-tag size="small" class="shot-tag shot-tag-scene">
                            <el-icon><Location /></el-icon>
                            {{ getShotScene(shot)?.location || getShotScene(shot)?.title || getShotScene(shot)?.id }}
                          </el-tag>
                        </template>
                        <el-button
                          v-else
                          type="primary"
                          link
                          size="small"
                          @click="openBindScene(shot, index)"
                        >
                          + {{ $t("workflow.clickToBindScene") }}
                        </el-button>
                      </div>
                    </div>
                  </div>
                  <div class="shot-card-actions">
                    <el-button
                      type="danger"
                      link
                      :icon="Delete"
                      @click="deleteShot(shot, index)"
                    >
                      {{ $t("common.delete") }}
                    </el-button>
                  </div>
                </div>
              </div>
            </div>

            <!-- 未拆分时显示 -->
            <div v-else class="empty-shots">
              <el-empty :description="$t('workflow.splitStoryboardFirst')">
                <el-button
                  type="primary"
                  @click="generateShots"
                  :loading="generatingShots"
                  :icon="MagicStick"
                >
                  {{
                    generatingShots
                      ? $t("workflow.aiSplitting")
                      : $t("workflow.aiAutoSplit")
                  }}
                </el-button>

                <!-- 任务进度显示 -->
                <div
                  v-if="generatingShots"
                  style="
                    margin-top: 24px;
                    max-width: 400px;
                    margin-left: auto;
                    margin-right: auto;
                  "
                >
                  <el-progress
                    :percentage="taskProgress"
                    :status="taskProgress === 100 ? 'success' : undefined"
                  >
                    <template #default="{ percentage }">
                      <span style="font-size: 12px">{{ percentage }}%</span>
                    </template>
                  </el-progress>
                  <div class="task-message">
                    {{ taskMessage }}
                  </div>
                </div>
              </el-empty>
            </div>
            </div> <!-- 阶段2内容结束 -->
          </div>
        </el-card>

        <!-- 阶段 3: 专业制作（内嵌，不跳转） -->
        <div v-show="currentStep === '3'" class="stage-card stage-card-fullscreen professional-embed">
          <ProfessionalEditor
            v-if="currentEpisode?.id"
            ref="professionalEditorRef"
            :drama-id="dramaId"
            :episode-number="episodeNumber"
            :episode-id="currentEpisode.id"
            embed-mode
          />
          <el-empty v-else :description="$t('workflow.loadingEpisode')" />
        </div>
      </div>

      <div class="actions-container" v-show="currentStep !== '0' && currentStep !== '3'">
        <div class="action-buttons" v-show="currentStep === '1'">
          <el-button size="large" @click="prevStep">
            <el-icon><ArrowLeft /></el-icon>
            {{ $t("workflow.prevStep") }}
          </el-button>
          <el-button
            type="success"
            size="large"
            @click="nextStep"
            :disabled="!allImagesGenerated"
          >
            {{ $t("workflow.nextStepSplitShots") }}
            <el-icon><ArrowRight /></el-icon>
          </el-button>
          <div v-if="!allImagesGenerated" style="margin-top: 8px">
            <el-alert
              type="warning"
              :closable="false"
              style="display: inline-block"
            >
              <template #title>
                <span style="font-size: 12px">
                  {{ $t("workflow.generateAllImagesFirst") }}
                </span>
              </template>
            </el-alert>
          </div>
        </div>

        <div class="action-buttons" v-show="currentStep === '2'">
          <el-button size="large" @click="prevStep">
            <el-icon><ArrowLeft /></el-icon>
            {{ $t("workflow.prevStep") }}
          </el-button>
          <el-button size="large" @click="regenerateShots" :icon="MagicStick" :loading="generatingShots">
            {{ generatingShots ? $t("workflow.aiSplitting") : $t("workflow.reSplitShots") }}
          </el-button>
          <el-button size="large" @click="goToProfessionalUI">
            {{ $t("workflow.nextStep") }}
            <el-icon><ArrowRight /></el-icon>
          </el-button>
        </div>
      </div>
    </div>

    <div class="components-box">
      <!-- 镜头编辑对话框 -->
      <el-dialog
        v-model="shotEditDialogVisible"
        :title="$t('workflow.editShot')"
        width="800px"
        :close-on-click-modal="false"
      >
        <el-form v-if="editingShot" label-width="100px" size="default">
          <el-form-item :label="$t('workflow.shotTitle')">
            <el-input
              v-model="editingShot.title"
              :placeholder="$t('workflow.shotTitlePlaceholder')"
            />
          </el-form-item>

          <el-row :gutter="16">
            <el-col :span="8">
              <el-form-item :label="$t('workflow.shotType')">
                <el-select
                  v-model="editingShot.shot_type"
                  :placeholder="$t('workflow.selectShotType')"
                >
                  <el-option :label="$t('workflow.longShot')" value="远景" />
                  <el-option :label="$t('workflow.fullShot')" value="全景" />
                  <el-option :label="$t('workflow.mediumShot')" value="中景" />
                  <el-option :label="$t('workflow.closeUp')" value="近景" />
                  <el-option
                    :label="$t('workflow.extremeCloseUp')"
                    value="特写"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item :label="$t('workflow.cameraAngle')">
                <el-select
                  v-model="editingShot.angle"
                  :placeholder="$t('workflow.selectAngle')"
                >
                  <el-option :label="$t('workflow.eyeLevel')" value="平视" />
                  <el-option :label="$t('workflow.lowAngle')" value="仰视" />
                  <el-option :label="$t('workflow.highAngle')" value="俯视" />
                  <el-option :label="$t('workflow.sideView')" value="侧面" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item :label="$t('workflow.cameraMovement')">
                <el-select
                  v-model="editingShot.movement"
                  :placeholder="$t('workflow.selectMovement')"
                >
                  <el-option
                    :label="$t('workflow.staticShot')"
                    value="固定镜头"
                  />
                  <el-option :label="$t('workflow.pushIn')" value="推镜" />
                  <el-option :label="$t('workflow.pullOut')" value="拉镜" />
                  <el-option :label="$t('workflow.followShot')" value="跟镜" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="16">
            <el-col :span="12">
              <el-form-item :label="$t('workflow.location')">
                <el-input
                  v-model="editingShot.location"
                  :placeholder="$t('workflow.locationPlaceholder')"
                />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item :label="$t('workflow.time')">
                <el-input
                  v-model="editingShot.time"
                  :placeholder="$t('workflow.timeSetting')"
                />
              </el-form-item>
            </el-col>
          </el-row>

          <el-form-item :label="$t('workflow.shotDescription')">
            <el-input
              v-model="editingShot.description"
              type="textarea"
              :rows="2"
              :placeholder="$t('workflow.shotDescriptionPlaceholder')"
            />
          </el-form-item>

          <el-form-item :label="$t('workflow.actionDescription')">
            <el-input
              v-model="editingShot.action"
              type="textarea"
              :rows="3"
              :placeholder="$t('workflow.detailedAction')"
            />
          </el-form-item>

          <el-form-item :label="$t('workflow.dialogue')">
            <el-input
              v-model="editingShot.dialogue"
              type="textarea"
              :rows="2"
              :placeholder="$t('workflow.characterDialogue')"
            />
          </el-form-item>

          <el-form-item :label="$t('workflow.result')">
            <el-input
              v-model="editingShot.result"
              type="textarea"
              :rows="2"
              :placeholder="$t('workflow.actionResult')"
            />
          </el-form-item>

          <el-form-item :label="$t('workflow.atmosphere')">
            <el-input
              v-model="editingShot.atmosphere"
              type="textarea"
              :rows="2"
              :placeholder="$t('workflow.atmosphereDescription')"
            />
          </el-form-item>


          <el-form-item :label="$t('workflow.videoPrompt')">
            <el-input
              v-model="editingShot.video_prompt"
              type="textarea"
              :rows="3"
              :placeholder="$t('workflow.videoPromptPlaceholder')"
            />
          </el-form-item>

          <el-form-item :label="$t('workflow.characters')">
            <el-select
              v-model="editingShot.characters"
              multiple
              filterable
              allow-create
              :placeholder="$t('workflow.selectCharacters')"
              collapse-tags
            >
              <el-option
                v-for="char in currentEpisode?.characters || []"
                :key="char.id"
                :label="char.name"
                :value="char.id"
              />
            </el-select>
          </el-form-item>

          <el-row :gutter="16">
            <el-col :span="12">
              <el-form-item :label="$t('workflow.bgmHint')">
                <el-input
                  v-model="editingShot.bgm_prompt"
                  :placeholder="$t('workflow.bgmAtmosphere')"
                />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item :label="$t('workflow.soundEffect')">
                <el-input
                  v-model="editingShot.sound_effect"
                  :placeholder="$t('workflow.soundEffectDescription')"
                />
              </el-form-item>
            </el-col>
          </el-row>

          <el-form-item :label="$t('workflow.durationSeconds')">
            <el-input-number
              v-model="editingShot.duration"
              :min="1"
              :max="60"
            />
          </el-form-item>
        </el-form>

        <template #footer>
          <el-button @click="shotEditDialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button
            type="primary"
            @click="saveShotEdit"
            :loading="savingShot"
            >{{ $t("common.save") }}</el-button
          >
        </template>
      </el-dialog>

      <!-- 绑定角色与场景（简易弹窗，替代完整编辑页） -->
      <el-dialog
        v-model="bindDialogVisible"
        :title="$t('workflow.associatedCharacters') + ' / ' + $t('workflow.associatedScenes')"
        width="480px"
      >
        <el-form label-width="100px">
          <el-form-item :label="$t('workflow.associatedCharacters')">
            <el-select
              v-model="bindForm.characterIds"
              multiple
              filterable
              :placeholder="$t('workflow.selectCharacters')"
              style="width: 100%"
            >
              <el-option
                v-for="c in currentEpisode?.characters || []"
                :key="c.id"
                :label="c.name"
                :value="c.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item :label="$t('workflow.associatedScenes')">
            <el-select
              v-model="bindForm.scene_id"
              clearable
              :placeholder="$t('workflow.selectScene')"
              style="width: 100%"
            >
              <el-option
                v-for="s in currentEpisode?.scenes || []"
                :key="s.id"
                :label="s.location || s.title || s.id"
                :value="Number(s.id)"
              />
            </el-select>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="bindDialogVisible = false">{{ $t("common.cancel") }}</el-button>
          <el-button type="primary" @click="saveBindDialog">{{ $t("common.save") }}</el-button>
        </template>
      </el-dialog>

      <!-- 提示词编辑对话框 -->
      <el-dialog
        v-model="promptDialogVisible"
        :title="$t('workflow.editPrompt')"
        width="600px"
      >
        <el-form label-width="80px">
          <el-form-item :label="$t('common.name')">
            <el-input v-model="currentEditItem.name" disabled />
          </el-form-item>
          <el-form-item
            v-if="currentEditType === 'scene'"
            :label="$t('workflow.time')"
          >
            <el-input
              v-model="currentEditItem.time"
              :placeholder="$t('workflow.timePlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('workflow.imagePrompt')">
            <el-input
              v-model="editPrompt"
              type="textarea"
              :rows="6"
              :placeholder="$t('workflow.imagePromptPlaceholder')"
            />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="promptDialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button type="primary" @click="savePrompt">{{
            $t("common.save")
          }}</el-button>
        </template>
      </el-dialog>

      <!-- 角色设计对话框 -->
      <el-dialog
        v-model="characterDesignDialogVisible"
        :title="$t('workflow.characterDesign')"
        width="900px"
        class="character-design-dialog"
        destroy-on-close
      >
        <div class="character-design-layout">
          <div class="character-design-left">
            <div class="single-composite-view">
              <span class="view-label">{{ $t('workflow.singleCompositeThreeView') }}</span>
              <div class="view-image-wrap composite">
                <el-image
                  v-if="characterDesignTarget?.image_url"
                  :src="getImageUrl(characterDesignTarget)"
                  fit="contain"
                  class="view-image"
                />
                <div v-else class="view-placeholder">
                  <el-icon :size="40"><Picture /></el-icon>
                </div>
              </div>
            </div>
            <div class="character-design-actions">
              <el-button size="small" @click="regenerateCharacterImageFromDesign">
                {{ $t('workflow.changeOne') }}
              </el-button>
              <el-button size="small" disabled>{{ $t('workflow.history') }}</el-button>
              <el-button size="small" @click="uploadCharacterImageFromDesign">
                {{ $t('workflow.manualUpload') }}
              </el-button>
            </div>
          </div>
          <div class="character-design-right">
            <el-form label-width="90px" label-position="top">
              <el-form-item :label="$t('workflow.characterName')">
                <el-input v-model="characterDesignForm.name" :placeholder="$t('workflow.characterName')" />
              </el-form-item>
              <el-form-item :label="$t('workflow.age')">
                <el-select
                  v-model="characterDesignForm.age"
                  :placeholder="$t('workflow.selectAge')"
                  clearable
                  style="width: 100%"
                >
                  <el-option
                    v-for="opt in ageOptions"
                    :key="opt.value"
                    :label="opt.label"
                    :value="opt.value"
                  />
                </el-select>
              </el-form-item>
              <el-form-item :label="$t('workflow.gender')">
                <el-select
                  v-model="characterDesignForm.gender"
                  :placeholder="$t('workflow.selectGender')"
                  clearable
                  style="width: 100%"
                >
                  <el-option
                    v-for="opt in genderOptions"
                    :key="opt.value"
                    :label="opt.label"
                    :value="opt.value"
                  />
                </el-select>
              </el-form-item>
              <el-form-item :label="$t('workflow.characterDescription')">
                <el-input
                  v-model="characterDesignForm.appearance"
                  type="textarea"
                  :rows="5"
                  :placeholder="$t('workflow.characterDescriptionPlaceholder')"
                />
              </el-form-item>
              <el-form-item :label="$t('workflow.backgroundStory')">
                <el-input
                  v-model="characterDesignForm.description"
                  type="textarea"
                  :rows="4"
                  :placeholder="$t('workflow.backgroundStoryPlaceholder')"
                />
              </el-form-item>
            </el-form>
          </div>
        </div>
        <template #footer>
          <el-button @click="characterDesignDialogVisible = false">{{ $t('common.cancel') }}</el-button>
          <el-button type="primary" :loading="characterDesignSaving" @click="saveCharacterDesign">
            {{ $t('common.confirm') }}
          </el-button>
        </template>
      </el-dialog>

      <!-- 角色库选择对话框 -->
      <el-dialog
        v-model="libraryDialogVisible"
        :title="$t('workflow.selectFromLibrary')"
        width="800px"
      >
        <div class="library-grid">
          <div
            v-for="item in libraryItems"
            :key="item.id"
            class="library-item"
            @click="selectLibraryItem(item)"
          >
            <el-image :src="getImageUrl(item)" fit="cover" />
            <div class="library-item-name">{{ item.name }}</div>
          </div>
        </div>
        <div v-if="libraryItems.length === 0" class="empty-library">
          <el-empty :description="$t('workflow.emptyLibrary')" />
        </div>
      </el-dialog>

      <!-- AI模型配置对话框 -->
      <el-dialog
        v-model="modelConfigDialogVisible"
        :title="$t('workflow.aiModelConfig')"
        width="600px"
        :close-on-click-modal="false"
      >
        <el-form label-width="120px">
          <el-form-item :label="$t('workflow.textGenModel')">
            <el-select
              v-model="selectedTextModel"
              :placeholder="$t('workflow.selectTextModel')"
              style="width: 100%"
            >
              <el-option
                v-for="model in textModels"
                :key="model.modelName"
                :label="model.modelName"
                :value="model.modelName"
              />
            </el-select>
            <div class="model-tip">
              {{ $t("workflow.textModelTip") }}
            </div>
          </el-form-item>

          <el-form-item :label="$t('workflow.imageGenModel')">
            <el-select
              v-model="selectedImageModel"
              :placeholder="$t('workflow.selectImageModel')"
              style="width: 100%"
            >
              <el-option
                v-for="model in imageModels"
                :key="model.modelName"
                :label="model.modelName"
                :value="model.modelName"
              />
            </el-select>
            <div class="model-tip">
              {{ $t("workflow.modelConfigTip") }}
            </div>
          </el-form-item>
        </el-form>

        <template #footer>
          <el-button @click="modelConfigDialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button type="primary" @click="saveModelConfig">{{
            $t("common.saveConfig")
          }}</el-button>
        </template>
      </el-dialog>

      <!-- 图片上传对话框 -->
      <el-dialog
        v-model="uploadDialogVisible"
        :title="$t('tooltip.uploadImage')"
        width="500px"
      >
        <el-upload
          class="upload-area"
          drag
          :action="uploadAction"
          :headers="uploadHeaders"
          :on-success="handleUploadSuccess"
          :on-error="handleUploadError"
          :show-file-list="false"
          accept="image/jpeg,image/png,image/jpg"
        >
          <el-icon class="el-icon--upload"><Upload /></el-icon>
          <div class="el-upload__text">
            {{ $t("workflow.dragFilesHere")
            }}<em>{{ $t("workflow.clickToUpload") }}</em>
          </div>
          <template #tip>
            <div class="el-upload__tip">
              {{ $t("workflow.uploadFormatTip") }}
            </div>
          </template>
        </el-upload>
      </el-dialog>

      <!-- 添加场景对话框 -->
      <el-dialog
        v-model="addSceneDialogVisible"
        :title="$t('workflow.addScene')"
        width="600px"
      >
        <el-form :model="newScene" label-width="100px">
          <el-form-item :label="$t('workflow.sceneImage')">
            <el-upload
              class="avatar-uploader"
              :action="`/api/v1/upload/image`"
              :show-file-list="false"
              :on-success="handleSceneImageSuccess"
              :before-upload="beforeAvatarUpload"
            >
              <img
                v-if="hasImage(newScene)"
                :src="getImageUrl(newScene)"
                class="avatar"
                style="width: 160px; height: 90px; object-fit: cover"
              />
              <el-icon
                v-else
                class="avatar-uploader-icon"
                style="
                  border: 1px dashed #d9d9d9;
                  border-radius: 6px;
                  cursor: pointer;
                  position: relative;
                  overflow: hidden;
                  width: 160px;
                  height: 90px;
                  font-size: 28px;
                  color: #8c939d;
                  text-align: center;
                  line-height: 90px;
                "
                ><Plus
              /></el-icon>
            </el-upload>
          </el-form-item>
          <el-form-item :label="$t('workflow.sceneName')">
            <el-input
              v-model="newScene.location"
              :placeholder="$t('workflow.sceneNamePlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('workflow.time')">
            <el-input
              v-model="newScene.time"
              :placeholder="$t('workflow.timePlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('workflow.sceneDescription')">
            <el-input
              v-model="newScene.prompt"
              type="textarea"
              :rows="4"
              :placeholder="$t('workflow.sceneDescriptionPlaceholder')"
            />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="addSceneDialogVisible = false">{{
            $t("common.cancel")
          }}</el-button>
          <el-button type="primary" @click="saveScene">{{
            $t("common.confirm")
          }}</el-button>
        </template>
      </el-dialog>

      <!-- 从剧本提取场景对话框 -->
      <el-dialog
        v-model="extractScenesDialogVisible"
        :title="$t('workflow.extractSceneDialogTitle')"
        width="500px"
      >
        <el-alert type="info" :closable="false" style="margin-bottom: 16px">
          {{ $t("workflow.extractSceneDialogTip") }}
        </el-alert>
        <template #footer>
          <el-button @click="extractScenesDialogVisible = false">
            {{ $t("common.cancel") }}
          </el-button>
          <el-button
            type="primary"
            @click="handleExtractScenes"
            :loading="extractingScenes"
          >
            {{ $t("workflow.startExtract") }}
          </el-button>
        </template>
      </el-dialog>

      <!-- 风格提取对话框 -->
      <el-dialog
        v-model="extractStyleDialogVisible"
        title="提取风格"
        width="500px"
      >
        <el-alert type="info" :closable="false" style="margin-bottom: 16px">
          系统将自动分析剧本内容，提取剧情类型风格标签（如：古装、现代、科幻等）
        </el-alert>

        <el-form label-width="80px">
          <el-form-item label="提取结果">
            <el-input
              v-model="extractedStyle"
              placeholder="点击开始提取后，结果将显示在这里"
              :disabled="extractingStyle"
              type="textarea"
              :rows="3"
            />
          </el-form-item>
        </el-form>

        <template #footer>
          <el-button @click="extractStyleDialogVisible = false">
            {{ $t("common.cancel") }}
          </el-button>
          <el-button
            type="primary"
            @click="handleExtractStyle"
            :loading="extractingStyle"
            v-if="!extractedStyle"
          >
            {{ $t("workflow.startExtract") }}
          </el-button>
          <el-button
            type="success"
            @click="saveExtractedStyle"
            :loading="savingStyle"
            v-else
          >
            {{ $t("common.save") }}
          </el-button>
        </template>
      </el-dialog>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  User,
  Location,
  Picture,
  MagicStick,
  ArrowRight,
  ArrowLeft,
  Place,
  Film,
  Edit,
  More,
  Upload,
  Delete,
  FolderAdd,
  Setting,
  Loading,
  WarningFilled,
  Document,
  Plus,
} from "@element-plus/icons-vue";
import { dramaAPI } from "@/api/drama";
import { generationAPI } from "@/api/generation";
import { characterLibraryAPI } from "@/api/character-library";
import { aiAPI } from "@/api/ai";
import type { AIServiceConfig } from "@/types/ai";
import { imageAPI } from "@/api/image";
import type { Drama } from "@/types/drama";
import { AppHeader } from "@/components/common";
import ProfessionalEditor from "@/views/drama/ProfessionalEditor.vue";
import { getImageUrl, hasImage } from "@/utils/image";

const route = useRoute();
const router = useRouter();
const { t: $t } = useI18n();
const dramaId = route.params.id as string;
const episodeNumber = parseInt(route.params.episodeNumber as string);

const drama = ref<Drama>();

// 生成 localStorage key
const getStepStorageKey = () =>
  `episode_workflow_step_${dramaId}_${episodeNumber}`;

// 从 localStorage 恢复步骤，如果没有则默认为 0
const savedStep = localStorage.getItem(getStepStorageKey());
const currentStep = ref(savedStep ? savedStep : "0");
const scriptContent = ref("");
const generatingScript = ref(false);
const isEditing = ref(false);
const saving = ref(false);
const originalScriptContent = ref("");
/** 编辑时使用的剧本内容缓冲区，避免 readonly 切换导致输入框内容丢失 */
const editingScriptContent = ref("");
const generatingShots = ref(false);
const extractingCharactersAndBackgrounds = ref(false);
const batchGeneratingCharacters = ref(false);
const batchGeneratingScenes = ref(false);
const generatingCharacterImages = ref<Record<number, boolean>>({});
const generatingSceneImages = ref<Record<string, boolean>>({});

// 选择状态
const selectedCharacterIds = ref<number[]>([]);
const selectedSceneIds = ref<number[]>([]);
const selectAllCharacters = ref(false);
const selectAllScenes = ref(false);

// 对话框状态
const promptDialogVisible = ref(false);
const libraryDialogVisible = ref(false);
const uploadDialogVisible = ref(false);
const modelConfigDialogVisible = ref(false);
const addSceneDialogVisible = ref(false);
const extractScenesDialogVisible = ref(false);
const extractStyleDialogVisible = ref(false);
const extractingStyle = ref(false);
const savingStyle = ref(false);
const extractedStyle = ref("");

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

const selectPresetStyle = (value: string) => {
  selectedStyle.value = value;
  customStyleInput.value = "";
};

const openExtractStyleDialog = () => {
  extractStyleDialogVisible.value = true;
  // 打开对话框时设置已提取的风格值
  extractedStyle.value = drama.value?.plot_style || "";
};

const handleExtractStyle = async () => {
  if (!currentEpisode.value?.id) {
    ElMessage.error("章节信息不存在");
    return;
  }

  try {
    extractingStyle.value = true;
    // 调用风格提取 API，AI 自动分析剧本内容
    const response = await aiAPI.extractStyle(currentEpisode.value.id.toString());
    // 假设 API 返回格式 { task_id, message }，需要等待任务完成
    const taskId = response.task_id;
    ElMessage.success("风格提取任务已提交，正在处理...");

    // 轮询任务状态
    let taskStatus = "processing";
    let attempts = 0;
    const maxAttempts = 300; // 最多轮询 300 次（约 5 分钟）

    while (taskStatus === "processing" && attempts < maxAttempts) {
      await new Promise(resolve => setTimeout(resolve, 1000)); // 每秒检查一次
      const statusResponse = await generationAPI.getTaskStatus(taskId);
      taskStatus = statusResponse.status;
      attempts++;

      if (taskStatus === "completed") {
        // 任务完成，获取提取结果
        // 从任务结果中获取风格
        let taskResult = statusResponse.result;
        if (typeof taskResult === 'string') {
          taskResult = JSON.parse(taskResult);
        }
        extractedStyle.value = taskResult?.style || "未提取到风格";
        ElMessage.success("风格提取完成！");
        break;
      } else if (taskStatus === "failed") {
        throw new Error(statusResponse.error || "风格提取失败");
      }
    }

    if (taskStatus === "processing") {
      throw new Error("风格提取超时，请稍后重试");
    }
  } catch (error: any) {
    ElMessage.error(error.message || "风格提取失败");
  } finally {
    extractingStyle.value = false;
  }
};

const saveExtractedStyle = async () => {
  if (!extractedStyle.value.trim()) {
    ElMessage.warning("请输入风格信息");
    return;
  }

  try {
    savingStyle.value = true;
    // 调用 API 保存风格
    await dramaAPI.update(dramaId, { plot_style: extractedStyle.value.trim() });

    ElMessage.success("风格保存成功！");
    extractStyleDialogVisible.value = false;
    // 刷新数据
    await loadDramaData();
  } catch (error: any) {
    ElMessage.error(error.message || "保存失败");
  } finally {
    savingStyle.value = false;
  }
};
const currentEditItem = ref<any>({ name: "" });
const currentEditType = ref<"character" | "scene">("character");
const editPrompt = ref("");
const libraryItems = ref<any[]>([]);
const currentUploadTarget = ref<any>(null);

// 角色设计弹框
const characterDesignDialogVisible = ref(false);
const characterDesignTarget = ref<any>(null);
const characterDesignSaving = ref(false);
const characterDesignForm = ref({
  name: "",
  age: "",
  gender: "",
  appearance: "",
  description: ""
});
const ageOptions = [
  { value: "婴儿", label: "婴儿" },
  { value: "幼儿", label: "幼儿" },
  { value: "儿童", label: "儿童" },
  { value: "青少年", label: "青少年" },
  { value: "青年", label: "青年" },
  { value: "成年", label: "成年" },
  { value: "中年", label: "中年" },
  { value: "年长者", label: "年长者" },
  { value: "老年", label: "老年" }
];
const genderOptions = [
  { value: "男", label: "男" },
  { value: "女", label: "女" },
  { value: "其他", label: "其他" }
];

// 添加场景相关
const newScene = ref<any>({
  location: "",
  time: "",
  prompt: "",
  image_url: "",
  local_path: "",
});
const extractingScenes = ref(false);
const uploadAction = computed(() => "/api/v1/upload/image");
const uploadHeaders = computed(() => ({
  Authorization: `Bearer ${localStorage.getItem("token")}`,
}));

// AI模型配置
interface ModelOption {
  modelName: string;
  configName: string;
  configId: number;
  priority: number;
}

const textModels = ref<ModelOption[]>([]);
const imageModels = ref<ModelOption[]>([]);
const selectedTextModel = ref<string>("");
const selectedImageModel = ref<string>("");

const hasScript = computed(() => {
  const currentEp = currentEpisode.value;
  return (
    currentEp && currentEp.script_content && currentEp.script_content.length > 0
  );
});

const currentEpisode = computed(() => {
  if (!drama.value?.episodes) return null;
  return drama.value.episodes.find((ep) => ep.episode_number === episodeNumber);
});

// 章节剧本显示/编辑内容：编辑时用缓冲区，非编辑时用 currentEpisode.script_content，避免 readonly 切换导致内容不显示
const displayScriptContent = computed({
  get() {
    return isEditing.value
      ? editingScriptContent.value
      : (currentEpisode.value?.script_content ?? "");
  },
  set(val: string) {
    if (isEditing.value) {
      editingScriptContent.value = val;
    } else if (currentEpisode.value) {
      currentEpisode.value.script_content = val;
    }
  },
});

const hasCharacters = computed(() => {
  return (
    currentEpisode.value?.characters &&
    currentEpisode.value.characters.length > 0
  );
});

const charactersCount = computed(() => {
  return currentEpisode.value?.characters?.length || 0;
});

const hasExtractedData = computed(() => {
  const hasScenes =
    currentEpisode.value?.scenes && currentEpisode.value.scenes.length > 0;
  // 只要有角色或场景，就认为已经提取过数据
  return hasCharacters.value || hasScenes;
});

const allImagesGenerated = computed(() => {
  // 如果没有提取任何数据，允许跳过（可能是空章节或用户想直接进入拆解分镜）
  if (!hasExtractedData.value) return true;

  const characters = currentEpisode.value?.characters || [];
  const scenes = currentEpisode.value?.scenes || [];

  // 如果角色和场景都为空，允许跳过
  if (characters.length === 0 && scenes.length === 0) return true;

  // 检查所有有数据的项是否都已生成图片
  const allCharsHaveImages =
    characters.length === 0 || characters.every((char) => char.image_url);
  const allScenesHaveImages =
    scenes.length === 0 || scenes.every((scene) => scene.image_url);

  return allCharsHaveImages && allScenesHaveImages;
});

// 工作流步骤列表（用于头部步骤条，computed 以响应 i18n）
const workflowStepList = computed(() => [
  { name: '0', index: 0, label: $t('workflow.steps.content') },
  { name: '1', index: 1, label: $t('workflow.steps.generateImages') },
  { name: '2', index: 2, label: $t('workflow.steps.splitStoryboard') },
  { name: '3', index: 3, label: $t('workflow.steps.professional') },
]);

// 步骤条点击（与 tab 切换一致）
const handleStepClick = (stepName: string) => {
  currentStep.value = stepName;
  const key = getStepStorageKey();
  if (key) localStorage.setItem(key, stepName);
};

// 标签页切换逻辑（保留供兼容）
const handleTabClick = (tab: any) => {
  const targetStep = tab.paneName;
  console.log(`切换到步骤: ${targetStep}`);
};

// 检查阶段是否完成
const isStepCompleted = (step: number): boolean => {
  if (step === 0) {
    return hasScript.value; // 剧本已填写
  }

  if (step === 1) {
    return hasExtractedData.value; // 角色和场景已提取
  }

  if (step === 2) {
    return currentEpisode.value?.storyboards && currentEpisode.value.storyboards.length > 0; // 分镜已拆分
  }

  if (step === 3) {
    return currentEpisode.value?.storyboards && currentEpisode.value.storyboards.length > 0; // 与步骤2一致，进入过专业制作即视为完成
  }

  return false;
};

const goBack = () => {
  // 使用 replace 避免在历史记录中留下当前页面
  router.replace(`/dramas/${dramaId}`);
};

// 加载AI模型配置
const loadAIConfigs = async () => {
  try {
    const [textList, imageList] = await Promise.all([
      aiAPI.list("text"),
      aiAPI.list("image"),
    ]);

    // 只使用激活的配置
    const activeTextList = textList.filter((c) => c.is_active);
    const activeImageList = imageList.filter((c) => c.is_active);

    // 展开模型列表并去重（保留优先级最高的）
    const allTextModels = activeTextList
      .flatMap((config) => {
        const models = Array.isArray(config.model)
          ? config.model
          : [config.model];
        return models.map((modelName) => ({
          modelName,
          configName: config.name,
          configId: config.id,
          priority: config.priority || 0,
        }));
      })
      .sort((a, b) => b.priority - a.priority);

    // 按模型名称去重，保留优先级最高的（已排序，第一个就是优先级最高的）
    const textModelMap = new Map<string, ModelOption>();
    allTextModels.forEach((model) => {
      if (!textModelMap.has(model.modelName)) {
        textModelMap.set(model.modelName, model);
      }
    });
    textModels.value = Array.from(textModelMap.values());

    const allImageModels = activeImageList
      .flatMap((config) => {
        const models = Array.isArray(config.model)
          ? config.model
          : [config.model];
        return models.map((modelName) => ({
          modelName,
          configName: config.name,
          configId: config.id,
          priority: config.priority || 0,
        }));
      })
      .sort((a, b) => b.priority - a.priority);

    // 按模型名称去重，保留优先级最高的
    const imageModelMap = new Map<string, ModelOption>();
    allImageModels.forEach((model) => {
      if (!imageModelMap.has(model.modelName)) {
        imageModelMap.set(model.modelName, model);
      }
    });
    imageModels.value = Array.from(imageModelMap.values());

    // 设置默认选择（优先级最高的）
    if (textModels.value.length > 0 && !selectedTextModel.value) {
      selectedTextModel.value = textModels.value[0].modelName;
    }
    if (imageModels.value.length > 0 && !selectedImageModel.value) {
      // 优先选择包含 nano 的模型
      const nanoModel = imageModels.value.find((m) =>
        m.modelName.toLowerCase().includes("nano"),
      );
      selectedImageModel.value = nanoModel
        ? nanoModel.modelName
        : imageModels.value[0].modelName;
    }

    // 验证已选择的模型是否还在可用列表中，如果不在则重置为默认值
    const availableTextModelNames = textModels.value.map((m) => m.modelName);
    const availableImageModelNames = imageModels.value.map((m) => m.modelName);

    if (
      selectedTextModel.value &&
      !availableTextModelNames.includes(selectedTextModel.value)
    ) {
      console.warn(
        `已选择的文本模型 ${selectedTextModel.value} 不在可用列表中，重置为默认值`,
      );
      selectedTextModel.value =
        textModels.value.length > 0 ? textModels.value[0].modelName : "";
      // 更新 localStorage
      if (selectedTextModel.value) {
        localStorage.setItem(
          `ai_text_model_${dramaId}`,
          selectedTextModel.value,
        );
      }
    }

    if (
      selectedImageModel.value &&
      !availableImageModelNames.includes(selectedImageModel.value)
    ) {
      console.warn(
        `已选择的图片模型 ${selectedImageModel.value} 不在可用列表中，重置为默认值`,
      );
      // 优先选择包含 nano 的模型
      const nanoModel = imageModels.value.find((m) =>
        m.modelName.toLowerCase().includes("nano"),
      );
      selectedImageModel.value =
        imageModels.value.length > 0
          ? nanoModel
            ? nanoModel.modelName
            : imageModels.value[0].modelName
          : "";
      // 更新 localStorage
      if (selectedImageModel.value) {
        localStorage.setItem(
          `ai_image_model_${dramaId}`,
          selectedImageModel.value,
        );
      }
    }
  } catch (error: any) {
    console.error("加载AI配置失败:", error);
  }
};

// 显示模型配置对话框
const showModelConfigDialog = () => {
  modelConfigDialogVisible.value = true;
  loadAIConfigs();
};

// 保存模型配置
const saveModelConfig = () => {
  if (!selectedTextModel.value || !selectedImageModel.value) {
    ElMessage.warning($t("workflow.pleaseSelectModels"));
    return;
  }

  // 保存模型名称到localStorage
  localStorage.setItem(`ai_text_model_${dramaId}`, selectedTextModel.value);
  localStorage.setItem(`ai_image_model_${dramaId}`, selectedImageModel.value);

  ElMessage.success($t("workflow.modelConfigSaved"));
  modelConfigDialogVisible.value = false;
};

const nextStep = () => {
  const current = parseInt(currentStep.value);
  if (current < 3) {
    currentStep.value = (current + 1).toString();
  }
};

const prevStep = () => {
  const current = parseInt(currentStep.value);
  if (current > 0) {
    currentStep.value = (current - 1).toString();
  }
};

// 从localStorage加载已保存的模型配置
const loadSavedModelConfig = () => {
  const savedTextModel = localStorage.getItem(`ai_text_model_${dramaId}`);
  const savedImageModel = localStorage.getItem(`ai_image_model_${dramaId}`);

  if (savedTextModel) {
    selectedTextModel.value = savedTextModel;
  }
  if (savedImageModel) {
    selectedImageModel.value = savedImageModel;
  }
};

const loadDramaData = async () => {
  try {
    const data = await dramaAPI.get(dramaId);
    drama.value = data;

    if (!hasScript.value) {
      scriptContent.value = "";
      // 如果没有剧本内容，重置到第一步
      currentStep.value = 0;
    }

    // 检查是否有生成中的角色或场景，自动启动轮询
    await checkAndStartPolling();
  } catch (error: any) {
    ElMessage.error(error.message || "加载项目数据失败");
  }
};

// 检查并启动轮询
const checkAndStartPolling = async () => {
  if (!currentEpisode.value) return;

  // 检查角色的生成状态
  for (const char of currentEpisode.value.characters || []) {
    if (
      char.image_generation_status === "pending" ||
      char.image_generation_status === "processing"
    ) {
      // 查找对应的image_generation记录
      try {
        const imageGenList = await imageAPI.listImages({
          drama_id: dramaId,
          status: char.image_generation_status as any,
        });

        // 找到这个角色的image_generation记录
        const charImageGen = imageGenList.items.find(
          (img) =>
            img.character_id === char.id &&
            (img.status === "pending" || img.status === "processing"),
        );

        if (charImageGen) {
          // 启动轮询
          generatingCharacterImages.value[char.id] = true;
          pollImageStatus(charImageGen.id, async () => {
            await loadDramaData();
            ElMessage.success(`${char.name}的图片生成完成！`);
          }).finally(() => {
            generatingCharacterImages.value[char.id] = false;
          });
        }
      } catch (error) {
        console.error("[轮询] 查询角色图片生成记录失败:", error);
      }
    }
  }

  // 检查场景的生成状态
  for (const scene of currentEpisode.value.scenes || []) {
    if (
      scene.image_generation_status === "pending" ||
      scene.image_generation_status === "processing"
    ) {
      // 查找对应的image_generation记录
      try {
        const imageGenList = await imageAPI.listImages({
          drama_id: dramaId,
          status: scene.image_generation_status as any,
        });

        // 找到这个场景的image_generation记录
        const sceneImageGen = imageGenList.items.find(
          (img) =>
            img.scene_id === scene.id &&
            (img.status === "pending" || img.status === "processing"),
        );

        if (sceneImageGen) {
          // 启动轮询
          generatingSceneImages.value[scene.id] = true;
          pollImageStatus(sceneImageGen.id, async () => {
            await loadDramaData();
            ElMessage.success(`${scene.location}的图片生成完成！`);
          }).finally(() => {
            generatingSceneImages.value[scene.id] = false;
          });
        }
      } catch (error) {
        console.error("[轮询] 查询场景图片生成记录失败:", error);
      }
    }
  }
};

const saveChapterScript = async () => {
  try {
    const existingEpisodes = drama.value?.episodes || [];

    // 查找当前章节
    const episodeIndex = existingEpisodes.findIndex(
      (ep) => ep.episode_number === episodeNumber,
    );

    let updatedEpisodes;
    if (episodeIndex >= 0) {
      // 更新已有章节
      updatedEpisodes = [...existingEpisodes];
      updatedEpisodes[episodeIndex] = {
        ...updatedEpisodes[episodeIndex],
        script_content: scriptContent.value,
      };
    } else {
      // 创建新章节
      const newEpisode = {
        episode_number: episodeNumber,
        title: `第${episodeNumber}集`,
        script_content: scriptContent.value,
      };
      updatedEpisodes = [...existingEpisodes, newEpisode];
    }

    await dramaAPI.saveEpisodes(dramaId, updatedEpisodes);
    ElMessage.success("章节保存成功！");
    await loadDramaData();
  } catch (error: any) {
    ElMessage.error(error.message || "保存失败");
  }
};

const editCurrentEpisodeScript = () => {
  scriptContent.value = currentEpisode.value?.script_content || "";
};

// 编辑剧本相关函数
const startEdit = () => {
  const content = currentEpisode.value?.script_content ?? "";
  if (content || content === "") {
    originalScriptContent.value = content;
    editingScriptContent.value = content;
    isEditing.value = true;
  }
};

const cancelEdit = () => {
  if (currentEpisode.value && originalScriptContent.value !== undefined) {
    currentEpisode.value.script_content = originalScriptContent.value;
  }
  editingScriptContent.value = "";
  isEditing.value = false;
};

const saveEdit = async () => {
  const contentToSave = isEditing.value
    ? editingScriptContent.value
    : (currentEpisode.value?.script_content ?? "");
  if (!contentToSave.trim()) {
    ElMessage.warning("剧本内容不能为空");
    return;
  }

  saving.value = true;
  try {
    // 先将当前编辑的内容赋值给 scriptContent 变量，因为 saveChapterScript 使用这个变量
    scriptContent.value = contentToSave;
    await saveChapterScript();
    if (currentEpisode.value) {
      currentEpisode.value.script_content = contentToSave;
    }
    editingScriptContent.value = "";
    isEditing.value = false;
    ElMessage.success("剧本已保存");
  } catch (error: any) {
    ElMessage.error(error.message || "保存失败");
    // 恢复原始内容
    if (currentEpisode.value && originalScriptContent.value !== undefined) {
      currentEpisode.value.script_content = originalScriptContent.value;
    }
    editingScriptContent.value = originalScriptContent.value;
  } finally {
    saving.value = false;
  }
};

const handleExtractCharactersAndBackgrounds = async () => {
  // 如果已经提取过，显示确认对话框
  if (hasExtractedData.value) {
    try {
      await ElMessageBox.confirm(
        $t("workflow.reExtractConfirmMessage"),
        $t("workflow.reExtractConfirmTitle"),
        {
          confirmButtonText: $t("common.confirm"),
          cancelButtonText: $t("common.cancel"),
          type: "warning",
          distinguishCancelAndClose: true,
        },
      );
    } catch {
      ElMessage.info($t("workflow.extractCancelled"));
      return;
    }
  }

  // 显示即将开始的提示
  if (hasExtractedData.value) {
    ElMessage.info($t("workflow.startReExtracting"));
  }

  await extractCharactersAndBackgrounds();
};

// 轮询检查图片生成状态
const pollImageStatus = async (
  imageGenId: number,
  onComplete: () => Promise<void>,
) => {
  const maxAttempts = 100; // 最多轮询100次
  const pollInterval = 6000; // 每6秒轮询一次

  for (let i = 0; i < maxAttempts; i++) {
    try {
      await new Promise((resolve) => setTimeout(resolve, pollInterval));

      const imageGen = await imageAPI.getImage(imageGenId);

      if (imageGen.status === "completed") {
        // 生成成功
        await onComplete();
        return;
      } else if (imageGen.status === "failed") {
        // 生成失败
        ElMessage.error(`图片生成失败: ${imageGen.error_msg || "未知错误"}`);
        return;
      }
      // 如果是pending或processing，继续轮询
    } catch (error: any) {
      console.error("[轮询] 检查图片状态失败:", error);
      // 继续轮询，不中断
    }
  }

  // 超时
  ElMessage.warning("图片生成超时，请稍后刷新页面查看结果");
};

const extractCharactersAndBackgrounds = async () => {
  if (!currentEpisode.value?.id) {
    ElMessage.error("章节信息不存在");
    return;
  }

  extractingCharactersAndBackgrounds.value = true;

  try {
    const episodeId = currentEpisode.value.id;

    // 并行创建异步任务
    const [characterTask, backgroundTask] = await Promise.all([
      generationAPI.generateCharacters({
        drama_id: dramaId.toString(),
        episode_id: episodeId,
        outline: currentEpisode.value.script_content || "",
        count: 0,
        model: selectedTextModel.value, // 传递用户选择的文本模型
      }),
      dramaAPI.extractBackgrounds(
        episodeId.toString(),
        selectedTextModel.value,
      ), // 传递用户选择的文本模型
    ]);

    ElMessage.success("任务已创建，正在后台处理...");

    // 并行轮询两个任务
    await Promise.all([
      pollExtractTask(characterTask.task_id, "character"),
      pollExtractTask(backgroundTask.task_id, "background"),
    ]);

    ElMessage.success($t("workflow.charactersAndScenesExtractSuccess"));
    await loadDramaData();
  } catch (error: any) {
    console.error($t("workflow.charactersAndScenesExtractFailed") + ":", error);

    const errorData = error.response?.data?.error;
    const errorMsg = errorData?.message || error.message || "提取失败";

    if (
      errorMsg.includes("no config found") ||
      errorMsg.includes("AI client") ||
      errorMsg.includes("failed to get AI client")
    ) {
      ElMessage({
        type: "warning",
        message: '未配置AI服务，请前往"设置 > AI服务配置"添加文本生成服务',
        duration: 5000,
        showClose: true,
      });
    } else {
      ElMessage.error(errorMsg);
    }
  } finally {
    extractingCharactersAndBackgrounds.value = false;
  }
};

// 轮询提取任务状态
const pollExtractTask = async (
  taskId: string,
  type: "character" | "background",
) => {
  const maxAttempts = 60; // 最多轮询60次（2分钟）
  const interval = 2000; // 每2秒查询一次

  for (let i = 0; i < maxAttempts; i++) {
    await new Promise((resolve) => setTimeout(resolve, interval));

    try {
      const task = await generationAPI.getTaskStatus(taskId);

      if (task.status === "completed") {
        // 任务完成
        if (type === "character" && task.result) {
          // 解析角色数据并保存
          const result =
            typeof task.result === "string"
              ? JSON.parse(task.result)
              : task.result;
          if (result.characters && result.characters.length > 0) {
            await dramaAPI.saveCharacters(
              dramaId,
              result.characters,
              currentEpisode.value?.id,
            );
          }
        }
        return;
      } else if (task.status === "failed") {
        // 任务失败
        throw new Error(
          task.error ||
            (type === "character"
              ? $t("workflow.characterGenerationFailed")
              : $t("workflow.sceneExtractionFailed")),
        );
      }
      // 否则继续轮询
    } catch (error: any) {
      console.error(`轮询${type}任务状态失败:`, error);
      throw error;
    }
  }

  throw new Error(
    type === "character"
      ? $t("workflow.characterGenerationTimeout")
      : $t("workflow.sceneExtractionTimeout"),
  );
};

const generateCharacterImage = async (characterId: number) => {
  generatingCharacterImages.value[characterId] = true;

  try {
    // 获取用户选择的图片生成模型
    const model = selectedImageModel.value || undefined;
    const response = await characterLibraryAPI.generateCharacterImage(
      characterId.toString(),
      model,
    );
    const imageGenId = response.image_generation?.id;

    if (imageGenId) {
      ElMessage.info("角色图片生成中，请稍候...");
      // 轮询检查生成状态
      await pollImageStatus(imageGenId, async () => {
        await loadDramaData();
        ElMessage.success("角色图片生成完成！");
      });
    } else {
      ElMessage.success("角色图片生成已启动");
      await loadDramaData();
    }
  } catch (error: any) {
    ElMessage.error(error.message || "生成失败");
  } finally {
    generatingCharacterImages.value[characterId] = false;
  }
};

const toggleSelectAllCharacters = () => {
  if (selectAllCharacters.value) {
    selectedCharacterIds.value =
      currentEpisode.value?.characters?.map((char) => char.id) || [];
  } else {
    selectedCharacterIds.value = [];
  }
};

const toggleSelectAllScenes = () => {
  if (selectAllScenes.value) {
    selectedSceneIds.value =
      currentEpisode.value?.scenes?.map((scene) => scene.id) || [];
  } else {
    selectedSceneIds.value = [];
  }
};

const batchGenerateCharacterImages = async () => {
  if (selectedCharacterIds.value.length === 0) {
    ElMessage.warning("请先选择要生成的角色");
    return;
  }

  batchGeneratingCharacters.value = true;
  try {
    // 获取用户选择的图片生成模型
    const model = selectedImageModel.value || undefined;

    // 使用批量生成API
    await characterLibraryAPI.batchGenerateCharacterImages(
      selectedCharacterIds.value.map((id) => id.toString()),
      model,
    );

    ElMessage.success($t("workflow.batchTaskSubmitted"));
    await loadDramaData();
  } catch (error: any) {
    ElMessage.error(error.message || $t("workflow.batchGenerateFailed"));
  } finally {
    batchGeneratingCharacters.value = false;
  }
};

const generateSceneImage = async (sceneId: string) => {
  generatingSceneImages.value[sceneId] = true;

  try {
    // 获取用户选择的图片生成模型
    const model = selectedImageModel.value || undefined;
    const response = await dramaAPI.generateSceneImage({
      scene_id: parseInt(sceneId),
      model,
    });
    const imageGenId = response.image_generation?.id;

    if (imageGenId) {
      ElMessage.info($t("workflow.sceneImageGenerating"));
      // 轮询检查生成状态
      await pollImageStatus(imageGenId, async () => {
        await loadDramaData();
        ElMessage.success($t("workflow.sceneImageComplete"));
      });
    } else {
      ElMessage.success($t("workflow.sceneImageStarted"));
      await loadDramaData();
    }
  } catch (error: any) {
    ElMessage.error(error.message || "生成失败");
  } finally {
    generatingSceneImages.value[sceneId] = false;
  }
};

const batchGenerateSceneImages = async () => {
  if (selectedSceneIds.value.length === 0) {
    ElMessage.warning("请先选择要生成的场景");
    return;
  }

  batchGeneratingScenes.value = true;
  try {
    const promises = selectedSceneIds.value.map((sceneId) =>
      generateSceneImage(sceneId.toString()),
    );
    const results = await Promise.allSettled(promises);

    const successCount = results.filter((r) => r.status === "fulfilled").length;
    const failCount = results.filter((r) => r.status === "rejected").length;

    if (failCount === 0) {
      ElMessage.success(
        $t("workflow.batchCompleteSuccess", { count: successCount }),
      );
    } else {
      ElMessage.warning(
        $t("workflow.batchCompletePartial", {
          success: successCount,
          fail: failCount,
        }),
      );
    }
  } catch (error: any) {
    ElMessage.error(error.message || $t("workflow.batchGenerateFailed"));
  } finally {
    batchGeneratingScenes.value = false;
  }
};

const taskProgress = ref(0);
const taskMessage = ref("");
let pollTimer: any = null;

const generateShots = async () => {
  if (!currentEpisode.value?.id) {
    ElMessage.error("章节信息不存在");
    return;
  }

  generatingShots.value = true;
  taskProgress.value = 0;
  taskMessage.value = "初始化任务...";

  try {
    const episodeId = currentEpisode.value.id.toString();

    // 【调试日志】输出当前操作的集数信息
    console.log("=== 开始生成分镜 ===");
    console.log("当前 episodeNumber (路由参数):", episodeNumber);
    console.log("当前 episodeId (从 currentEpisode 获取):", episodeId);
    console.log("currentEpisode 完整信息:", {
      id: currentEpisode.value?.id,
      episode_number: currentEpisode.value?.episode_number,
      title: currentEpisode.value?.title,
    });
    console.log(
      "所有剧集列表:",
      drama.value?.episodes?.map((ep) => ({
        id: ep.id,
        episode_number: ep.episode_number,
        title: ep.title,
      })),
    );

    // 创建异步任务
    const response = await generationAPI.generateStoryboard(
      episodeId,
      selectedTextModel.value,
    );

    taskMessage.value = response.message || "任务已创建";

    // 开始轮询任务状态
    await pollTaskStatus(response.task_id);
  } catch (error: any) {
    ElMessage.error(error.message || "拆分失败");
    generatingShots.value = false;
  }
};

const pollTaskStatus = async (taskId: string) => {
  const checkStatus = async () => {
    try {
      const task = await generationAPI.getTaskStatus(taskId);

      taskProgress.value = task.progress;
      taskMessage.value = task.message || `处理中... ${task.progress}%`;

      if (task.status === "completed") {
        // 任务完成
        if (pollTimer) {
          clearInterval(pollTimer);
          pollTimer = null;
        }
        generatingShots.value = false;

        ElMessage.success($t("workflow.splitSuccess"));

        // 切换到专业制作 tab（不跳转页面）
        currentStep.value = "3";
        const key = getStepStorageKey();
        if (key) localStorage.setItem(key, "3");
      } else if (task.status === "failed") {
        // 任务失败
        if (pollTimer) {
          clearInterval(pollTimer);
          pollTimer = null;
        }
        generatingShots.value = false;
        ElMessage.error(task.error || "分镜拆分失败");
      }
      // 否则继续轮询
    } catch (error: any) {
      if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
      }
      generatingShots.value = false;
      ElMessage.error("查询任务状态失败: " + error.message);
    }
  };

  // 立即检查一次
  await checkStatus();

  // 每2秒轮询一次
  pollTimer = setInterval(checkStatus, 2000);
};

const regenerateShots = async () => {
  await ElMessageBox.confirm($t("workflow.reSplitConfirm"), $t("common.tip"), {
    type: "warning",
  });

  await generateShots();
};

const shotEditDialogVisible = ref(false);
const editingShot = ref<any>(null);
const editingShotIndex = ref<number>(-1);
const savingShot = ref(false);
const professionalEditorRef = ref<{ loadData?: () => Promise<void> } | null>(null);

const editShot = (shot: any, index: number) => {
  editingShot.value = { ...shot };
  // 确保characters字段是数组类型
  if (!Array.isArray(editingShot.value.characters)) {
    editingShot.value.characters = [];
  } else {
    // 将角色对象转换为角色ID数组
    editingShot.value.characters = editingShot.value.characters.map((char: any) => {
      return typeof char === 'object' ? char.id : char;
    });
  }
  editingShotIndex.value = index;
  shotEditDialogVisible.value = true;
};

// 解析镜头关联的场景（用于卡片列表展示）
const getShotScene = (shot: any) => {
  if (shot.scene && typeof shot.scene === "object") return shot.scene;
  if (shot.background && typeof shot.background === "object") return shot.background;
  if (!shot.scene_id || !currentEpisode.value?.scenes) return null;
  return currentEpisode.value.scenes.find(
    (s: any) => String(s.id) === String(shot.scene_id),
  ) || null;
};

// 解析镜头关联角色名称（与专业制作一致：支持对象数组或 ID 数组）
const getShotCharacterNames = (shot: any): string[] => {
  const chars = shot?.characters;
  if (!chars || !Array.isArray(chars) || chars.length === 0) return [];
  const first = chars[0];
  if (typeof first === "object" && first !== null && "name" in first) {
    return chars.map((c: any) => c.name || String(c.id || ""));
  }
  const episodeChars = currentEpisode.value?.characters || [];
  return chars
    .map((id: number) => episodeChars.find((c: any) => c.id === id || String(c.id) === String(id)))
    .filter(Boolean)
    .map((c: any) => c.name || "");
};

// 卡片内系统提示词失焦保存
const saveShotSystemPrompt = async (shot: any) => {
  if (!shot?.id) return;
  try {
    await dramaAPI.updateStoryboard(shot.id.toString(), {
      video_prompt: shot.video_prompt ?? "",
    });
    ElMessage.success("系统提示词已保存");
    if (currentStep.value === "3" && professionalEditorRef.value?.loadData) {
      await professionalEditorRef.value.loadData();
    }
  } catch (e: any) {
    ElMessage.error(e?.message || "保存失败");
  }
};

// 删除分镜
const deleteShot = async (shot: any, index: number) => {
  try {
    await ElMessageBox.confirm(
      "确定删除该镜头？删除后无法恢复。",
      "删除确认",
      { type: "warning" },
    );
    await dramaAPI.deleteStoryboard(Number(shot.id));
    await loadDramaData();
    ElMessage.success("已删除");
  } catch (e: any) {
    if (e !== "cancel") ElMessage.error(e?.message || "删除失败");
  }
};

// 绑定角色/场景：打开简易绑定弹窗（仅角色+场景，不再打开完整编辑页）
const bindingShotIndex = ref<number>(-1);
const bindDialogVisible = ref(false);
const bindForm = ref<{ characterIds: number[]; scene_id: number | null }>({
  characterIds: [],
  scene_id: null,
});

const openBindCharacter = (shot: any, index: number) => {
  bindingShotIndex.value = index;
  const chars = shot.characters || [];
  bindForm.value = {
    characterIds: Array.isArray(chars)
      ? chars.map((c: any) => (typeof c === "object" ? c.id : c)).filter(Boolean)
      : [],
    scene_id: shot.scene_id != null ? Number(shot.scene_id) : null,
  };
  bindDialogVisible.value = true;
};

const openBindScene = (shot: any, index: number) => {
  openBindCharacter(shot, index);
};

const saveBindDialog = async () => {
  if (bindingShotIndex.value < 0 || !currentEpisode.value?.storyboards) return;
  const shot = currentEpisode.value.storyboards[bindingShotIndex.value];
  if (!shot?.id) return;
  try {
    await dramaAPI.updateStoryboard(shot.id.toString(), {
      characters: bindForm.value.characterIds,
      scene_id: bindForm.value.scene_id ?? undefined,
    });
    await loadDramaData();
    bindDialogVisible.value = false;
    ElMessage.success("关联已保存");
    if (currentStep.value === "3" && professionalEditorRef.value?.loadData) {
      await professionalEditorRef.value.loadData();
    }
  } catch (e: any) {
    ElMessage.error(e?.message || "保存失败");
  }
};

const saveShotEdit = async () => {
  if (!editingShot.value) return;

  try {
    savingShot.value = true;

    // 调用API更新镜头
    await dramaAPI.updateStoryboard(
      editingShot.value.id.toString(),
      editingShot.value,
    );

    // 更新本地数据
    if (currentEpisode.value?.storyboards) {
      currentEpisode.value.storyboards[editingShotIndex.value] = {
        ...editingShot.value,
      };
    }

    // 若当前在专业制作页，刷新其分镜数据，使生成提示词展示与旁白与编辑镜头内容同步
    if (currentStep.value === "3" && professionalEditorRef.value?.loadData) {
      await professionalEditorRef.value.loadData();
    }

    ElMessage.success("镜头修改成功");
    shotEditDialogVisible.value = false;
  } catch (error: any) {
    ElMessage.error("保存失败: " + (error.message || "未知错误"));
  } finally {
    savingShot.value = false;
  }
};

// 对话框相关方法
const openCharacterDesignDialog = (char: any) => {
  console.log("Opening character design dialog for char:", char);
  characterDesignTarget.value = char;
  characterDesignForm.value = {
    name: char.name ?? "",
    age: char.age || "",
    gender: char.gender || "",
    appearance: char.appearance ?? "",
    description: char.description ?? ""
  };
  console.log("Set characterDesignForm.value:", characterDesignForm.value);
  characterDesignDialogVisible.value = true;
};
const saveCharacterDesign = async () => {
  if (!characterDesignTarget.value?.id) return;
  characterDesignSaving.value = true;
  try {
    await characterLibraryAPI.updateCharacter(characterDesignTarget.value.id, {
      name: characterDesignForm.value.name || undefined,
      age: characterDesignForm.value.age || undefined,
      gender: characterDesignForm.value.gender || undefined,
      appearance: characterDesignForm.value.appearance || undefined,
      description: characterDesignForm.value.description || undefined
    });
    ElMessage.success($t("common.updateSuccess") || "保存成功");
    await loadDramaData();
    characterDesignDialogVisible.value = false;
  } catch (e: any) {
    ElMessage.error(e?.message || "保存失败");
  } finally {
    characterDesignSaving.value = false;
  }
};
const regenerateCharacterImageFromDesign = () => {
  if (characterDesignTarget.value?.id) {
    generateCharacterImage(characterDesignTarget.value.id);
  }
};
const uploadCharacterImageFromDesign = () => {
  if (characterDesignTarget.value?.id) {
    uploadCharacterImage(characterDesignTarget.value.id);
    characterDesignDialogVisible.value = false;
  }
};

const openPromptDialog = (item: any, type: "character" | "scene") => {
  currentEditItem.value = item;
  currentEditItem.value.name = item.name || item.location;
  currentEditType.value = type;
  editPrompt.value = item.prompt || item.appearance || item.description || "";
  promptDialogVisible.value = true;
};

const savePrompt = async () => {
  try {
    if (currentEditType.value === "character") {
      await characterLibraryAPI.updateCharacter(currentEditItem.value.id, {
        appearance: editPrompt.value,
      });
      await generateCharacterImage(currentEditItem.value.id);
    } else {
      // 保存场景提示词和时间（合并到一个 API 调用）
      await dramaAPI.updateScene(currentEditItem.value.id.toString(), {
        prompt: editPrompt.value,
        time: currentEditItem.value.time || "",
      });

      ElMessage.success("保存成功");
      await loadDramaData();
    }
    promptDialogVisible.value = false;
  } catch (error: any) {
    ElMessage.error(error.message || "保存失败");
  }
};

const uploadCharacterImage = (characterId: number) => {
  currentUploadTarget.value = { id: characterId, type: "character" };
  uploadDialogVisible.value = true;
};

const uploadSceneImage = (sceneId: string) => {
  currentUploadTarget.value = { id: sceneId, type: "scene" };
  uploadDialogVisible.value = true;
};

const selectFromLibrary = async (characterId: number) => {
  try {
    const result = await characterLibraryAPI.list({ page_size: 50 });
    libraryItems.value = result.items || [];
    currentUploadTarget.value = characterId;
    libraryDialogVisible.value = true;
  } catch (error: any) {
    ElMessage.error(error.message || $t("workflow.loadLibraryFailed"));
  }
};

const addToCharacterLibrary = async (character: any) => {
  if (!character.image_url) {
    ElMessage.warning($t("workflow.generateImageFirst"));
    return;
  }

  try {
    await ElMessageBox.confirm(
      $t("workflow.addToLibraryConfirm", { name: character.name }),
      $t("workflow.addToLibrary"),
      {
        confirmButtonText: $t("common.confirm"),
        cancelButtonText: $t("common.cancel"),
        type: "info",
      },
    );

    await characterLibraryAPI.addCharacterToLibrary(character.id.toString());
    ElMessage.success($t("workflow.addedToLibrary"));
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(error.message || $t("workflow.addFailed"));
    }
  }
};

const selectLibraryItem = async (item: any) => {
  try {
    if (currentUploadTarget.value?.type === "character") {
      await characterLibraryAPI.applyFromLibrary(
        currentUploadTarget.value.id.toString(),
        item.id,
      );
      ElMessage.success("应用角色形象成功！");
      await loadDramaData();
      libraryDialogVisible.value = false;
    }
  } catch (error: any) {
    ElMessage.error(error.message || "应用失败");
  }
};

const handleUploadSuccess = async (response: any) => {
  try {
    const imageUrl = response.url || response.data?.url;
    const localPath = response.local_path || response.data?.local_path;

    if (!imageUrl && !localPath) {
      ElMessage.error("上传失败：未获取到图片地址");
      return;
    }

    if (currentUploadTarget.value?.type === "character") {
      await characterLibraryAPI.updateCharacter(
        currentUploadTarget.value.id.toString(),
        {
          image_url: imageUrl,
          local_path: localPath,
        },
      );
      ElMessage.success("上传成功！");
    } else if (currentUploadTarget.value?.type === "scene") {
      // 更新场景图片
      await dramaAPI.updateScene(currentUploadTarget.value.id.toString(), {
        image_url: imageUrl,
        local_path: localPath,
      });
      ElMessage.success($t("workflow.sceneImageUploadSuccess"));
    }

    await loadDramaData();
    uploadDialogVisible.value = false;
  } catch (error: any) {
    ElMessage.error(error.message || "上传失败");
  }
};

const handleUploadError = () => {
  ElMessage.error("上传失败，请重试");
};

const deleteCharacter = async (characterId: number) => {
  try {
    await ElMessageBox.confirm(
      $t("workflow.deleteCharacterConfirm"),
      $t("workflow.deleteConfirmTitle"),
      {
        type: "warning",
        confirmButtonText: $t("workflow.confirmButtonText"),
        cancelButtonText: $t("workflow.cancelButtonText"),
      },
    );

    await characterLibraryAPI.deleteCharacter(characterId);
    ElMessage.success("角色已删除");
    await loadDramaData();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(error.message || "删除失败");
    }
  }
};

const deleteScene = async (sceneId: number) => {
  try {
    await ElMessageBox.confirm(
      $t("workflow.deleteSceneConfirm"),
      $t("workflow.deleteConfirmTitle"),
      {
        type: "warning",
        confirmButtonText: $t("workflow.confirmButtonText"),
        cancelButtonText: $t("workflow.cancelButtonText"),
      },
    );

    await dramaAPI.deleteScene(sceneId);
    ElMessage.success("场景已删除");
    await loadDramaData();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(error.message || "删除失败");
    }
  }
};

const goToProfessionalUI = () => {
  if (!currentEpisode.value?.id) {
    ElMessage.error("章节信息不存在");
    return;
  }
  currentStep.value = "3";
  const key = getStepStorageKey();
  if (key) localStorage.setItem(key, "3");
};

const goToCompose = () => {
  if (!currentEpisode.value?.id) {
    ElMessage.error("章节信息不存在");
    return;
  }

  router.push({
    name: "SceneComposition",
    params: {
      id: dramaId,
      episodeId: currentEpisode.value.id,
    },
  });
};

// 打开添加场景对话框
const openAddSceneDialog = () => {
  newScene.value = {
    location: "",
    time: "",
    prompt: "",
    image_url: "",
    local_path: "",
  };
  addSceneDialogVisible.value = true;
};

// 保存场景
const saveScene = async () => {
  if (!newScene.value.location) {
    ElMessage.warning($t("workflow.pleaseEnterSceneName"));
    return;
  }

  if (!currentEpisode.value?.id) {
    ElMessage.error($t("workflow.chapterInfoNotExist"));
    return;
  }

  try {
    // 创建场景，关联到当前章节
    await dramaAPI.createScene({
      drama_id: parseInt(dramaId),
      episode_id: parseInt(currentEpisode.value.id),
      location: newScene.value.location,
      time: newScene.value.time || "",
      prompt: newScene.value.prompt,
      image_url: newScene.value.image_url,
      local_path: newScene.value.local_path,
    });

    ElMessage.success($t("workflow.sceneAddSuccess"));
    addSceneDialogVisible.value = false;

    // 重新加载数据以更新场景列表
    await loadDramaData();
  } catch (error: any) {
    ElMessage.error(error.message || $t("workflow.sceneAddFailed"));
  }
};

// 处理场景图片上传成功
const handleSceneImageSuccess = (response: any) => {
  console.log("场景图片上传响应:", response);

  // 处理不同的响应结构
  const imageUrl = response.url || response.data?.url;
  const localPath = response.local_path || response.data?.local_path;

  if (imageUrl) {
    newScene.value.image_url = imageUrl;
  }
  if (localPath) {
    newScene.value.local_path = localPath;
  }

  console.log("更新后的 newScene:", newScene.value);

  if (imageUrl || localPath) {
    ElMessage.success($t("workflow.imageUploadSuccess"));
  } else {
    ElMessage.warning($t("workflow.imageUploadSuccessNoUrl"));
  }
};

// 图片上传前的校验
const beforeAvatarUpload = (file: File) => {
  const isImage = file.type.startsWith("image/");
  const isLt10M = file.size / 1024 / 1024 < 10;

  if (!isImage) {
    ElMessage.error("只能上传图片文件!");
    return false;
  }
  if (!isLt10M) {
    ElMessage.error("图片大小不能超过 10MB!");
    return false;
  }
  return true;
};

// 打开从剧本提取场景对话框
const openExtractSceneDialog = () => {
  extractScenesDialogVisible.value = true;
};

// 从剧本提取场景
const handleExtractScenes = async () => {
  if (!currentEpisode.value?.id) {
    ElMessage.error($t("workflow.chapterInfoNotExist"));
    return;
  }

  try {
    extractingScenes.value = true;
    await dramaAPI.extractBackgrounds(currentEpisode.value.id.toString());

    ElMessage.success($t("workflow.sceneExtractSubmitted"));
    extractScenesDialogVisible.value = false;

    // 自动刷新几次
    let checkCount = 0;
    const maxChecks = 5;
    const checkInterval = setInterval(async () => {
      checkCount++;
      await loadDramaData();

      if (checkCount >= maxChecks) {
        clearInterval(checkInterval);
      }
    }, 3000);
  } catch (error: any) {
    ElMessage.error(error.message || $t("workflow.sceneExtractFailed"));
  } finally {
    extractingScenes.value = false;
  }
};

// 监听步骤变化，保存到 localStorage
watch(currentStep, (newStep) => {
  localStorage.setItem(getStepStorageKey(), newStep.toString());
});

onMounted(() => {
  loadDramaData();
  loadSavedModelConfig();
  loadAIConfigs();
});
</script>

<style scoped lang="scss">
/* ========================================
   Page Layout / 页面布局 - 紧凑边距
   ======================================== */
.page-container {
  min-height: 100vh;
  background: var(--bg-primary);
  // padding: var(--space-2) var(--space-3);
  transition: background var(--transition-normal);
}

@media (min-width: 768px) {
  .page-container {
    // padding: var(--space-3) var(--space-4);
  }
}

@media (min-width: 1024px) {
  .page-container {
    // padding: var(--space-4) var(--space-5);
  }
}

.content-wrapper {
  display: flex;
  flex-direction: column;
  margin: 0 auto;
  width: 100%;
  height: 100vh;
  overflow: hidden;

  /* 顶部 header 和底部操作栏不收缩，内容区填满剩余空间 */
  > :first-child {
    flex-shrink: 0;
  }
  > .actions-container {
    flex-shrink: 0;
  }
}

.content-container {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.actions-container {
  height: 70px;
  background: var(--bg-card);
  overflow: hidden;
}

/* Header styles matching PageHeader component */
.page-header {
  margin-bottom: var(--space-3);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border-primary);
}

.header-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  flex-shrink: 0;
}

.back-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 0.875rem;
  background: var(--bg-card);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  color: var(--text-secondary);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;

  &:hover {
    background: var(--bg-card-hover);
    color: var(--text-primary);
    border-color: var(--border-secondary);
  }
}

.nav-divider {
  width: 1px;
  height: 2rem;
  background: var(--border-primary);
}

.header-title {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.025em;
  line-height: 1.2;
  white-space: nowrap;
}

.header-center {
  flex: 1;
  display: flex;
  justify-content: center;
}

.header-right {
  flex-shrink: 0;
}

/* 左侧：返回箭头 + 剧名(粗体) + 第X集(灰) */
.episode-header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
  min-width: 0;
}

.back-btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 8px;
  color: var(--text-secondary);
  border-radius: var(--radius-md);
  transition: color var(--transition-fast), background var(--transition-fast);
}

.back-btn-icon:hover {
  color: var(--text-primary);
  background: var(--bg-hover);
}

.episode-header-title {
  font-size: 1.125rem;
  font-weight: 700;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.episode-header-subtitle {
  font-size: 0.9375rem;
  color: var(--text-muted);
  white-space: nowrap;
}

/* 中间：步骤条 - 小圆 + 对勾 + 文案（与参考图一致：约 18px 圆、浅灰底、完成为绿色对勾） */
.workflow-steps-strip {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  flex: 1;
  min-width: 0;
}

.workflow-step-item {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  padding: 4px 0;
  border-radius: var(--radius-md);
  transition: opacity var(--transition-fast);
}

.workflow-step-item:hover {
  opacity: 0.9;
}

.workflow-step-circle {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 1px solid rgba(255, 255, 255, 0.25);
  background: rgba(255, 255, 255, 0.08);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.workflow-step-item.completed .workflow-step-circle {
  background: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.2);
}

.workflow-step-item.completed .workflow-step-circle .el-icon {
  font-size: 10px;
  color: #52c41a;
}

.workflow-step-item.active .workflow-step-circle {
  background: rgba(82, 196, 26, 0.25);
  border-color: #52c41a;
}

.workflow-step-item.active .workflow-step-circle .el-icon {
  font-size: 10px;
  color: #52c41a;
}

.workflow-step-label {
  font-size: 0.8125rem;
  color: var(--text-muted);
  white-space: nowrap;
}

.workflow-step-item.active .workflow-step-label,
.workflow-step-item.completed .workflow-step-label {
  color: var(--text-primary);
}

.workflow-card {
  height: calc(100% - 24px);
  margin: 12px;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
  border: 1px solid var(--border-primary);

  :deep(.el-card__body) {
    padding: 0;
  }
}

.custom-steps {
  display: flex;
  align-items: center;
  gap: 12px;

  .step-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    border-radius: 20px;
    background: var(--bg-card-hover);
    transition: all 0.3s;

    &.active {
      background: var(--accent-light);

      .step-circle {
        background: var(--accent);
        color: var(--text-inverse);
      }
    }

    &.current {
      background: var(--accent);
      color: var(--text-inverse);

      .step-circle {
        background: var(--bg-card);
        color: var(--accent);
      }

      .step-text {
        color: var(--text-inverse);
      }
    }

    .step-circle {
      width: 28px;
      height: 28px;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      background: var(--border-secondary);
      color: var(--text-secondary);
      font-weight: 600;
      transition: all 0.3s;
    }

    .step-text {
      font-size: 14px;
      font-weight: 500;
      white-space: nowrap;
    }
  }

  .step-arrow {
    color: var(--border-secondary);
  }
}

.stage-card {
  margin: 12px;

  &.professional-embed {
    padding: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    min-height: calc(100vh - 140px);
  }

  &.professional-embed :deep(.professional-editor) {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
  }

  &.professional-embed :deep(.professional-editor .editor-main) {
    flex: 1;
    min-height: 0;
  }

  &.stage-card-fullscreen {
    .stage-body-fullscreen {
      min-height: calc(100vh - 200px);
    }
  }
}

.stage-header {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .header-left {
    display: flex;
    align-items: center;
    gap: 16px;

    .header-info {
      h2 {
        margin: 0 0 4px 0;
        font-size: 20px;
      }

      p {
        margin: 0;
        color: var(--text-muted);
        font-size: 14px;
      }
    }
  }
}

.stage-body {
  background: var(--bg-card);
}

/* 分镜列表卡片样式（系统提示词、关联角色、关联场景） */
.shots-list-cards {
  .shots-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 16px;

    h3 {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
      color: var(--text-primary);
    }

    .shots-summary {
      font-size: 13px;
      color: var(--text-secondary);
    }
  }

  .shot-card-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .shot-card {
    display: flex;
    gap: 12px;
    padding: 14px 16px;
    background: var(--bg-secondary);
    border: 1px solid var(--border-primary);
    border-radius: 8px;
    align-items: flex-start;
  }

  .shot-card-left {
    flex-shrink: 0;
  }

  .shot-number-circle {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: var(--accent);
    color: var(--text-inverse);
    font-size: 14px;
    font-weight: 600;
  }

  .shot-card-main {
    flex: 1;
    min-width: 0;
  }

  .shot-card-title-row {
    margin-bottom: 8px;
  }

  .shot-title-text {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .shot-card-prompt-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 6px;
  }

  .shot-label {
    font-size: 12px;
    color: var(--text-secondary);
    font-weight: 500;
  }

  .shot-char-count {
    font-size: 12px;
    color: var(--text-muted);
  }

  .shot-system-prompt-input {
    margin-bottom: 12px;
  }

  .shot-system-prompt-input :deep(.el-textarea__inner) {
    background: var(--bg-card);
    border-color: var(--border-primary);
    color: var(--text-primary);
    font-size: 13px;
    line-height: 1.5;
  }

  .shot-meta {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .shot-meta-row {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;

    .shot-label {
      flex-shrink: 0;
    }
  }

  .shot-tag {
    &.shot-tag-character {
      background: rgba(64, 158, 255, 0.12);
      border-color: rgba(64, 158, 255, 0.3);
      color: var(--accent);
    }

    &.shot-tag-scene {
      background: rgba(230, 162, 60, 0.12);
      border-color: rgba(230, 162, 60, 0.35);
      color: #e6a23c;
    }
  }

  .shot-card-actions {
    flex-shrink: 0;
  }
}

.action-buttons {
  display: flex;
  gap: 12px;
  margin: 12px 0;
  flex-wrap: wrap;
  justify-content: center;
  align-items: center;
}

.action-buttons-inline {
  display: flex;
  gap: 12px;
}

.script-textarea {
  margin: 16px 0;

  &.script-textarea-fullscreen {
    :deep(textarea) {
      min-height: 500px;
      font-size: 14px;
      line-height: 1.8;
    }
  }
}

.overview-section {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.episode-editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.episode-editor-title {
  display: flex;
  align-items: center;
  gap: 8px;

  h3 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .save-status-text {
    color: var(--text-muted);
    font-size: 14px;
    font-weight: 500;
  }
}

.episode-editor-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.editor-action-btn {
  height: 34px;
  padding: 0 14px;
  border-radius: 10px;
  border: 1px solid #2a3750;
  background: #111d31;
  color: #d8e3f2;

  &:hover {
    background: #17253c;
    border-color: #3a4a67;
    color: #f3f8ff;
  }

  &.is-disabled {
    opacity: 0.55;
    background: #0e1727;
    border-color: #26344b;
    color: #8a96ab;
  }

  :deep(.el-icon) {
    margin-right: 4px;
  }
}

.extract-action-btn {
  background: #16233a;
}

.next-step-btn {
  border-color: #4ea6e5;
  background: linear-gradient(180deg, #92dbff 0%, #72c9f5 100%);
  color: #0f2435;
  font-weight: 600;

  &:hover {
    background: linear-gradient(180deg, #a4e4ff 0%, #82d0f8 100%);
    border-color: #67b7ef;
    color: #0a1d2b;
  }
}

.script-content-panel {
  background: #1f242e;
  border: 1px solid #2f3542;
  border-radius: 10px;
  padding: 16px;
}

.script-readonly-input {
  :deep(.el-textarea__inner) {
    min-height: 280px;
    resize: none;
    border: 1px solid #404757;
    box-shadow: none;
    background: #2a303b;
    color: #a6aebd;
    font-size: 14px;
    line-height: 1.9;
    cursor: not-allowed;
    -webkit-text-fill-color: #a6aebd;
  }

  :deep(.el-textarea.is-disabled .el-textarea__inner) {
    opacity: 1;
  }
}

.script-editor-input {
  :deep(.el-textarea__inner) {
    min-height: 280px;
    border: 1px solid #38517c;
    box-shadow: none;
    resize: vertical;
    font-size: 14px;
    line-height: 1.9;
    background: #131a26;
    color: #dbe7f5;
  }
}

.image-gen-section {
  margin-bottom: 32px;

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
    padding: 16px;
    background: var(--bg-secondary);
    // border-radius: 8px;
    // border: 1px solid var(--border-primary);

    .section-title {
      display: flex;
      align-items: center;
      gap: 16px;

      h3 {
        display: flex;
        align-items: center;
        gap: 8px;
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: var(--text-primary);

        .el-icon {
          color: var(--accent);
          font-size: 18px;
        }
      }

      .el-alert {
        border-radius: 4px;
      }
    }

    .section-actions {
      display: flex;
      align-items: center;
    }
  }
}

.empty-shots {
  padding: 60px 0;
  text-align: center;
}

.extracted-title {
  margin-bottom: 8px;
  color: var(--text-secondary);
}

.secondary-text {
  color: var(--text-muted);
  margin-left: 4px;
}

.task-message {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-muted);
  text-align: center;
}

.model-tip {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-muted);
}

.fixed-card {
  height: 100%;
  display: flex;
  flex-direction: column;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--border-primary);
  box-shadow: var(--shadow-card);
  transition: all 0.2s;

  &:hover {
    box-shadow: var(--shadow-card-hover);
  }

  :deep(.el-card__body) {
    flex: 1;
    padding: 0;
    display: flex;
    flex-direction: column;
  }

  .card-header {
    padding: 14px;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border-primary);
    display: flex;
    justify-content: space-between;
    align-items: center;

    .header-left {
      flex: 1;
      min-width: 0;

      h4 {
        margin: 0 0 4px 0;
        font-size: 14px;
        font-weight: 600;
        color: var(--text-primary);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .el-tag {
        margin-top: 0;
      }
    }
  }

  .card-image-container {
    flex: 1;
    width: 100%;
    min-height: 200px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-secondary);

    .char-image,
    .scene-image {
      width: 100%;
      height: 100%;
      position: relative;
      z-index: 1;

      .el-image {
        width: 100%;
        height: 100%;
        border-radius: 0;
      }
    }

    .char-placeholder,
    .scene-placeholder {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      color: var(--text-muted);
      padding: 20px;

      &.generating {
        color: var(--warning);
        background: var(--warning-light);

        .rotating {
          animation: rotating 2s linear infinite;
        }
      }

      &.failed {
        color: var(--error);
        background: var(--error-light);
      }
      position: relative;
      z-index: 1;

      .el-icon {
        opacity: 0.5;
      }

      span {
        margin-top: 10px;
        font-size: 12px;
      }
    }
  }

  .card-actions {
    padding: 10px;
    background: var(--bg-card);
    border-top: 1px solid var(--border-primary);
    display: flex;
    justify-content: center;
    gap: 8px;

    .el-button {
      margin: 0;
    }
  }
}

.character-image-list,
.scene-image-list {
  padding: 5px;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
  margin-top: 16px;

  .character-item,
  .scene-item {
    min-height: 360px;
  }
}

// 角色设计对话框
.character-design-dialog {
  .character-design-layout {
    display: flex;
    gap: 24px;
    min-height: 360px;
  }
  .character-design-left {
    flex: 0 0 320px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .single-composite-view {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .view-label {
    font-size: 12px;
    color: var(--text-secondary);
  }
  .view-image-wrap {
    width: 100%;
    aspect-ratio: 1/1;
    border-radius: 8px;
    overflow: hidden;
    background: var(--bg-secondary);
    border: 1px solid var(--border-primary);
  }
  .view-image-wrap.composite {
    min-height: 260px;
  }
  .view-image {
    width: 100%;
    height: 100%;
  }
  .view-placeholder {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-muted);
  }
  .character-design-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .character-design-right {
    flex: 1;
    min-width: 0;
  }
}

// 角色库选择对话框
.library-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  max-height: 500px;
  overflow-y: auto;
  padding: 8px;

  .library-item {
    cursor: pointer;
    border: 2px solid transparent;
    border-radius: 8px;
    overflow: hidden;
    transition: all 0.3s;

    &:hover {
      border-color: var(--accent);
      transform: translateY(-2px);
      box-shadow: var(--shadow-lg);
    }

    .el-image {
      width: 100%;
      height: 150px;
    }

    .library-item-name {
      padding: 8px;
      text-align: center;
      font-size: 12px;
      background: var(--bg-secondary);
      color: var(--text-primary);
    }
  }
}

.empty-library {
  padding: 40px 0;
}

// 上传区域
.upload-area {
  :deep(.el-upload-dragger) {
    width: 100%;
    height: 200px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
  }
}

// 旋转动画
@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* ========================================
   Dark Mode / 深色模式
   ======================================== */
:deep(.el-card) {
  background: var(--bg-card);
  border-color: var(--border-primary);
}

:deep(.el-card__header) {
  background: var(--bg-secondary);
  border-color: var(--border-primary);
}

:deep(.el-table) {
  --el-table-bg-color: var(--bg-card);
  --el-table-header-bg-color: var(--bg-secondary);
  --el-table-tr-bg-color: var(--bg-card);
  --el-table-row-hover-bg-color: var(--bg-card-hover);
  --el-table-border-color: var(--border-primary);
  --el-table-text-color: var(--text-primary);
  background: var(--bg-card);
}

:deep(.el-table th.el-table__cell),
:deep(.el-table td.el-table__cell) {
  background: var(--bg-card);
  border-color: var(--border-primary);
}

:deep(
  .el-table--striped .el-table__body tr.el-table__row--striped td.el-table__cell
) {
  background: var(--bg-secondary);
}

:deep(.el-table__header-wrapper th) {
  background: var(--bg-secondary) !important;
  color: var(--text-secondary);
}

:deep(.el-dialog) {
  background: var(--bg-card);
}

:deep(.el-dialog__header) {
  background: var(--bg-card);
}

:deep(.el-form-item__label) {
  color: var(--text-primary);
}

:deep(.el-input__wrapper) {
  background: var(--bg-secondary);
  box-shadow: 0 0 0 1px var(--border-primary) inset;
}

:deep(.el-input__inner) {
  color: var(--text-primary);
}

:deep(.el-textarea__inner) {
  background: var(--bg-secondary);
  color: var(--text-primary);
  box-shadow: 0 0 0 1px var(--border-primary) inset;
}

:deep(.el-select-dropdown) {
  background: var(--bg-elevated);
  border-color: var(--border-primary);
}

:deep(.el-upload-dragger) {
  background: var(--bg-secondary);
  border-color: var(--border-primary);
}

/* ========================================
   Tabs / 标签页样式
   ======================================== */
:deep(.el-tabs) {
  margin-bottom: 16px;
}

:deep(.el-tabs__header) {
  margin-bottom: 16px;
}

:deep(.el-tabs__nav) {
  border-bottom: 1px solid var(--border-primary);
}

:deep(.el-tabs__item) {
  color: var(--text-secondary);
  font-weight: 500;
  margin-right: 24px;
  padding: 0 8px;
}

:deep(.el-tabs__item.is-active) {
  color: var(--primary-color);
  font-weight: 600;
}

:deep(.el-tabs__item:hover) {
  color: var(--text-primary);
}

/* 标签页完成状态样式 */
:deep(.el-tabs__item.step-completed) {
  color: var(--success-color);
  font-weight: 600;

  &::after {
    content: "✓";
    margin-left: 4px;
    font-size: 14px;
  }
}

/* 标签页下划线 */
:deep(.el-tabs__active-bar) {
  background: var(--primary-color);
  height: 2px;
}

/* ========================================
   Empty Stage / 空阶段样式
   ======================================== */
.empty-stage {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  min-height: 300px;

  :deep(.el-empty) {
    margin-bottom: 24px;
  }

  :deep(.el-empty__description) {
    color: var(--text-secondary);
    font-size: 14px;
    margin-bottom: 16px;
  }
}
</style>
