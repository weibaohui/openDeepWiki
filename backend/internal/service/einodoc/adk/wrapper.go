package adk

import (
	"github.com/cloudwego/eino/schema"
)

// ==================== 辅助函数 ====================

// MessageFromSchema 将 schema.Message 转换为 adk.Message
// adk.Message 实际上是 *schema.Message 的别名
func MessageFromSchema(msg *schema.Message) *schema.Message {
	return msg
}
