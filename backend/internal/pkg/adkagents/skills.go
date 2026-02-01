package adkagents

import (
	"context"
	"os"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"k8s.io/klog/v2"
)

var skmOnce sync.Once
var skillMiddleware adk.AgentMiddleware

func (m *Manager) GetOrCreateSkillMiddleware() (adk.AgentMiddleware, error) {
	skmOnce.Do(func() {
		//TODO 增加，改成Config
		skillPath := os.Getenv("SKILL_PATH")
		if skillPath == "" {
			skillPath = "../skills"
		}
		//处理Skills
		skillBackend, err := skill.NewLocalBackend(&skill.LocalBackendConfig{
			BaseDir: skillPath,
		})

		skills, err := skillBackend.List(context.Background())
		if err != nil {
			klog.Errorf("failed to list skills: %w", err)
		}
		for _, skill := range skills {
			klog.V(6).Infof("[Manager] Skill: %s, Description: %s", skill.Name, skill.Description)
		}
		klog.V(6).Infof("[Manager] 创建Skill中间件,加载 %d 个技能", len(skills))
		sm, err := skill.New(context.Background(), &skill.Config{
			Backend:    skillBackend,
			UseChinese: true,
		})
		if err != nil {
			klog.Errorf("failed to create skill middleware: %w", err)
		}
		skillMiddleware = sm
	})
	return skillMiddleware, nil
}
