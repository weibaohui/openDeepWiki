package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAgentVersion_TableName(t *testing.T) {
	v := &AgentVersion{}
	assert.Equal(t, "agent_versions", v.TableName())
}

func TestAgentVersion_GormTableName(t *testing.T) {
	v := &AgentVersion{}
	assert.Equal(t, "agent_versions", v.GormTableName())
}

func TestAgentVersion_Fields(t *testing.T) {
	now := time.Now()
	restoreFrom := 1
	v := &AgentVersion{
		ID:                1,
		FileName:          "test.yaml",
		Content:           "name: test",
		Version:           1,
		SavedAt:           now,
		Source:            "web",
		RestoreFromVersion: &restoreFrom,
		CreatedAt:         now,
	}

	assert.Equal(t, uint(1), v.ID)
	assert.Equal(t, "test.yaml", v.FileName)
	assert.Equal(t, "name: test", v.Content)
	assert.Equal(t, 1, v.Version)
	assert.Equal(t, now, v.SavedAt)
	assert.Equal(t, "web", v.Source)
	assert.NotNil(t, v.RestoreFromVersion)
	assert.Equal(t, 1, *v.RestoreFromVersion)
	assert.Equal(t, now, v.CreatedAt)
}
