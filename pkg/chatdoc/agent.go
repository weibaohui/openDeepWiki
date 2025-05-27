package chatdoc

import (
	"github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
)

type Agent interface {
	HandleTask(task chatdoc.Task) (chatdoc.Task, error)
	SetConfig(cfg chatdoc.RoleConfig)
}

var RegisteredAgents = map[string]Agent{}

func RegisterAgent(name string, agent Agent) {
	RegisteredAgents[name] = agent
}

func RegisterAgentWithConfig(name string, agent Agent, cfg chatdoc.RoleConfig) {
	agent.SetConfig(cfg)
	RegisteredAgents[name] = agent
}
