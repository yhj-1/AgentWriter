package model

import "encoding/json"

// parseJSON 解析 JSON 字符串到目标对象
func parseJSON(jsonStr string, target interface{}) {
	if jsonStr == "" {
		return
	}
	_ = json.Unmarshal([]byte(jsonStr), target)
}

// toJSON 将对象转为 JSON 字符串
func toJSON(obj interface{}) string {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(bytes)
}
