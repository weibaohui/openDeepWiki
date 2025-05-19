package service

var localChatService = &chatService{
	MaxIterations: 10,
}
var localUserService = &userService{}
var localOperationLogService = NewOperationLogService()
var localAiService = &aiService{}
var localMcpService = &mcpService{}
var localDocService = &docService{}

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

func McpService() *mcpService {

	return localMcpService
}

func GitService() *gitService {
	return localGitService
}
