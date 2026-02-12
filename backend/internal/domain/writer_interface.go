package domain

import (
	"context"
	"errors"
)

type WriterName string

var (
	DefaultWriter     WriterName = "DefaultWriter"
	DBModelWriter     WriterName = "DBModelWriter"
	APIWriter         WriterName = "APIWriter"
	UserRequestWriter WriterName = "UserRequestWriter"
	TitleRewriter     WriterName = "TitleRewriter"
	TocWriter         WriterName = "TocWriter"
)

type Writer interface {
	Name() WriterName
	Generate(ctx context.Context, localPath string, title string, taskID uint) (string, error)
}

// 错误定义
var (
	ErrInvalidLocalPath         = errors.New("invalid local path")
	ErrAgentExecutionFailed     = errors.New("agent execution failed")
	ErrEmptyContent             = errors.New("empty content")
	ErrNoAgentOutput            = errors.New("no agent output")
	ErrYAMLParseFailed          = errors.New("YAML parse failed")
	ErrTaskNotFound             = errors.New("task not found")
	ErrRepoNotFound             = errors.New("repo not found")
	ErrTaskCreationFailed       = errors.New("task creation failed")
	ErrDirMakerGenerationFailed = errors.New("dir maker generation failed")
)

// Agent 名称常量
const (
	AgentTocEditor       = "toc_editor"         // 目录制定者
	AgentTocChecker      = "toc_checker"        // 目录校验 Agent
	AgentGen             = "document_generator" // 文档生成 Agent
	AgentCheck           = "markdown_checker"   // Markdown 校验 Agent
	AgentDocCheck        = "document_checker"   // 文档校验 Agent
	AgentDBModelExplorer = "db_model_explorer"  // 数据库模型探索 Agent
	AgentMdCheck         = "markdown_checker"   // Markdown 校验 Agent
	AgentProblemSolver   = "problem_solver"     // 问题解决 Agent
	AgentAPIExplorer     = "api_explorer"       // API 探索 Agent
	AgentTitleRewriter   = "title_rewriter"     // 标题重写 Agent
)
