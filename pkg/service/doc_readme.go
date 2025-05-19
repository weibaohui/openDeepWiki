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
		仓库名称是%s
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
