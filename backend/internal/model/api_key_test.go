package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIKey_MaskAPIKey 测试 API Key 脱敏功能
func TestAPIKey_MaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "正常长度key",
			apiKey:   "sk-1234567890abcdef",
			expected: "sk-***cdef",
		},
		{
			name:     "短key（小于等于7位）",
			apiKey:   "sk-123",
			expected: "***",
		},
		{
			name:     "正好7位",
			apiKey:   "1234567",
			expected: "***",
		},
		{
			name:     "长key",
			apiKey:   "abcdefghijklmnopqrstuvwxyz",
			expected: "abc***wxyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &APIKey{APIKey: tt.apiKey}
			assert.Equal(t, tt.expected, key.MaskAPIKey())
		})
	}
}

// TestAPIKey_IsAvailable 测试 API Key 可用性检查
func TestAPIKey_IsAvailable(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)
	earlier := now.Add(-time.Hour)

	tests := []struct {
		name               string
		apiKey             *APIKey
		expectedAvailable  bool
	}{
		{
			name: "enabled状态且无限速重置时间",
			apiKey: &APIKey{
				Status:           "enabled",
				RateLimitResetAt: nil,
			},
			expectedAvailable: true,
		},
		{
			name: "enabled状态且限速已过期",
			apiKey: &APIKey{
				Status:           "enabled",
				RateLimitResetAt: &earlier,
			},
			expectedAvailable: true,
		},
		{
			name: "disabled状态",
			apiKey: &APIKey{
				Status:           "disabled",
				RateLimitResetAt: nil,
			},
			expectedAvailable: false,
		},
		{
			name: "unavailable状态",
			apiKey: &APIKey{
				Status:           "unavailable",
				RateLimitResetAt: nil,
			},
			expectedAvailable: false,
		},
		{
			name: "enabled状态但限速未过期",
			apiKey: &APIKey{
				Status:           "enabled",
				RateLimitResetAt: &later,
			},
			expectedAvailable: false,
		},
		{
			name: "disabled状态但有限速时间",
			apiKey: &APIKey{
				Status:           "disabled",
				RateLimitResetAt: &later,
			},
			expectedAvailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedAvailable, tt.apiKey.IsAvailable())
		})
	}
}

// TestAPIKey_BeforeUpdate 测试 GORM 钩子
func TestAPIKey_BeforeUpdate(t *testing.T) {
	key := &APIKey{
		Name:     "test",
		UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// 调用钩子前的UpdatedAt
	oldUpdatedAt := key.UpdatedAt

	// 模拟调用 BeforeUpdate
	err := key.BeforeUpdate(nil)
	require.NoError(t, err)

	// UpdatedAt 应该被更新到当前时间附近
	assert.True(t, key.UpdatedAt.After(oldUpdatedAt))
	assert.True(t, key.UpdatedAt.Before(time.Now().Add(time.Second)))
}

// TestAPIKey_TableName 测试表名
func TestAPIKey_TableName(t *testing.T) {
	key := APIKey{}
	assert.Equal(t, "api_keys", key.TableName())
}
