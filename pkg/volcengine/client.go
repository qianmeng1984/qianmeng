package volcengine

import (
	"bufio"
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
		ApiKey:           strings.TrimSpace(apiKey),
		BaseURL:          strings.TrimSpace(baseURL),
		ChatModelID:      chatModelID,
		EmbeddingModelID: embeddingModelID,
		HttpCli:          &http.Client{Timeout: 60 * time.Second},
	}
}

// ================= 通用结构 =================

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ================= Chat 相关结构 =================

// 👇👇👇 新增：流式选项结构 👇👇👇
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	// 👇👇👇 新增：告诉 API 我要看账单 👇👇👇
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// 普通响应结构
type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// 流式响应块结构
type StreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage Usage `json:"usage"` // 有些流式最后一块会带 Usage
}

// ================= Embedding 相关结构 =================

type MultimodalInput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type EmbeddingRequest struct {
	Model string            `json:"model"`
	Input []MultimodalInput `json:"input"`
}

type EmbeddingResponse struct {
	Data struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Usage Usage `json:"usage"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ================= 方法实现 =================

// SendMessage 普通对话 (保留用于查询分析)
func (c *Client) SendMessage(content string) (string, Usage, error) {
	url := "https://ark.cn-beijing.volces.com/api/v3/chat/completions"
	reqBody := ChatRequest{
		Model:    c.ChatModelID,
		Messages: []Message{{Role: "user", Content: content}},
		Stream:   false,
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	emptyUsage := Usage{}
	resp, err := c.HttpCli.Do(req)
	if err != nil {
		return "", emptyUsage, fmt.Errorf("网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", emptyUsage, fmt.Errorf("Chat API报错 (%d): %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", emptyUsage, fmt.Errorf("解析Chat响应失败: %v", err)
	}
	if chatResp.Error.Message != "" {
		return "", emptyUsage, fmt.Errorf("业务错误: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) > 0 {
		return chatResp.Choices[0].Message.Content, chatResp.Usage, nil
	}
	return "", emptyUsage, fmt.Errorf("未收到回复")
}

// SendMessageStream 流式发送消息
func (c *Client) SendMessageStream(content string, onToken func(string)) (Usage, error) {
	url := "https://ark.cn-beijing.volces.com/api/v3/chat/completions"
	reqBody := ChatRequest{
		Model:    c.ChatModelID,
		Messages: []Message{{Role: "user", Content: content}},
		Stream:   true, // 开启流式
		// 👇👇👇 关键修改：显式要求返回 Usage 👇👇👇
		StreamOptions: &StreamOptions{
			IncludeUsage: true,
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	emptyUsage := Usage{}
	resp, err := c.HttpCli.Do(req)
	if err != nil {
		return emptyUsage, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return emptyUsage, fmt.Errorf("Stream API报错 (%d): %s", resp.StatusCode, string(body))
	}

	// 使用 Scanner 逐行读取 SSE 数据
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		data = strings.TrimSpace(data)

		if data == "[DONE]" {
			break
		}

		var streamResp StreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue
		}

		// 提取增量内容
		if len(streamResp.Choices) > 0 {
			chunk := streamResp.Choices[0].Delta.Content
			if chunk != "" {
				onToken(chunk)
			}
		}

		// 提取 Usage (通常在流的最后一条消息里)
		if streamResp.Usage.TotalTokens > 0 {
			emptyUsage = streamResp.Usage
		}
	}

	return emptyUsage, nil
}

// CreateEmbedding 生成向量
func (c *Client) CreateEmbedding(text string) ([]float32, Usage, error) {
	url := "https://ark.cn-beijing.volces.com/api/v3/embeddings/multimodal"
	reqBody := EmbeddingRequest{
		Model: c.EmbeddingModelID,
		Input: []MultimodalInput{{Type: "text", Text: text}},
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	emptyUsage := Usage{}
	resp, err := c.HttpCli.Do(req)
	if err != nil {
		return nil, emptyUsage, err
	}
	defer resp.Body.Close()

	var embedResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, emptyUsage, fmt.Errorf("解析Embedding响应失败: %v", err)
	}
	if embedResp.Error.Message != "" {
		return nil, emptyUsage, fmt.Errorf("Embedding API错误: %s", embedResp.Error.Message)
	}
	if len(embedResp.Data.Embedding) == 0 {
		return embedResp.Data.Embedding, embedResp.Usage, nil
	}
	return embedResp.Data.Embedding, embedResp.Usage, nil
}
