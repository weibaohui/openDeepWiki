package chatdoc

import (
	"context"
	"os"

	"github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
	"gopkg.in/yaml.v3"
)

type ChatDocService struct {
	Roles         []chatdoc.Role
	Collaboration []chatdoc.Collaboration
}

func NewChatDocService() *ChatDocService {
	roles := loadRoles()
	collab := loadCollaboration()
	return &ChatDocService{
		Roles:         roles,
		Collaboration: collab,
	}
}

func loadRoles() []chatdoc.Role {
	f, err := os.ReadFile("data/chatdoc_roles.yaml")
	if err != nil {
		return nil
	}
	var data struct {
		Roles []chatdoc.Role `yaml:"roles"`
	}
	_ = yaml.Unmarshal(f, &data)
	return data.Roles
}

func loadCollaboration() []chatdoc.Collaboration {
	f, err := os.ReadFile("data/chatdoc_collaboration.yaml")
	if err != nil {
		return nil
	}
	var data struct {
		Collaboration []chatdoc.Collaboration `yaml:"collaboration"`
	}
	_ = yaml.Unmarshal(f, &data)
	return data.Collaboration
}

// 示例：发起协作会话
func (svc *ChatDocService) StartSession(ctx context.Context) *chatdoc.ChatDocSession {
	return &chatdoc.ChatDocSession{
		ID:           "session-id",
		CurrentStage: "init",
		History:      []string{},
		Roles:        svc.Roles,
	}
}
