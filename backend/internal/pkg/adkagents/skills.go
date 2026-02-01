package adkagents

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/opendeepwiki/backend/config"
	"k8s.io/klog/v2"
)

var skmOnce sync.Once
var skillMiddleware adk.AgentMiddleware
var skillBackend *skill.LocalBackend

func (m *Manager) GetOrCreateSkillMiddleware(cfg *config.Config) (adk.AgentMiddleware, error) {
	skmOnce.Do(func() {
		//处理Skills
		sb, err := skill.NewLocalBackend(&skill.LocalBackendConfig{
			BaseDir: cfg.Skill.Dir,
		})
		if err != nil {
			klog.Errorf("failed to create skill backend: %v", err)
			return
		}
		skillBackend = sb

		skills, err := skillBackend.List(context.Background())
		if err != nil {
			klog.Errorf("failed to list skills: %v", err)
		}

		klog.V(6).Infof("[Manager] 创建Skill中间件,加载 %d 个技能", len(skills))
		for _, skillDef := range skills {
			klog.V(6).Infof("[Manager] Skill: %s, Description: %s", skillDef.Name, skillDef.Description)
		}

		// 创建 skill middleware，它会自动提供一个 "skill" 工具
		// 不需要为每个 skill 创建单独的工具，middleware 会处理所有 skill 调用
		sm, err := skill.New(context.Background(), &skill.Config{
			Backend:    skillBackend,
			UseChinese: true,
		})
		if err != nil {
			klog.Errorf("failed to create skill middleware: %v", err)
		}
		skillMiddleware = sm
	})
	return skillMiddleware, nil
}

func (m *Manager) skillMiddlewareHaveSkills() bool {
	skills, err := skillBackend.List(context.Background())
	if err != nil {
		klog.Errorf("failed to list skills: %v", err)
	}
	return len(skills) > 0
}
