package service

import (
	"context"
	"io"

	"github.com/weibaohui/openDeepWiki/pkg/models"
	"k8s.io/klog/v2"
)

type docService struct {
	repo *models.Repo
}

var localReadmeService = &docReadmeService{
	parent: localDocService,
}

func NewDocService(repo *models.Repo) *docService {
	return &docService{
		repo: repo,
	}
}

func (d *docService) ReadmeService() *docReadmeService {
	return localReadmeService
}

// handleAIChat 处理AI聊天的核心逻辑，包括消息处理和响应读取
//
// 参数:
// - c: gin上下文
// - message: 用户输入的消息
// - responseCh: 用于实时返回AI响应的channel
// 返回:
// - chan struct{}: 用于从外部终止处理过程的channel
// - error: 处理过程中的错误
func (d *docService) chat(ctx context.Context, message string) (io.Reader, error) {
	// 创建一个带有读写功能的管道
	pr, pw := io.Pipe()

	// 启动一个goroutine来处理AI服务的输出
	go func() {
		defer pw.Close()
		// 调用AI服务处理消息，将输出写入管道
		err := ChatService().RunOneRound(ctx, message, pw)
		if err != nil {
			klog.Errorf("AI处理消息失败: %v", err)
			return
		}
	}()

	return pr, nil
}

func (d *docService) writeFile(ctx context.Context, s string) error {
	klog.Infof("收到待写入文件内容:%s", s)
	return nil
}

func (d *docService) readAll(ctx context.Context, reader io.Reader) (string, error) {

	var all string
	// 创建一个缓冲区用于临时存储读取的数据
	buf := make([]byte, 1024)

	// 启动一个goroutine来持续读取输出
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			all += string(buf[:n])
			// 输出到控制台
			klog.V(6).Infof("AI响应: %s", string(buf[:n]))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			klog.Errorf("读取AI响应失败: %v", err)
			break
		}
	}
	return all, nil
}
