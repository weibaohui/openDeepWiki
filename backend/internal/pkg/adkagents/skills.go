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
		if err != nil {
			klog.Errorf("failed to create skill backend: %v", err)
			return
		}

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
