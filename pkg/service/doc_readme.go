package service

import (
	"context"
)

type docReadmeService struct {
	parent *docService
}

func (s *docReadmeService) prompt() string {
	return `
		你是一个文档生成助手，你需要根据以下信息生成一个README.md文件
		`
}
func (s *docReadmeService) Generate(ctx context.Context) error {
	reader, err := s.parent.chat(ctx, s.prompt())
	if err != nil {
		return err

	}
	all, err := s.parent.readAll(ctx, reader)
	if err != nil {
		return err
	}

	return s.parent.writeFile(ctx, all)
}
