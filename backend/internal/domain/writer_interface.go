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
)

type Writer interface {
	Name() WriterName
	Generate(ctx context.Context, localPath string, title string, taskID uint) (string, error)
}

// 错误定义
var (
	ErrInvalidLocalPath     = errors.New("invalid local path")
	ErrAgentExecutionFailed = errors.New("agent execution failed")
	ErrEmptyContent         = errors.New("empty content")
	ErrNoAgentOutput        = errors.New("no agent output")
)

// Agent 名称常量
const (
	AgentGen             = "document_generator" // 文档生成 Agent
	AgentCheck           = "markdown_checker"   // Markdown 校验 Agent
	AgentDocCheck        = "document_checker"   // 文档校验 Agent
	AgentDBModelExplorer = "db_model_explorer"
	AgentMdCheck         = "markdown_checker"
	AgentProblemSolver   = "problem_solver"
	AgentAPIExplorer     = "api_explorer"
	AgentTitleRewriter   = "title_rewriter"
)
