package skills

import (
	"strings"
)

// Matcher Skill 匹配器
type Matcher struct {
	registry Registry
}

// NewMatcher 创建匹配器
func NewMatcher(registry Registry) *Matcher {
	return &Matcher{registry: registry}
}

// Match 根据任务匹配 Skills
func (m *Matcher) Match(task Task) ([]*Match, error) {
	enabled := m.registry.ListEnabled()
	matches := make([]*Match, 0)

	for _, skill := range enabled {
		score, reason := m.calculateScore(skill, task)
		if score > 0 {
			matches = append(matches, &Match{
				Skill:  skill,
				Score:  score,
				Reason: reason,
			})
		}
	}

	// 按分数排序
	sortMatches(matches)
	return matches, nil
}

// MatchByDescription 根据描述匹配（简单版本）
func (m *Matcher) MatchByDescription(description string) ([]*Match, error) {
	task := Task{Description: description}
	return m.Match(task)
}

// MatchByType 根据类型匹配
func (m *Matcher) MatchByType(taskType string) ([]*Match, error) {
	task := Task{Type: taskType}
	return m.Match(task)
}

// MatchForRepo 根据仓库类型匹配
func (m *Matcher) MatchForRepo(repoType string, description string) ([]*Match, error) {
	task := Task{
		RepoType:    repoType,
		Description: description,
	}
	return m.Match(task)
}

// calculateScore 计算匹配分数
func (m *Matcher) calculateScore(skill *Skill, task Task) (float64, string) {
	score := 0.0
	reasons := make([]string, 0)

	desc := strings.ToLower(skill.Description)
	taskDesc := strings.ToLower(task.Description)
	taskType := strings.ToLower(task.Type)
	repoType := strings.ToLower(task.RepoType)

	// 1. 描述关键词匹配（最高权重 0.5）
	keywords := extractKeywords(taskDesc)
	keywordMatches := 0
	for _, kw := range keywords {
		if strings.Contains(desc, kw) {
			keywordMatches++
		}
	}
	if len(keywords) > 0 {
		matchRatio := float64(keywordMatches) / float64(len(keywords))
		score += matchRatio * 0.5
		if matchRatio > 0.3 {
			reasons = append(reasons, "keyword match")
		}
	}

	// 2. 任务类型匹配（权重 0.3）
	if taskType != "" {
		// 直接包含
		if strings.Contains(desc, taskType) {
			score += 0.3
			reasons = append(reasons, "task type match")
		} else {
			// 同义词匹配
			synonyms := getTaskTypeSynonyms(taskType)
			for _, syn := range synonyms {
				if strings.Contains(desc, syn) {
					score += 0.25
					reasons = append(reasons, "task type synonym match")
					break
				}
			}
		}
	}

	// 3. 仓库类型匹配（权重 0.2）
	if repoType != "" && strings.Contains(desc, repoType) {
		score += 0.2
		reasons = append(reasons, "repo type match")
	}

	// 4. 标签匹配（权重 0.1 每个）
	for _, tag := range task.Tags {
		tagLower := strings.ToLower(tag)
		if strings.Contains(desc, tagLower) {
			score += 0.1
			reasons = append(reasons, "tag match")
			break // 只计算一次
		}
	}

	if score == 0 {
		return 0, ""
	}

	// 限制最大分数为 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score, strings.Join(reasons, ", ")
}

// extractKeywords 提取关键词
func extractKeywords(text string) []string {
	// 停用词表
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"is": true, "are": true, "was": true, "were": true,
		"this": true, "that": true, "these": true, "those": true,
		"to": true, "of": true, "in": true, "for": true, "on": true,
		"with": true, "by": true, "at": true, "from": true,
		"it": true, "its": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "could": true, "should": true,
		"can": true, "may": true, "might": true, "must": true,
		"about": true, "into": true, "through": true, "during": true,
		"before": true, "after": true, "above": true, "below": true,
		"between": true, "under": true, "again": true, "further": true,
		"then": true, "once": true, "here": true, "there": true,
		"when": true, "where": true, "why": true, "how": true,
		"all": true, "each": true, "few": true, "more": true, "most": true,
		"other": true, "some": true, "such": true, "no": true, "nor": true,
		"not": true, "only": true, "own": true, "same": true, "so": true,
		"than": true, "too": true, "very": true, "just": true, "now": true,
		"also": true, "get": true, "use": true, "using": true, "used": true,
		"make": true, "made": true, "see": true, "seen": true, "come": true,
		"came": true, "know": true, "knew": true, "take": true, "took": true,
		"think": true, "thought": true, "say": true, "said": true, "go": true, "went": true,
		"help": true, "helps": true, "helped": true, "show": true, "shows": true, "showed": true,
		"he": true, "him": true, "his": true, "she": true, "her": true, "hers": true,
		"they": true, "them": true, "their": true, "theirs": true,
		"we": true, "us": true, "our": true, "ours": true,
		"you": true, "your": true, "yours": true,
		"i": true, "me": true, "my": true, "mine": true,
	}

	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
			r == ',' || r == '.' || r == '!' || r == '?' || r == ';' ||
			r == ':' || r == '(' || r == ')' || r == '[' || r == ']' ||
			r == '{' || r == '}' || r == '"' || r == '\'' || r == '`' ||
			r == '/' || r == '\\' || r == '|' || r == '&' || r == '*' ||
			r == '+' || r == '=' || r == '<' || r == '>' || r == '@' ||
			r == '#' || r == '$' || r == '%' || r == '^'
	})

	keywords := make([]string, 0)
	seen := make(map[string]bool)

	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if len(word) > 2 && !stopWords[word] && !seen[word] {
			keywords = append(keywords, word)
			seen[word] = true
		}
	}

	return keywords
}

// getTaskTypeSynonyms 获取任务类型同义词
func getTaskTypeSynonyms(taskType string) []string {
	synonyms := map[string][]string{
		"overview":     {"introduction", "summary", "getting started", "quick start"},
		"architecture": {"structure", "design", "pattern", "organization", "layout"},
		"api":          {"interface", "endpoint", "method", "function", "rpc", "rest", "grpc"},
		"business-flow": {"workflow", "process", "logic", "flow", "business logic"},
		"deployment":   {"deploy", "install", "setup", "configure", "config", "production"},
		"database":     {"db", "sql", "schema", "model", "entity", "storage", "data"},
		"frontend":     {"ui", "client", "web", "react", "vue", "angular"},
		"backend":      {"server", "api", "service", "handler"},
	}

	if syns, ok := synonyms[taskType]; ok {
		return syns
	}
	return nil
}

// sortMatches 排序匹配结果（按分数降序）
func sortMatches(matches []*Match) {
	// 简单冒泡排序（数据量小）
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
}
