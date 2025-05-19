package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/weibaohui/openDeepWiki/pkg/models"
	"k8s.io/klog/v2"
)

type gitService struct{}

var localGitService = &gitService{}

// InitRepo 初始化或克隆Git仓库
func (g *gitService) InitRepo(repo *models.Repo) error {
	if repo == nil {
		return fmt.Errorf("repo is nil")
	}

	// 构建仓库本地路径
	// 将 URL 转换为目录格式 (例如: github.com/user/repo)
	cleanURL := cleanGitURL(repo.URL)
	repoPath := filepath.Join("data", "repos", cleanURL)

	// 确保父目录存在
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return fmt.Errorf("创建仓库目录失败: %v", err)
	}

	// 检查目录是否为空
	empty, err := isDirEmpty(repoPath)
	if err != nil {
		return fmt.Errorf("检查目录是否为空失败: %v", err)
	}

	// 如果目录不为空，先删除
	if !empty {
		if err := os.RemoveAll(repoPath); err != nil {
			return fmt.Errorf("清理已存在的仓库目录失败: %v", err)
		}
		// 重新创建目录
		if err := os.MkdirAll(repoPath, 0755); err != nil {
			return fmt.Errorf("重新创建仓库目录失败: %v", err)
		}
	}

	// 如果是Git仓库，执行clone操作
	if repo.RepoType == "git" && repo.URL != "" {
		// 准备git clone命令
		cmd := exec.Command("git", "clone")
		if repo.Branch != "" {
			cmd.Args = append(cmd.Args, "-b", repo.Branch)
		}
		cmd.Args = append(cmd.Args, repo.URL, repoPath)

		// 执行clone命令
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("克隆仓库失败: %v, 输出: %s", err, string(output))
		}
		klog.V(4).Infof("成功克隆仓库 %s 到 %s", repo.URL, repoPath)
	} else {
		// 如果不是Git仓库，初始化为空的Git仓库
		cmd := exec.Command("git", "init")
		cmd.Dir = repoPath
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("初始化Git仓库失败: %v, 输出: %s", err, string(output))
		}
		klog.V(4).Infof("成功初始化空的Git仓库在 %s", repoPath)
	}

	return nil
}

func (g *gitService) GetRepoPath(repo *models.Repo) (string, error) {
	if repo == nil {
		return "", fmt.Errorf("repo is nil")
	}

	// 构建仓库本地路径
	// 将 URL 转换为目录格式 (例如: github.com/user/repo)
	cleanURL := cleanGitURL(repo.URL)
	repoPath := filepath.Join("data", "repos", cleanURL)

	return repoPath, nil
}

// cleanGitURL 将 Git URL 转换为标准的目录格式
// 例如：
// https://github.com/user/repo.git -> github.com/user/repo
// cleanGitURL 将 Git 仓库的 URL 标准化为目录路径格式。
// 支持去除协议前缀、转换 SSH 风格地址为路径格式，并移除 .git 后缀。
func cleanGitURL(url string) string {
	// 移除协议前缀 (https://, git://)
	for _, prefix := range []string{"https://", "http://", "git://"} {
		if len(url) > len(prefix) && url[:len(prefix)] == prefix {
			url = url[len(prefix):]
			break
		}
	}

	// 处理 SSH 格式 (git@github.com:user/repo)
	for _, prefix := range []string{"git@"} {
		if len(url) > len(prefix) && url[:len(prefix)] == prefix {
			url = url[len(prefix):]
			// 将 : 替换为 /
			url = strings.Replace(url, ":", "/", 1)
			break
		}
	}

	// 移除 .git 后缀
	if strings.HasSuffix(url, ".git") {
		url = url[:len(url)-4]
	}

	return url
}

// isDirEmpty 判断指定目录是否为空。
// 如果目录不存在，返回 true 和 nil，表示目录视为“空”。
// 如果目录存在且无任何文件或子目录，返回 true。
// 如果目录中存在至少一个文件或子目录，返回 false。
// 发生其他错误时，返回 false 和相应的错误信息。
func isDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == nil {
		return false, nil
	}
	if err.Error() == "EOF" {
		return true, nil
	}
	return false, err
}
