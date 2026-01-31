// Package einodoc 基于 Eino 框架的仓库文档解析服务
// 使用 Eino ADK 模式：Chain + Graph + State
package einodoc

import (
	"fmt"
	"sync"

	"k8s.io/klog/v2"
)

// RepoDocState Workflow 共享状态
// 存储整个仓库解析流程的上下文和中间结果
type RepoDocState struct {
	mu sync.RWMutex

	// 输入参数
	RepoURL   string `json:"repo_url"`   // 仓库 Git URL
	LocalPath string `json:"local_path"` // 本地克隆路径

	// 仓库分析结果
	RepoTree  string   `json:"repo_tree"`  // 仓库目录结构
	RepoType  string   `json:"repo_type"`  // 仓库类型: go / java / python / frontend / mixed
	TechStack []string `json:"tech_stack"` // 技术栈: gin / spring / react ...

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
	Title    string    `json:"title"`    // 章节标题
	Sections []Section `json:"sections"` // 章节下的小节列表
}

// Section 小节定义
type Section struct {
	Title string   `json:"title"` // 小节标题
	Hints []string `json:"hints"` // 写作提示/关注点
}

// NewRepoDocState 创建新的状态对象
// repoURL: 仓库 Git 地址
// localPath: 本地存储路径
func NewRepoDocState(repoURL, localPath string) *RepoDocState {
	klog.V(6).Infof("[RepoDocState] 创建新的状态对象: repoURL=%s, localPath=%s", repoURL, localPath)
	return &RepoDocState{
		LocalPath:       localPath,
		SectionsContent: make(map[string]string),
		Outline:         make([]Chapter, 0),
		TechStack:       make([]string, 0),
	}
}

// SetRepoTree 设置仓库目录结构（线程安全）
func (s *RepoDocState) SetRepoTree(tree string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	klog.V(6).Infof("[RepoDocState] 设置仓库目录结构: length=%d", len(tree))
	s.RepoTree = tree
}

// SetRepoInfo 设置仓库基本信息（线程安全）
// repoType: 仓库类型 (go/java/python/frontend/mixed)
// techStack: 技术栈列表
func (s *RepoDocState) SetRepoInfo(repoType string, techStack []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	klog.V(6).Infof("[RepoDocState] 设置仓库信息: repoType=%s, techStack=%v", repoType, techStack)
	s.RepoType = repoType
	s.TechStack = techStack
}

// SetOutline 设置大纲（线程安全）
// outline: 章节大纲列表
func (s *RepoDocState) SetOutline(outline []Chapter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	klog.V(6).Infof("[RepoDocState] 设置大纲: chapters=%d", len(outline))
	for i, ch := range outline {
		klog.V(6).Infof("[RepoDocState]   Chapter[%d]: %s, sections=%d", i, ch.Title, len(ch.Sections))
	}
	s.Outline = outline
}

// GetCurrentSection 获取当前正在处理的小节（线程安全）
// 返回: chapterIdx, sectionIdx, chapter指针, section指针, 是否存在
func (s *RepoDocState) GetCurrentSection() (chapterIdx, sectionIdx int, chapter *Chapter, section *Section, exists bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.CurrentChapterIdx >= len(s.Outline) {
		klog.V(6).Infof("[RepoDocState] 获取当前小节: 所有章节已处理完成")
		return s.CurrentChapterIdx, s.CurrentSectionIdx, nil, nil, false
	}

	ch := &s.Outline[s.CurrentChapterIdx]
	if s.CurrentSectionIdx >= len(ch.Sections) {
		klog.V(6).Infof("[RepoDocState] 获取当前小节: chapterIdx=%d, 当前章节的小节已处理完", s.CurrentChapterIdx)
		return s.CurrentChapterIdx, s.CurrentSectionIdx, ch, nil, false
	}

	klog.V(6).Infof("[RepoDocState] 获取当前小节: chapterIdx=%d, sectionIdx=%d, chapter=%s, section=%s",
		s.CurrentChapterIdx, s.CurrentSectionIdx, ch.Title, ch.Sections[s.CurrentSectionIdx].Title)
	return s.CurrentChapterIdx, s.CurrentSectionIdx, ch, &ch.Sections[s.CurrentSectionIdx], true
}

// SetSectionContent 设置小节内容（线程安全）
// chapterIdx: 章节索引
// sectionIdx: 小节索引
// content: 内容文本
func (s *RepoDocState) SetSectionContent(chapterIdx, sectionIdx int, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := sectionKey(chapterIdx, sectionIdx)
	klog.V(6).Infof("[RepoDocState] 设置小节内容: key=%s, contentLength=%d", key, len(content))
	s.SectionsContent[key] = content
}

// GetSectionContent 获取小节内容（线程安全）
// chapterIdx: 章节索引
// sectionIdx: 小节索引
// 返回: 内容文本
func (s *RepoDocState) GetSectionContent(chapterIdx, sectionIdx int) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := sectionKey(chapterIdx, sectionIdx)
	content := s.SectionsContent[key]
	klog.V(6).Infof("[RepoDocState] 获取小节内容: key=%s, exists=%v, length=%d", key, content != "", len(content))
	return content
}

// MoveToNextSection 移动到下一小节（线程安全）
// 返回: 是否还有更多小节需要处理
func (s *RepoDocState) MoveToNextSection() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.CurrentChapterIdx >= len(s.Outline) {
		klog.V(6).Infof("[RepoDocState] 移动到下一个小节: 所有章节已处理完成")
		return false
	}

	ch := s.Outline[s.CurrentChapterIdx]
	prevCh := s.CurrentChapterIdx
	prevSec := s.CurrentSectionIdx
	s.CurrentSectionIdx++

	if s.CurrentSectionIdx >= len(ch.Sections) {
		// 当前章节的 section 处理完了，移动到下一章
		s.CurrentChapterIdx++
		s.CurrentSectionIdx = 0
		klog.V(6).Infof("[RepoDocState] 移动到下一个小节: 章节[%d]完成，进入章节[%d]", prevCh, s.CurrentChapterIdx)
	} else {
		klog.V(6).Infof("[RepoDocState] 移动到下一个小节: chapter[%d] section[%d] -> section[%d]",
			prevCh, prevSec, s.CurrentSectionIdx)
	}

	return s.CurrentChapterIdx < len(s.Outline)
}

// IsComplete 检查是否处理完成（线程安全）
// 返回: 是否所有小节都已处理完成
func (s *RepoDocState) IsComplete() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	complete := s.CurrentChapterIdx >= len(s.Outline)
	klog.V(6).Infof("[RepoDocState] 检查是否完成: currentChapter=%d, totalChapters=%d, complete=%v",
		s.CurrentChapterIdx, len(s.Outline), complete)
	return complete
}

// SetFinalDocument 设置最终文档（线程安全）
// doc: 最终生成的文档内容
func (s *RepoDocState) SetFinalDocument(doc string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	klog.V(6).Infof("[RepoDocState] 设置最终文档: length=%d", len(doc))
	s.FinalDocument = doc
}

// GetFinalDocument 获取最终文档（线程安全）
// 返回: 最终生成的文档内容
func (s *RepoDocState) GetFinalDocument() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	klog.V(6).Infof("[RepoDocState] 获取最终文档: length=%d", len(s.FinalDocument))
	return s.FinalDocument
}

// sectionKey 生成小节内容的 key
// 格式: "chapterIdx/sectionIdx"
func sectionKey(chapterIdx, sectionIdx int) string {
	return fmt.Sprintf("%d/%d", chapterIdx, sectionIdx)
}

// RepoDocResult 仓库文档解析结果
// 作为 Service 的返回结果
type RepoDocResult struct {
	RepoURL         string            `json:"repo_url"`                   // 仓库 Git URL
	LocalPath       string            `json:"local_path"`                 // 本地克隆路径
	RepoType        string            `json:"repo_type"`                  // 仓库类型
	TechStack       []string          `json:"tech_stack"`                 // 技术栈
	Outline         []Chapter         `json:"outline"`                    // 文档大纲
	Document        string            `json:"document"`                   // 最终文档内容
	SectionsCount   int               `json:"sections_count"`             // 生成的小节数量
	Completed       bool              `json:"completed"`                  // 是否完成
	SectionsContent map[string]string `json:"sections_content,omitempty"` // 各小节内容
}
