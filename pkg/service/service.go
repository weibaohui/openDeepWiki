package service

var localChatService = &chatService{}
var localUserService = &userService{}
var localOperationLogService = NewOperationLogService()
var localAiService = &aiService{}
var localMcpService = &mcpService{}

// ChatService 返回或创建一个新的chatService实例
func ChatService() *chatService {
	return localChatService
}

func UserService() *userService {
	return localUserService
}

func OperationLogService() *operationLogService {
	return localOperationLogService
}

func AIService() *aiService {
	return localAiService

}

func ConfigService() *configService {
	return NewConfigService()
}

// McpService 返回本地的 mcpService 实例指针。
func McpService() *mcpService {

	return localMcpService
}

// GitService 返回本地的 gitService 实例指针。
func GitService() *gitService {
	return localGitService
}
