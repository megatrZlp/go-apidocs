package source

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gfile"
)

// FetchURL 拉取远程文本内容（HTTP 200 视为成功），返回字符串。
func FetchURL(u string) (string, error) {
	resp, err := http.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http status %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// LoadSpecFromSource 按 src 加载 OpenAPI 文档，支持本地、HTTP、file://；返回解析后的 JSON 与原始文本。
// 说明：会归一化 Windows 路径并移除片段（例如 #Lx-y），兼容 IDE 复制的路径片段。
func LoadSpecFromSource(src string) (*gjson.Json, string, error) {
	// 归一化 src：移除片段、处理 IDE 复制的前导 #、兼容 Windows 路径与 file:// 前缀
	s := strings.TrimSpace(src)
	// 移除开头的 # 或 #/，避免误判为 JSON Pointer
	if strings.HasPrefix(s, "#/") {
		s = s[1:]
	} else if strings.HasPrefix(s, "#") {
		s = s[1:]
	}
	// 丢弃 # 后的片段（例如 #L10-20）
	if i := strings.Index(s, "#"); i >= 0 {
		s = s[:i]
	}
	// 处理形如 /C:/path 的前导斜杠（Windows）
	if strings.HasPrefix(s, "/") && len(s) >= 3 && s[2] == ':' {
		s = s[1:]
	}
	// 兼容 file:// 前缀与 /C:/ 的组合
	if strings.HasPrefix(strings.ToLower(s), "file://") {
		s = strings.TrimPrefix(s, "file://")
		if strings.HasPrefix(s, "/") && len(s) >= 3 && s[2] == ':' {
			s = s[1:]
		}
	}
	var content string
	var err error
	// http(s) 走远程拉取；否则按本地文件读取
	if strings.HasPrefix(strings.ToLower(s), "http") {
		content, err = FetchURL(s)
		if err != nil {
			return nil, "", err
		}
	} else {
		content = gfile.GetContents(s)
		if content == "" {
			return nil, "", fmt.Errorf("empty content from %s", s)
		}
	}
	// 解析 JSON 文本为 gjson.Json
	j, e := gjson.LoadContent([]byte(content))
	if e != nil {
		return nil, "", e
	}
	return j, content, nil
}

// OrderedPathsFromContent 使用流式解码提取 paths 键的原始顺序，保证菜单与正文一致。
func OrderedPathsFromContent(content string) []string {
	// 使用流式解码器，精确读取 paths 下键的出现顺序以用于菜单与正文排序
	dec := json.NewDecoder(strings.NewReader(content))
	// 找到 "paths" 键
	for {
		tok, err := dec.Token()
		if err != nil {
			return []string{}
		}
		if key, ok := tok.(string); ok && key == "paths" {
			break
		}
	}
	// 读取 paths 对象开始标记
	if _, err := dec.Token(); err != nil {
		return []string{}
	}
	// 遍历 paths 的每个键，记录顺序
	keys := make([]string, 0, 64)
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			break
		}
		k, _ := t.(string)
		keys = append(keys, k)
		// 跳过对应值的解码（保留原始顺序即可）
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			break
		}
	}
	// 读取 paths 对象结束标记
	_, _ = dec.Token()
	return keys
}
