package http

import (
	"github.com/opendeepwiki/backend/internal/pkg/skills"
)

// Provider HTTP Provider
type Provider struct {
	client *Client
}

// NewProvider 创建 HTTP Provider
func NewProvider() *Provider {
	return &Provider{
		client: NewClient(),
	}
}

// Type 返回 Provider 类型
func (p *Provider) Type() string {
	return "http"
}

// Create 创建 HTTP Skill
func (p *Provider) Create(config skills.SkillConfig) (skills.Skill, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return NewHTTPSkill(config, p.client), nil
}
