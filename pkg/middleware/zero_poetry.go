// 基础中间件：pkg/middleware/zero_poetry.go
// 作者：mrlaoliai
package middleware

import (
	"regexp"
	"strings"
)

// ZeroPoetryProcessor 强制执行输出的纯净性
type ZeroPoetryProcessor struct {
	filters []*regexp.Regexp
}

// NewZeroPoetryProcessor 装载预设的正则约束
func NewZeroPoetryProcessor() *ZeroPoetryProcessor {
	return &ZeroPoetryProcessor{
		filters: []*regexp.Regexp{
			// 过滤身份声明
			regexp.MustCompile(`(?i)(as an ai language model,?|i am an ai,?)`),
			// 过滤社交辞令
			regexp.MustCompile(`(?i)(i'm here to help\.?|understood,? i will\.?)`),
			// 过滤过度解释
			regexp.MustCompile(`(?i)certainly!? here is the.*?:\n`),
		},
	}
}

// Process 针对传入的增量文本或完整文本执行清洗
func (z *ZeroPoetryProcessor) Process(content string) string {
	result := content
	for _, re := range z.filters {
		result = re.ReplaceAllString(result, "")
	}
	return strings.TrimSpace(result)
}
