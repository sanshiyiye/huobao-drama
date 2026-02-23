# Design: 修复剧情风格和艺术风格混淆使用的问题

## Context
当前系统中，剧情风格和艺术风格被混淆使用在同一个 `Drama.Style` 字段中。这种设计导致了以下问题：

1. 用户在项目创建时设定的艺术风格（如 ghibli、guoman）会被章节内容提取的剧情风格（如科幻、古装）覆盖
2. 图片生成时只使用单一风格，导致生成的图片不符合预期
3. 用户无法同时指定艺术风格和剧情风格，限制了创作自由度

## Goals / Non-Goals

### Goals
- 明确区分剧情风格和艺术风格的概念和存储方式
- 确保提取剧情风格后不会覆盖原有的艺术风格
- 图片生成时同时使用两种风格构建提示词
- 提供清晰的用户界面，让用户能够理解和操作两种风格

### Non-Goals
- 不支持章节级别的剧情风格设定（当前系统架构将风格视为整个剧本的属性）
- 不修改现有的艺术风格选择列表
- 不添加新的剧情风格提取算法

## Decisions

### 1. 数据模型设计
**决定**：在 `Drama` 模型中新增 `PlotStyle` 字段存储剧情风格，保留 `Style` 字段作为艺术风格

**理由**：
- 剧情风格和艺术风格都是剧本级别的属性
- 保留现有字段名 `Style` 用于艺术风格，保持向后兼容
- 新增 `PlotStyle` 字段专门用于剧情风格，语义明确

**数据模型变更**：
```go
type Drama struct {
    ID         uint   `gorm:"primaryKey;autoIncrement" json:"id"`
    Title      string `gorm:"type:varchar(200);not null" json:"title"`
    Style      string `gorm:"type:varchar(50);default:'realistic'" json:"style"` // 艺术风格
    PlotStyle  string `gorm:"type:varchar(50)" json:"plot_style"` // 剧情风格
    // 其他字段...
}
```

### 2. 业务逻辑设计
**决定**：修改风格提取服务，将提取的剧情风格保存到 `PlotStyle` 字段

**理由**：
- 章节内容提取的风格是剧情风格，应该存储到专门的字段中
- 避免与项目创建时设定的艺术风格混淆

**业务逻辑变更**：
```go
// 风格提取时
drama.PlotStyle = style // 保存到剧情风格字段
// 而不是 drama.Style = style
```

### 3. 提示词构建设计
**决定**：图片生成时同时使用艺术风格和剧情风格构建提示词

**理由**：
- 艺术风格决定图片的视觉表现（如 ghibli 风格）
- 剧情风格决定图片的内容主题（如科幻场景）
- 两者结合可以生成更符合用户预期的图片

**提示词构建逻辑**：
```go
// 同时添加艺术风格和剧情风格提示词
prompt := ""
if drama.Style != "" && drama.Style != "realistic" {
    prompt += getArtStylePrompt(drama.Style) + "\n"
}
if drama.PlotStyle != "" {
    prompt += getPlotStylePrompt(drama.PlotStyle) + "\n"
}
prompt += userPrompt // 用户输入的具体内容
```

## Risks / Trade-offs

### 风险 1：数据迁移
- **风险**：现有数据中 `Style` 字段可能已经包含剧情风格
- **缓解**：在数据迁移时，可以将现有 `Style` 字段的值复制到 `PlotStyle` 字段，然后将 `Style` 字段重置为默认值（如 realistic）

### 风险 2：API 兼容性
- **风险**：修改 API 可能会影响现有客户端
- **缓解**：保持现有 API 字段不变，新增字段支持向后兼容

## Migration Plan

1. 创建数据库迁移文件，为 `dramas` 表添加 `plot_style` 字段
2. 编写数据迁移脚本，处理现有数据
3. 更新相关业务逻辑和 API
4. 测试验证功能

## Open Questions

- 是否需要在前端界面提供剧情风格的手动编辑功能？
- 剧情风格提取失败时，是否需要提供默认值？
