# 角色提取提示词改进规格说明

## MODIFIED Requirements

### Requirement: 支持剧情背景上下文传入

The `GetCharacterExtractionPromptTest` method MUST support passing a context parameter to enhance plot background matching for character/prop generation. The style prefix SHALL include the plot background context to influence the overall visual style.

#### Scenario: 传入中国古代剧情背景
```go
// 调用方式
prompt := p.GetCharacterExtractionPromptTest("ancient Chinese style", "Chinese ancient imperial court, traditional Han clothing")

// 生成的风格前缀
"ancient Chinese style, Chinese ancient imperial court, traditional Han clothing, full body portrait, white background."
```

#### Scenario: 传入现代都市背景
```go
// 调用方式
prompt := p.GetCharacterExtractionPromptTest("modern urban style", "modern city office, business casual clothing")

// 生成的风格前缀
"modern urban style, modern city office, business casual clothing, full body portrait, white background."
```

### Requirement: 动态剧情背景分析（待实现）

The `GetCharacterExtractionPromptTest` method SHALL internally analyze script content to automatically extract plot background information. This includes era, culture, and setting analysis.

#### Scenario: 自动识别中国古代剧情
```go
// 传入剧本内容
scriptContent := "在古代中国的皇宫中，皇帝正在批阅奏章..."

// 方法内部分析后生成的风格前缀
"ancient Chinese style, Chinese imperial court, traditional Han dynasty clothing, full body portrait, white background."
```

#### Scenario: 自动识别西方中世纪剧情
```go
// 传入剧本内容
scriptContent := "在中世纪的欧洲城堡中，骑士正在准备出征..."

// 方法内部分析后生成的风格前缀
"medieval European style, European castle, knight armor, full body portrait, white background."
```

## ADDED Requirements

### Requirement: 剧情背景分析接口

The system SHALL provide a script content analysis interface for extracting plot background information. This interface MUST support era, culture, and setting analysis.

#### Scenario: 分析剧本获取背景信息
```go
// 定义分析接口
type ScriptAnalyzer interface {
    AnalyzeEra(script string) string        // 分析时代背景
    AnalyzeCulture(script string) string    // 分析文化背景
    AnalyzeSetting(script string) string    // 分析场景类型
    GenerateContext(script string) string   // 生成完整上下文
}

// 使用接口
analyzer := NewScriptAnalyzer()
context := analyzer.GenerateContext(scriptContent)
```

### Requirement: 默认上下文处理

The system SHALL provide reasonable default values when no context is passed or analysis fails. The default context MUST be generic and applicable to most scenarios.

#### Scenario: 未传入上下文时的默认处理
```go
// 未传入上下文
prompt := p.GetCharacterExtractionPromptTest("realistic style", "")

// 生成的风格前缀（使用默认值）
"realistic style, generic modern style, full body portrait, white background."
```

#### Scenario: 分析失败时的处理
```go
// 分析失败（如无法识别剧情背景）
prompt := p.GetCharacterExtractionPromptTest("fantasy style", "")

// 生成的风格前缀（使用通用默认值）
"fantasy style, generic fantasy style, full body portrait, white background."
```
