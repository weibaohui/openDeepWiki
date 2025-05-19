package service

import (
	"context"
	"fmt"
)

type docReadmeService struct {
	parent *docService
}

func (d *docService) ReadmeService() *docReadmeService {
	return &docReadmeService{
		parent: d,
	}
}
func (s *docReadmeService) prompt(ctx context.Context) string {
	prompt := `
		你是一个文档生成助手，你需要根据以下信息生成一个README.md文件。
		仓库存放路径在%s
		仓库名称是%s。
		请你根据存放路径，先读取仓库文件夹目录结构，再根据目录结构，读取仓库的中的必要文件，然后根据文件内容，生成一个README.md文件。
		原仓库中的Readme文档，只能作为参考。	
		请你生成README.md文件，包含以下信息：
		1. 仓库名称
		2. 仓库描述
		3. 仓库的使用方法
		4. 仓库的依赖
		5. 仓库的安装方法
		6. 仓库的配置方法
		7. 仓库的使用示例
		8. 仓库的注意事项
		9. 仓库的贡献者
		10. 仓库的许可证
		11. 仓库的版本号
		12. 仓库的更新日志
		13. 仓库的问题反馈
		14. 仓库的贡献指南
		等相关信息。
		请务必使用<finalResult></finalResult>包裹最终结果。
		`
	path, _ := s.parent.RepoService().GetRepoPath(ctx)
	repName := s.parent.RepoService().GetRepoName(ctx)
	return fmt.Sprintf(prompt, path, repName)
}
func (s *docReadmeService) Generate(ctx context.Context) error {
	reader, err := s.parent.chat(ctx, s.prompt(ctx))
	if err != nil {
		return err

	}
	all, err := s.parent.readAll(ctx, reader)
	if err != nil {
		return err
	}

	return s.parent.writeFile(ctx, all)
}
