package service

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"k8s.io/klog/v2"

	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/git"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

// CloneRepository 手动触发克隆仓库（用于克隆失败的仓库）
func (s *RepositoryService) CloneRepository(ctx context.Context, repoID uint) error {
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return fmt.Errorf("获取仓库失败: %w", err)
	}

	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if currentStatus == statemachine.RepoStatusCloning || currentStatus == statemachine.RepoStatusAnalyzing {
		return fmt.Errorf("仓库状态不允许重新克隆: current=%s", currentStatus)
	}

	// 先删除已存在的本地目录（如果有）
	if repo.LocalPath != "" {
		_ = git.RemoveRepo(repo.LocalPath)
	}

	// 重新生成路径
	repoName := git.ParseRepoName(repo.URL)
	repo.LocalPath = filepath.Join(s.cfg.Data.RepoDir, repoName+"-"+fmt.Sprintf("%d", time.Now().Unix()))

	// 保存新路径
	if err := s.repoRepo.Save(repo); err != nil {
		return fmt.Errorf("更新仓库路径失败: %w", err)
	}

	// 异步克隆
	go s.cloneRepository(repoID)

	return nil
}

// cloneRepository 克隆仓库
// 状态迁移: pending -> cloning -> ready/error
func (s *RepositoryService) cloneRepository(repoID uint) {
	klog.V(6).Infof("开始克隆仓库: repoID=%d", repoID)

	// 获取仓库
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		klog.Errorf("获取仓库失败: repoID=%d, error=%v", repoID, err)
		return
	}

	// 状态迁移: pending -> cloning
	oldStatus := statemachine.RepositoryStatus(repo.Status)
	newStatus := statemachine.RepoStatusCloning

	// 使用状态机验证迁移
	if err := s.repoStateMachine.Transition(oldStatus, newStatus, repoID); err != nil {
		klog.Errorf("仓库状态迁移失败: repoID=%d, error=%v", repoID, err)
		return
	}

	// 更新数据库状态
	repo.Status = string(newStatus)
	if err := s.repoRepo.Save(repo); err != nil {
		klog.Errorf("更新仓库状态失败: repoID=%d, error=%v", repoID, err)
		return
	}

	klog.V(6).Infof("仓库状态已更新为 cloning: repoID=%d", repoID)

	// 执行克隆
	err = git.Clone(git.CloneOptions{
		URL:       repo.URL,
		TargetDir: repo.LocalPath,
	})

	if err != nil {
		// 克隆失败，状态迁移: cloning -> error
		repo.Status = string(statemachine.RepoStatusError)
		repo.ErrorMsg = fmt.Sprintf("克隆失败: %v", err)

		if err := s.repoRepo.Save(repo); err != nil {
			klog.Errorf("更新仓库状态失败: repoID=%d, error=%v", repoID, err)
		}

		klog.Errorf("仓库克隆失败: repoID=%d, error=%v", repoID, err)
		return
	}

	sizeMB, err := git.DirSizeMB(repo.LocalPath)
	if err != nil {
		klog.Errorf("计算仓库大小失败: repoID=%d, error=%v", repoID, err)
	} else {
		repo.SizeMB = sizeMB
		klog.V(6).Infof("仓库大小已记录: repoID=%d, sizeMB=%.2f", repoID, sizeMB)
	}

	branch, commit, err := git.GetBranchAndCommit(repo.LocalPath)
	if err != nil {
		klog.Errorf("获取仓库分支与提交信息失败: repoID=%d, error=%v", repoID, err)
	} else {
		repo.CloneBranch = branch
		repo.CloneCommit = commit
		klog.V(6).Infof("仓库分支与提交信息已记录: repoID=%d, branch=%s, commit=%s", repoID, branch, commit)
	}

	// 克隆成功，状态迁移: cloning -> ready
	repo.Status = string(statemachine.RepoStatusReady)
	repo.ErrorMsg = ""

	if err := s.repoRepo.Save(repo); err != nil {
		klog.Errorf("更新仓库状态失败: repoID=%d, error=%v", repoID, err)
		return
	}

	klog.V(6).Infof("仓库克隆成功，状态已更新为 ready: repoID=%d, localPath=%s", repoID, repo.LocalPath)
}
