// Package einodoc 基于 Eino 框架的仓库文档解析服务
// 使用 Eino ADK 模式：Chain + Graph + State
package einodoc

import (
	"fmt"
	"sync"
)

// RepoDocState Workflow 共享状态
type RepoDocState struct {
	mu sync.RWMutex

	// 输入参数
	RepoURL   string `json:"repo_url"`
	LocalPath string `json:"local_path"`

	// 仓库分析结果
	RepoType  string   `json:"repo_type"`  // go / java / python / frontend / mixed
	TechStack []string `json:"tech_stack"` // gin / spring / react ...

	// 大纲结构
	Outline []Chapter `json:"outline"`

	// 当前处理进度
	CurrentChapterIdx int `json:"current_chapter_idx"`
	CurrentSectionIdx int `json:"current_section_idx"`

	// 输出内容
	SectionsContent map[string]string `json:"sections_content"` // key: chapter/section
	FinalDocument   string            `json:"final_document"`
}

// Chapter 章节定义
type Chapter struct {
	Title    string    `json:"title"`
	Sections []Section `json:"sections"`
}

// Section 小节定义
type Section struct {
	Title string   `json:"title"`
	Hints []string `json:"hints"` // 写作提示
}

// NewRepoDocState 创建新的状态对象
func NewRepoDocState(repoURL, localPath string) *RepoDocState {
	return &RepoDocState{
		RepoURL:         repoURL,
		LocalPath:       localPath,
		SectionsContent: make(map[string]string),
		Outline:         make([]Chapter, 0),
		TechStack:       make([]string, 0),
	}
}

// SetRepoInfo 设置仓库基本信息（线程安全）
func (s *RepoDocState) SetRepoInfo(repoType string, techStack []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RepoType = repoType
	s.TechStack = techStack
}

// SetOutline 设置大纲（线程安全）
func (s *RepoDocState) SetOutline(outline []Chapter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Outline = outline
}

// GetCurrentSection 获取当前正在处理的小节（线程安全）
func (s *RepoDocState) GetCurrentSection() (chapterIdx, sectionIdx int, chapter *Chapter, section *Section, exists bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.CurrentChapterIdx >= len(s.Outline) {
		return s.CurrentChapterIdx, s.CurrentSectionIdx, nil, nil, false
	}

	ch := &s.Outline[s.CurrentChapterIdx]
	if s.CurrentSectionIdx >= len(ch.Sections) {
		return s.CurrentChapterIdx, s.CurrentSectionIdx, ch, nil, false
	}

	return s.CurrentChapterIdx, s.CurrentSectionIdx, ch, &ch.Sections[s.CurrentSectionIdx], true
}

// SetSectionContent 设置小节内容（线程安全）
func (s *RepoDocState) SetSectionContent(chapterIdx, sectionIdx int, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := sectionKey(chapterIdx, sectionIdx)
	s.SectionsContent[key] = content
}

// GetSectionContent 获取小节内容（线程安全）
func (s *RepoDocState) GetSectionContent(chapterIdx, sectionIdx int) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := sectionKey(chapterIdx, sectionIdx)
	return s.SectionsContent[key]
}

// MoveToNextSection 移动到下一小节（线程安全）
func (s *RepoDocState) MoveToNextSection() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.CurrentChapterIdx >= len(s.Outline) {
		return false
	}

	ch := s.Outline[s.CurrentChapterIdx]
	s.CurrentSectionIdx++

	if s.CurrentSectionIdx >= len(ch.Sections) {
		// 当前章节的 section 处理完了，移动到下一章
		s.CurrentChapterIdx++
		s.CurrentSectionIdx = 0
	}

	return s.CurrentChapterIdx < len(s.Outline)
}

// IsComplete 检查是否处理完成（线程安全）
func (s *RepoDocState) IsComplete() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.CurrentChapterIdx >= len(s.Outline)
}

// SetFinalDocument 设置最终文档（线程安全）
func (s *RepoDocState) SetFinalDocument(doc string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FinalDocument = doc
}

// GetFinalDocument 获取最终文档（线程安全）
func (s *RepoDocState) GetFinalDocument() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.FinalDocument
}

// sectionKey 生成小节内容的 key
func sectionKey(chapterIdx, sectionIdx int) string {
	return fmt.Sprintf("%d/%d", chapterIdx, sectionIdx)
}

// RepoDocResult 仓库文档解析结果
type RepoDocResult struct {
	RepoURL         string            `json:"repo_url"`
	LocalPath       string            `json:"local_path"`
	RepoType        string            `json:"repo_type"`
	TechStack       []string          `json:"tech_stack"`
	Outline         []Chapter         `json:"outline"`
	Document        string            `json:"document"`
	SectionsCount   int               `json:"sections_count"`
	Completed       bool              `json:"completed"`
	SectionsContent map[string]string `json:"sections_content,omitempty"`
}
