package volcengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client 封装火山引擎的调用客户端
type Client struct {
	ApiKey           string
	BaseURL          string
	ChatModelID      string
	EmbeddingModelID string
	HttpCli          *http.Client
}

// NewClient 初始化
func NewClient(apiKey, baseURL, chatModelID, embeddingModelID string) *Client {
	return &Client{
		// 👇 2. 关键修改：用 strings.TrimSpace 把 Key 前后的空格和换行符都剪掉
		ApiKey:           strings.TrimSpace(apiKey),
		BaseURL:          strings.TrimSpace(baseURL), // URL 最好也处理一下
		ChatModelID:      chatModelID,
		EmbeddingModelID: embeddingModelID,
		HttpCli:          &http.Client{Timeout: 60 * time.Second},
	}
}

// ================= Chat 相关结构 (保持不变) =================
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// ================= Embedding 相关结构 (大改！) =================

// MultimodalInput 多模态输入的单项结构
type MultimodalInput struct {
	Type string `json:"type"` // 固定为 "text"
	Text string `json:"text"` // 文本内容
}

// EmbeddingRequest 适配 Vision 模型的请求结构
type EmbeddingRequest struct {
	Model string            `json:"model"`
	Input []MultimodalInput `json:"input"` // 这里变成了一个对象数组
}

type EmbeddingResponse struct {
	Data struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ================= 方法实现 =================

// SendMessage 发送聊天消息 (逻辑微调，确保 URL 正确)
func (c *Client) SendMessage(content string) (string, error) {
	// 强制拼接正确的 Chat 地址
	// 假设 BaseURL 是 https://ark.cn-beijing.volces.com/api/v3
	url := "https://ark.cn-beijing.volces.com/api/v3/chat/completions"

	reqBody := ChatRequest{
		Model: c.ChatModelID,
		Messages: []Message{
			{Role: "user", Content: content},
		},
		Stream: false,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	fmt.Printf("正在请求聊天模型 (Model: %s)...\n", c.ChatModelID)
	resp, err := c.HttpCli.Do(req)
	if err != nil {
		return "", fmt.Errorf("网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Chat API报错 (%d): %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("解析Chat响应失败: %v", err)
	}
	if chatResp.Error.Message != "" {
		return "", fmt.Errorf("业务错误: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) > 0 {
		return chatResp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("未收到回复")
}

// CreateEmbedding 生成向量 (重点修改！)
func (c *Client) CreateEmbedding(text string) ([]float32, error) {
	// 1. 🛑 必须使用多模态专用的 API 地址
	url := "https://ark.cn-beijing.volces.com/api/v3/embeddings/multimodal"

	// 2. 🛑 构造多模态请求体 (把文本包装进 Input 对象)
	reqBody := EmbeddingRequest{
		Model: c.EmbeddingModelID,
		Input: []MultimodalInput{
			{
				Type: "text", // 告诉模型这是文本
				Text: text,   // 你的文本内容
			},
		},
	}

	jsonData, _ := json.Marshal(reqBody)

	// 3. 发送请求
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	// 记得保留之前的 apiKey TrimSpace 修改
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HttpCli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 4. 解析结果
	var embedResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		// 打印一下原始 body 方便调试，如果还报错的话
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if embedResp.Error.Message != "" {
		return nil, fmt.Errorf("Embedding API错误: %s", embedResp.Error.Message)
	}

	// 🚨 修改点：这里不再检查 len(Data) == 0，也不用 [0]
	if len(embedResp.Data.Embedding) == 0 {
		return nil, fmt.Errorf("没有返回向量数据")
	}

	return embedResp.Data.Embedding, nil
}
