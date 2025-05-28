package chatdoc

import (
	"context"

	"github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
)

type Agent interface {
	HandleTask(ctx context.Context, s *chatDocService, task chatdoc.Task) (chatdoc.Task, error)
	SetConfig(cfg chatdoc.RoleConfig)
}

var RegisteredAgents = map[string]Agent{}

func RegisterAgentWithConfig(name string, agent Agent, cfg chatdoc.RoleConfig) {
	agent.SetConfig(cfg)
	RegisteredAgents[name] = agent
}
