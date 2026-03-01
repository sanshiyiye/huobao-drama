package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TruncateBase64InJSON 截断 JSON 中的 base64 图片数据，避免日志过大
// 支持多种格式：
// 1. 顶层字段：image_url, input_reference, Image (数组)
// 2. 嵌套字段：content[].image_url.url
// 3. 响应字段：b64_json, data (数组中的 base64)
// 4. data URI 格式：data:image/...;base64,xxxxx
func TruncateBase64InJSON(raw []byte) string {
	const truncateLen = 50 // 保留前50个字符，足够识别格式

	var body interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		// 如果不是 JSON，直接截断字符串
		return truncateString(string(raw), 500)
	}

	// 递归处理 JSON 结构
	processed := truncateBase64InValue(body, truncateLen)

	// 重新序列化为 JSON
	out, err := json.Marshal(processed)
	if err != nil {
		return truncateString(string(raw), 500)
	}
	return string(out)
}

// truncateBase64InValue 递归处理 JSON 值，截断 base64 数据
func truncateBase64InValue(v interface{}, truncateLen int) interface{} {
	switch val := v.(type) {
	case string:
		// 检查是否是 base64 数据 URI
		if isBase64DataURI(val) {
			if len(val) > truncateLen {
				return val[:truncateLen] + fmt.Sprintf("...[base64 truncated, len=%d]", len(val))
			}
		}
		// 检查是否是纯 base64 字符串（长度超过阈值且看起来像 base64）
		if len(val) > 100 && looksLikeBase64(val) {
			return val[:truncateLen] + fmt.Sprintf("...[base64 truncated, len=%d]", len(val))
		}
		return val

	case []interface{}:
		// 处理数组
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = truncateBase64InValue(item, truncateLen)
		}
		return result

	case map[string]interface{}:
		// 处理对象
		result := make(map[string]interface{})
		for key, value := range val {
			// 特殊处理已知的 base64 字段
			if isBase64Field(key) {
				result[key] = truncateBase64String(value, truncateLen)
			} else {
				result[key] = truncateBase64InValue(value, truncateLen)
			}
		}
		return result

	default:
		return v
	}
}

// isBase64DataURI 检查字符串是否是 data URI 格式的 base64
func isBase64DataURI(s string) bool {
	return strings.HasPrefix(s, "data:image/") && strings.Contains(s, ";base64,")
}

// looksLikeBase64 检查字符串是否看起来像 base64 编码
func looksLikeBase64(s string) bool {
	// base64 字符集：A-Z, a-z, 0-9, +, /, =
	// 简单检查：如果字符串很长且主要由这些字符组成，可能是 base64
	if len(s) < 100 {
		return false
	}
	
	// 检查前100个字符，如果大部分是 base64 字符，可能是 base64
	checkLen := 100
	if len(s) < checkLen {
		checkLen = len(s)
	}
	
	base64Chars := 0
	for i := 0; i < checkLen; i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || 
		   (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=' {
			base64Chars++
		}
	}
	
	// 如果超过 80% 是 base64 字符，认为是 base64
	return float64(base64Chars)/float64(checkLen) > 0.8
}

// isBase64Field 检查字段名是否是已知的 base64 字段
func isBase64Field(key string) bool {
	base64Fields := []string{
		"image_url",
		"input_reference",
		"Image",        // OpenAI Image 请求中的参考图数组
		"b64_json",     // OpenAI 响应中的 base64 图片
		"data",         // 某些响应中的 data 字段可能包含 base64
	}
	
	keyLower := strings.ToLower(key)
	for _, field := range base64Fields {
		if strings.ToLower(field) == keyLower {
			return true
		}
	}
	return false
}

// truncateBase64String 截断 base64 字符串值
func truncateBase64String(v interface{}, truncateLen int) interface{} {
	switch val := v.(type) {
	case string:
		if isBase64DataURI(val) || (len(val) > 100 && looksLikeBase64(val)) {
			if len(val) > truncateLen {
				return val[:truncateLen] + fmt.Sprintf("...[base64 truncated, len=%d]", len(val))
			}
		}
		return val

	case []interface{}:
		// 如果是数组（如 Image[]），处理每个元素
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = truncateBase64String(item, truncateLen)
		}
		return result

	default:
		return v
	}
}
