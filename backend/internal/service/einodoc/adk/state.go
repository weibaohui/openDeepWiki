package adk

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/opendeepwiki/backend/internal/service/einodoc"
	"k8s.io/klog/v2"
)

// StateManager ADK 模式状态管理器
// 负责管理 Workflow 执行过程中的共享状态
type StateManager struct {
	state *einodoc.RepoDocState
}

// NewStateManager 创建新的状态管理器
func NewStateManager(repoURL, localPath string) *StateManager {
	return &StateManager{
		state: einodoc.NewRepoDocState(repoURL, localPath),
	}
}

// GetState 获取当前状态
func (sm *StateManager) GetState() *einodoc.RepoDocState {
	return sm.state
}

// ToJSON 将状态转换为 JSON 字符串
func (sm *StateManager) ToJSON() string {
	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		klog.Errorf("[StateManager] 状态序列化失败: %v", err)
		return "{}"
	}
	return string(data)
}

// SetRepoTree 设置仓库目录结构
func (sm *StateManager) SetRepoTree(tree string) {
	sm.state.SetRepoTree(tree)
}

// GetRepoTree 获取仓库目录结构
func (sm *StateManager) GetRepoTree() string {
	return sm.state.RepoTree
}

// SetRepoInfo 设置仓库信息
func (sm *StateManager) SetRepoInfo(repoType string, techStack []string) {
	sm.state.SetRepoInfo(repoType, techStack)
}

// GetRepoInfo 获取仓库信息
func (sm *StateManager) GetRepoInfo() (string, []string) {
	return sm.state.RepoType, sm.state.TechStack
}

// SetOutline 设置文档大纲
func (sm *StateManager) SetOutline(outline []einodoc.Chapter) {
	sm.state.SetOutline(outline)
}

// GetOutline 获取文档大纲
func (sm *StateManager) GetOutline() []einodoc.Chapter {
	return sm.state.Outline
}

// SetSectionContent 设置小节内容
func (sm *StateManager) SetSectionContent(chapterIdx, sectionIdx int, content string) {
	sm.state.SetSectionContent(chapterIdx, sectionIdx, content)
}

// GetSectionContent 获取小节内容
func (sm *StateManager) GetSectionContent(chapterIdx, sectionIdx int) string {
	return sm.state.GetSectionContent(chapterIdx, sectionIdx)
}

// SetLocalPath 设置本地路径
func (sm *StateManager) SetLocalPath(path string) {
	sm.state.LocalPath = path
}

// GetLocalPath 获取本地路径
func (sm *StateManager) GetLocalPath() string {
	return sm.state.LocalPath
}

// BuildResult 构建最终结果
func (sm *StateManager) BuildResult() *einodoc.RepoDocResult {
	// 组装最终文档
	doc := sm.buildFinalDocument()

	return &einodoc.RepoDocResult{
		RepoURL:         sm.state.RepoURL,
		LocalPath:       sm.state.LocalPath,
		RepoType:        sm.state.RepoType,
		TechStack:       sm.state.TechStack,
		Outline:         sm.state.Outline,
		Document:        doc,
		SectionsCount:   len(sm.state.SectionsContent),
		Completed:       true,
		SectionsContent: sm.state.SectionsContent,
	}
}

// buildFinalDocument 组装最终文档
func (sm *StateManager) buildFinalDocument() string {
	var doc strings.Builder

	doc.WriteString(fmt.Sprintf("# %s 项目文档\n\n", sm.extractRepoName()))
	doc.WriteString(fmt.Sprintf("**仓库地址:** %s\n\n", sm.state.RepoURL))
	doc.WriteString(fmt.Sprintf("**项目类型:** %s\n\n", sm.state.RepoType))
	doc.WriteString(fmt.Sprintf("**技术栈:** %v\n\n", sm.state.TechStack))
	doc.WriteString("---\n\n")

	for chIdx, chapter := range sm.state.Outline {
		doc.WriteString(fmt.Sprintf("## %s\n\n", chapter.Title))

		for secIdx, section := range chapter.Sections {
			doc.WriteString(fmt.Sprintf("### %s\n\n", section.Title))

			content := sm.state.GetSectionContent(chIdx, secIdx)
			if content != "" {
				doc.WriteString(content)
				doc.WriteString("\n\n")
			} else {
				doc.WriteString("*（此小节内容待生成）*\n\n")
			}
		}
	}

	return doc.String()
}

// extractRepoName 从仓库 URL 提取仓库名称
func (sm *StateManager) extractRepoName() string {
	parts := strings.Split(sm.state.RepoURL, "/")
	if len(parts) > 0 {
		name := strings.TrimSuffix(parts[len(parts)-1], ".git")
		return name
	}
	return "Unknown"
}

// StateToMessages 将状态转换为 LLM 消息列表
// 用于在 Agent 之间传递上下文
func (sm *StateManager) StateToMessages() []*schema.Message {
	messages := make([]*schema.Message, 0)

	// 系统消息：当前状态概述
	systemContent := fmt.Sprintf(`当前 Workflow 状态：
- 仓库: %s
- 类型: %s
- 技术栈: %v
- 大纲章节数: %d
- 已生成小节数: %d`,
		sm.state.RepoURL,
		sm.state.RepoType,
		sm.state.TechStack,
		len(sm.state.Outline),
		len(sm.state.SectionsContent),
	)

	messages = append(messages, &schema.Message{
		Role:    schema.System,
		Content: systemContent,
	})

	return messages
}

// UpdateFromAnalysis 从分析结果更新状态
func (sm *StateManager) UpdateFromAnalysis(analysis *RepoAnalysis) {
	sm.SetRepoInfo(analysis.RepoType, analysis.TechStack)
	klog.V(6).Infof("[StateManager] 从分析结果更新状态: type=%s, stack=%v", analysis.RepoType, analysis.TechStack)
}

// UpdateFromOutline 从大纲更新状态
func (sm *StateManager) UpdateFromOutline(outline *DocOutline) {
	sm.SetOutline(outline.Chapters)
	klog.V(6).Infof("[StateManager] 从大纲更新状态: chapters=%d", len(outline.Chapters))
}
