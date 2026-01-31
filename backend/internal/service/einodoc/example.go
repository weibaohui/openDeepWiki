// Package einodoc 使用示例
//
// 基本使用示例：
//
//	import (
//	    "context"
//	    "github.com/opendeepwiki/backend/config"
//	    "github.com/opendeepwiki/backend/internal/pkg/llm"
//	    "github.com/opendeepwiki/backend/internal/service/einodoc"
//	)
//
//	func main() {
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
//	        log.Fatal(err)
//	    }
//
//	    // 解析仓库
//	    ctx := context.Background()
//	    result, err := service.ParseRepo(ctx, "https://github.com/example/repo.git")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // 输出结果
//	    fmt.Println(result.Document)
//	}
//
package einodoc
