// 内部使用：pkg/middleware/zero_poetry.go
// 作者：mrlaoliai
package middleware

import (
	"regexp"
)

// ZeroPoetryProcessor 强制执行输出的纯净性，移除 AI 的社交废话
type ZeroPoetryProcessor struct {
	filters []*regexp.Regexp
}

func NewZeroPoetryProcessor() *ZeroPoetryProcessor {
	return &ZeroPoetryProcessor{
		filters: []*regexp.Regexp{
			// 1. 身份声明类
			regexp.MustCompile(`(?i)(as an ai language model,?|i am an ai,?)`),
			// 2. 确认/社交辞令类
			regexp.MustCompile(`(?i)(i'm here to help\.?|understood,? i will\.?|certainly!?|surely!?)`),
			// 3. 引导过渡类 (例如：Here is the updated code:)
			regexp.MustCompile(`(?i)(here is the .*?:|i've updated the .*? to:)\n?`),
			// 4. 结尾冗余类
			regexp.MustCompile(`(?i)(let me know if you need anything else\.?)`),
		},
	}
}

// Process 针对文本执行清洗。
// 注意：为了保证流式输出的连贯性，严禁使用 TrimSpace，必须保留原始空白符。
func (z *ZeroPoetryProcessor) Process(content string) string {
	if content == "" {
		return ""
	}

	result := content
	for _, re := range z.filters {
		// 使用 ReplaceAllString 直接替换为空，不触碰周边空格
		result = re.ReplaceAllString(result, "")
	}

	return result
}
