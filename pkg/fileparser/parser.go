package fileparser

import (
	"bytes"
	"fmt"
	"strings"

	// 引入 UniPDF 的核心模块
	"github.com/unidoc/unipdf/v3/common/license"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

// init 函数会在程序启动时自动执行
func init() {
	// 设置为社区免费模式（这会避免一些未授权的报错，虽然控制台可能还是会打印一行 copyright log，忽略即可）
	// 注意：如果你用于商业项目需要购买 Key，毕设直接用即可。
	license.SetMeteredKey("无需Key-社区版自动生效")
}

// ParseContent 主入口
func ParseContent(fileBytes []byte, fileName string) (string, error) {
	lowerName := strings.ToLower(fileName)

	if strings.HasSuffix(lowerName, ".txt") || strings.HasSuffix(lowerName, ".md") {
		return string(fileBytes), nil
	}

	if strings.HasSuffix(lowerName, ".pdf") {
		return parsePDF(fileBytes)
	}

	return "", fmt.Errorf("不支持的文件格式: %s", fileName)
}

// parsePDF 使用 UniPDF 进行解析
func parsePDF(content []byte) (string, error) {
	// 1. 创建 PDF 读取器
	pdfReader, err := model.NewPdfReader(bytes.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("PDF 读取失败: %v", err)
	}

	// 2. 获取总页数
	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return "", fmt.Errorf("无法获取页码: %v", err)
	}

	var textBuilder strings.Builder

	// 3. 遍历每一页提取文字
	for i := 0; i < numPages; i++ {
		pageNum := i + 1
		page, err := pdfReader.GetPage(pageNum)
		if err != nil {
			continue
		}

		// 创建提取器
		ex, err := extractor.New(page)
		if err != nil {
			continue
		}

		// 提取文字
		text, err := ex.ExtractText()
		if err != nil {
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	result := textBuilder.String()
	if strings.TrimSpace(result) == "" {
		return "", fmt.Errorf("解析结果为空，可能是纯图片扫描件")
	}

	return result, nil
}
