// Package einodoc 使用示例
//
// 基本使用示例：
//
//	import (
//	    "context"
//	    "log"
//	    "github.com/opendeepwiki/backend/config"
//	    "github.com/opendeepwiki/backend/internal/pkg/llm"
//	    "github.com/opendeepwiki/backend/internal/service/einodoc"
//	)
//
//	func main() {
//	    // 加载配置
//	    cfg := config.GetConfig()
//
//	    // 创建 LLM 客户端
//	    llmClient := llm.NewClient(
//	        cfg.LLM.APIURL,
//	        cfg.LLM.APIKey,
//	        cfg.LLM.Model,
//	        cfg.LLM.MaxTokens,
//	    )
//
//	    // 创建服务
//	    service, err := einodoc.NewRepoDocService(cfg.Data.RepoDir, llmClient)
//	    if err != nil {
//	        log.Fatalf("创建服务失败: %v", err)
//	    }
//
//	    // 解析仓库
//	    ctx := context.Background()
//	    result, err := service.ParseRepo(ctx, "https://github.com/example/repo.git")
//	    if err != nil {
//	        log.Fatalf("解析仓库失败: %v", err)
//	    }
//
//	    // 输出结果
//	    fmt.Printf("Repository: %s\n", result.RepoURL)
//	    fmt.Printf("Type: %s\n", result.RepoType)
//	    fmt.Printf("Tech Stack: %v\n", result.TechStack)
//	    fmt.Printf("Sections: %d\n", result.SectionsCount)
//	    fmt.Printf("Document length: %d chars\n", len(result.Document))
//	}
//
// 高级使用示例（使用 EinoRepoDocService）：
//
//	func advancedExample() {
//	    cfg := config.GetConfig()
//	    llmClient := llm.NewClient(...)
//
//	    // 创建高级服务
//	    service, err := einodoc.NewEinoRepoDocService(cfg.Data.RepoDir, llmClient)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // 获取工具列表（可用于扩展）
//	    tools := service.GetTools()
//	    fmt.Printf("Available tools: %d\n", len(tools))
//
//	    // 获取 ChatModel（可用于自定义调用）
//	    chatModel := service.GetChatModel()
//
//	    // 解析仓库
//	    ctx := context.Background()
//	    result, err := service.ParseRepo(ctx, "https://github.com/example/repo.git")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    fmt.Printf("Document generated: %d chars\n", len(result.Document))
//	}
//
// 调试日志输出：
//
// 本包使用 klog.V(6).Infof() 输出详细日志，可以通过设置日志级别查看：
//
//	import "k8s.io/klog/v2"
//
//	func init() {
//	    // 设置日志级别为 6 以查看详细输出
//	    flag.Set("v", "6")
//	    klog.InitFlags(nil)
//	}
//
// 日志输出内容包括：
// - 工具创建和执行信息
// - LLM 调用参数和结果
// - Workflow 各步骤执行状态
// - 状态变更信息
//
package einodoc
