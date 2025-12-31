package textsplitter

// SplitText 简单粗暴的切分逻辑
// text: 原始长文
// chunkSize: 每段大概多少字 (推荐 300-500)
// overlap: 重叠部分 (防止切分时把一句话切断了语义，推荐 50)
func SplitText(text string, chunkSize int, overlap int) []string {
	var chunks []string
	runes := []rune(text) // 转成 rune 处理中文，防止乱码
	length := len(runes)

	if length <= chunkSize {
		return []string{text}
	}

	for i := 0; i < length; i += (chunkSize - overlap) {
		end := i + chunkSize
		if end > length {
			end = length
		}

		// 截取片段
		chunk := string(runes[i:end])
		chunks = append(chunks, chunk)

		// 如果已经到了末尾，就停止
		if end == length {
			break
		}
	}
	return chunks
}
