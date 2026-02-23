# Tasks: 修复剧情风格和艺术风格混淆使用的问题

## 1. 数据模型修改
- [ ] 1.1 在 `domain/models/drama.go` 中新增 `PlotStyle` 字段
- [ ] 1.2 创建数据库迁移文件，为 `dramas` 表添加 `plot_style` 字段

## 2. 业务逻辑修改
- [ ] 2.1 修改 `application/services/character_library_service.go` 中的风格提取逻辑，将提取的剧情风格保存到 `PlotStyle` 字段
- [ ] 2.2 修改 `application/services/image_generation_service.go` 中的提示词构建逻辑，同时使用艺术风格和剧情风格

## 3. API 修改
- [ ] 3.1 修改 `api/handlers/drama_handler.go`，添加对 `PlotStyle` 字段的支持
- [ ] 3.2 更新 `application/services/drama_service.go`，添加 `PlotStyle` 字段的 getter/setter 方法

## 4. 前端修改
- [ ] 4.1 修改 `web/src/views/drama/DramaManagement.vue`，在项目编辑界面显示和编辑艺术风格
- [ ] 4.2 修改 `web/src/views/drama/EpisodeWorkflow.vue`，在章节内容界面显示剧情风格
- [ ] 4.3 更新 `web/src/types/drama.ts`，添加 `PlotStyle` 字段类型定义

## 5. 测试验证
- [ ] 5.1 测试项目创建和编辑功能
- [ ] 5.2 测试风格提取功能
- [ ] 5.3 测试图片生成功能
- [ ] 5.4 验证剧情风格和艺术风格是否正确分离
