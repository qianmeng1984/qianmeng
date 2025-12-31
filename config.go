package main

type Config struct {
	ServerPort string
	// VolcEngine 配置结构体
	VolcEngine struct {
		APIKey           string
		ChatModelID      string // 👈 改名了！之前叫 ModelID
		BaseURL          string // 👈 这里是你缺少的字段定义！
		EmbeddingModelID string // 👈 新增：嵌入模型 ID
	}
	Database struct {
		DSN string // 数据库连接字符串
	}
}

func LoadConfig() *Config {
	// 模拟加载配置
	return &Config{
		ServerPort: ":8080",
		VolcEngine: struct {
			APIKey           string
			ChatModelID      string
			BaseURL          string
			EmbeddingModelID string
		}{
			// 1. 填入你的 API Key
			APIKey: "xxx",

			// 2. 填入你的模型接入点 ID (ep-xxxxx)
			// 如果控制台还没配好，暂时填 "doubao-pro-32k"
			ChatModelID: "doubao-1-5-vision-pro-32k-250115",

			// 3. 👈 这一行补上了！这是火山方舟的标准地址
			BaseURL: "https://ark.cn-beijing.volces.com/api/v3/chat/completions",

			EmbeddingModelID: "doubao-embedding-vision-250615",
		},
		Database: struct{ DSN string }{
			DSN: "host=localhost user=rune password=rune_password dbname=rag_db port=5432 sslmode=disable TimeZone=Asia/Shanghai",
		},
	}
}
