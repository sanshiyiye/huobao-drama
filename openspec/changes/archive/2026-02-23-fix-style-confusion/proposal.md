# Change: 修复剧情风格和艺术风格混淆使用的问题

## Why
当前系统中，剧情风格和艺术风格被混淆使用在同一个 `Drama.Style` 字段中。用户在项目创建时设定的艺术风格（如 ghibli、guoman）会被章节内容提取的剧情风格（如科幻、古装）覆盖，导致图片生成时丢失原有的艺术风格提示词。

## What Changes
- **数据模型**：在 `Drama` 模型中新增 `PlotStyle` 字段存储剧情风格，保留 `Style` 字段作为艺术风格
- **业务逻辑**：修改风格提取服务，将提取的剧情风格保存到 `PlotStyle` 字段
- **提示词构建**：图片生成时同时使用艺术风格和剧情风格构建提示词
- **前端显示**：在编辑项目界面和章节内容界面分别显示艺术风格和剧情风格

## Impact
- Affected specs: drama
- Affected code:
  - `domain/models/drama.go` - 数据模型修改
  - `application/services/character_library_service.go` - 风格提取逻辑修改
  - `application/services/image_generation_service.go` - 提示词构建逻辑修改
  - `web/src/views/drama/DramaManagement.vue` - 项目编辑界面修改
  - `web/src/views/drama/EpisodeWorkflow.vue` - 章节内容界面修改
