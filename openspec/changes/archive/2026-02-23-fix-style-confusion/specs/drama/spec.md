## ADDED Requirements

### Requirement: 剧情风格存储
系统 SHALL 在 `Drama` 模型中新增 `PlotStyle` 字段，专门用于存储剧情风格（如科幻、古装、现代等）。

#### Scenario: 剧情风格提取成功
- **WHEN** 用户从章节内容中提取剧情风格
- **THEN** 系统将提取的剧情风格保存到 `PlotStyle` 字段
- **AND** 原有的艺术风格保持不变

#### Scenario: 剧情风格提取失败
- **WHEN** 剧情风格提取失败
- **THEN** `PlotStyle` 字段保持为空
- **AND** 系统显示提取失败的错误信息

### Requirement: 艺术风格保留
系统 SHALL 保留 `Drama.Style` 字段作为艺术风格（如 ghibli、guoman、realistic 等）。

#### Scenario: 项目创建时设定艺术风格
- **WHEN** 用户创建新剧本项目
- **THEN** 系统允许用户选择艺术风格
- **AND** 选择的艺术风格存储到 `Style` 字段

#### Scenario: 项目编辑时修改艺术风格
- **WHEN** 用户在项目编辑界面修改艺术风格
- **THEN** 系统更新 `Style` 字段
- **AND** 剧情风格保持不变

### Requirement: 双风格提示词构建
系统 SHALL 在图片生成时同时使用艺术风格和剧情风格构建提示词。

#### Scenario: 同时有艺术风格和剧情风格
- **WHEN** 剧本既有艺术风格又有剧情风格
- **THEN** 图片生成提示词包含艺术风格提示词和剧情风格提示词
- **AND** 两种风格提示词以合适的方式组合

#### Scenario: 只有艺术风格
- **WHEN** 剧本只有艺术风格，没有剧情风格
- **THEN** 图片生成提示词只包含艺术风格提示词

#### Scenario: 只有剧情风格
- **WHEN** 剧本只有剧情风格，没有艺术风格
- **THEN** 图片生成提示词只包含剧情风格提示词
- **AND** 使用默认的艺术风格（realistic）

### Requirement: 风格信息显示
系统 SHALL 在用户界面中清晰地区分和显示艺术风格和剧情风格。

#### Scenario: 项目编辑界面显示风格信息
- **WHEN** 用户查看项目编辑界面
- **THEN** 界面显示艺术风格的选择和编辑控件
- **AND** 界面显示剧情风格的显示和编辑控件

#### Scenario: 章节内容界面显示风格信息
- **WHEN** 用户查看章节内容界面
- **THEN** 界面显示剧情风格的提取和显示控件
- **AND** 界面显示艺术风格的只读信息

## MODIFIED Requirements
（无现有需求需要修改）

## REMOVED Requirements
（无需求需要删除）
