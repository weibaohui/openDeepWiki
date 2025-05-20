package models

import (
	"time"
)

type Config struct {
	ID                   uint      `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	ProductName          string    `json:"product_name,omitempty"` // 产品名称
	ApiKey               string    `json:"api_key,omitempty"`
	ApiURL               string    `json:"api_url,omitempty"`
	ApiModel             string    `json:"api_model,omitempty"`
	LoginType            string    `json:"login_type,omitempty"`
	JwtTokenSecret       string    `json:"jwt_token_secret,omitempty"`
	AnySelect            bool      `gorm:"default:true" json:"any_select"`
	PrintConfig          bool      `json:"print_config"`
	Temperature          float32   `json:"temperature"`                                        // 模型温度
	TopP                 float32   `json:"top_p"`                                              //  模型topP参数
	MaxHistory           int32     `json:"max_history"`                                        //  模型对话上下文历史记录数
	MaxIterations        int32     `json:"max_iterations"`                                     //  模型自动化对话的轮数限制
	ResourceCacheTimeout int       `gorm:"default:60" json:"resource_cache_timeout,omitempty"` // 资源缓存时间（秒）
	CreatedAt            time.Time `json:"created_at,omitempty"`                               // Automatically managed by GORM for creation time
	UpdatedAt            time.Time `json:"updated_at,omitempty"`                               // Automatically managed by GORM for update time
}
