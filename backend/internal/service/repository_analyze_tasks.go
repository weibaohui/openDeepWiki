package service

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

func (s *RepositoryService) AnalyzeDatabaseModel(ctx context.Context, repoID uint) (*model.Task, error) {
	return s.runAnalyzeTask(ctx, repoID, analyzeTaskSpec{
		taskTitle: "数据库模型分析",
		sortOrder: 10,
		validate: func() error {
			if s.dbModelParser == nil || s.docService == nil {
				return fmt.Errorf("数据库模型解析服务未初始化")
			}
			return nil
		},
		generator: func(ctx context.Context, repo *model.Repository, task *model.Task) (string, error) {
			return s.dbModelParser.Generate(ctx, repo.LocalPath, task.Title, repo.ID, task.ID)
		},
	})
}

// AnalyzeAPI 异步触发API接口分析任务。
func (s *RepositoryService) AnalyzeAPI(ctx context.Context, repoID uint) (*model.Task, error) {
	return s.runAnalyzeTask(ctx, repoID, analyzeTaskSpec{

		taskTitle: "API接口分析",
		sortOrder: 20,
		validate: func() error {
			if s.apiAnalyzer == nil || s.docService == nil {
				return fmt.Errorf("API接口分析服务未初始化")
			}
			return nil
		},
		generator: func(ctx context.Context, repo *model.Repository, task *model.Task) (string, error) {
			return s.apiAnalyzer.Generate(ctx, repo.LocalPath, task.Title, repo.ID, task.ID)
		},
	})
}

// AnalyzeProblem 异步触发问题分析任务。
func (s *RepositoryService) AnalyzeProblem(ctx context.Context, repoID uint, problem string) (*model.Task, error) {
	// 截取问题前20个字符作为标题
	title := "问题分析: " + problem
	if len(title) > 50 {
		runes := []rune(title)
		if len(runes) > 47 {
			title = string(runes[:47]) + "..."
		}
	}
	return s.runAnalyzeTask(ctx, repoID, analyzeTaskSpec{
		taskTitle: title,
		sortOrder: 30,
		validate: func() error {
			if s.problemAnalyzer == nil || s.docService == nil {
				return fmt.Errorf("问题分析服务未初始化")
			}
			return nil
		},
		generator: func(ctx context.Context, repo *model.Repository, task *model.Task) (string, error) {
			return s.problemAnalyzer.Generate(ctx, repo.LocalPath, problem, repo.ID, task.ID)
		},
		afterSuccess: func(ctx context.Context, repo *model.Repository, task *model.Task) error {
			//进行标题重写
			if s.titleRewriter != nil {
				_, _, _, err := s.titleRewriter.RewriteTitle(ctx, task.DocID)
				if err != nil {
					klog.Errorf("标题重写失败: repoID=%d, taskID=%d, docID=%d, error=%v", repo.ID, task.ID, task.DocID, err)
				}
			}
			return nil
		},
	})
}
