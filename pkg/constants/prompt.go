package constants

// promptKey 是用于 context 中存储 prompt 相关信息的 key 类型
type promptKey string

const (
	// SystemPrompt 系统提示信息的 key
	SystemPrompt promptKey = "systemPrompt"
)
