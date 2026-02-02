package service

import (
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

// InitDefaultTemplates 初始化预置模板数据
func InitDefaultTemplates(db *gorm.DB) error {
	// 检查是否已存在通用模板
	var count int64
	db.Model(&model.DocumentTemplate{}).Where("`key` = ?", "general").Count(&count)
	if count > 0 {
		// 已存在，跳过初始化
		return nil
	}

	// 使用事务创建预置模板
	return db.Transaction(func(tx *gorm.DB) error {
		// 1. 通用模板
		generalTemplate := &model.DocumentTemplate{
			Key:         "general",
			Name:        "通用模板",
			Description: "适用于大多数项目的通用文档模板",
			IsSystem:    true,
			SortOrder:   1,
			Chapters: []model.TemplateChapter{
				{
					Title:     "项目概览",
					SortOrder: 1,
					Documents: []model.TemplateDocument{
						{
							Title:         "项目概览",
							Filename:      "overview.md",
							ContentPrompt: "生成项目的整体概览，包括项目背景、主要功能、技术栈等基本信息。",
							SortOrder:     1,
						},
					},
				},
				{
					Title:     "架构分析",
					SortOrder: 2,
					Documents: []model.TemplateDocument{
						{
							Title:         "数据架构",
							Filename:      "data_architecture.md",
							ContentPrompt: "分析项目的数据模型、数据库设计、数据流转和数据持久化方案。",
							SortOrder:     1,
						},
						{
							Title:         "业务架构",
							Filename:      "business_architecture.md",
							ContentPrompt: "分析项目的业务模块划分、领域模型、业务逻辑分层。",
							SortOrder:     2,
						},
					},
				},
				{
					Title:     "核心接口",
					SortOrder: 3,
					Documents: []model.TemplateDocument{
						{
							Title:         "核心接口",
							Filename:      "api.md",
							ContentPrompt: "分析项目的核心 API 接口，包括请求参数、响应格式、接口逻辑。",
							SortOrder:     1,
						},
					},
				},
				{
					Title:     "业务流程",
					SortOrder: 4,
					Documents: []model.TemplateDocument{
						{
							Title:         "业务流程",
							Filename:      "business_flow.md",
							ContentPrompt: "分析项目的核心业务逻辑流程，包括流程图和关键业务步骤。",
							SortOrder:     1,
						},
					},
				},
				{
					Title:     "部署配置",
					SortOrder: 5,
					Documents: []model.TemplateDocument{
						{
							Title:         "部署配置",
							Filename:      "deployment.md",
							ContentPrompt: "分析项目的部署方式、环境配置、依赖要求和部署步骤。",
							SortOrder:     1,
						},
					},
				},
			},
		}

		if err := tx.Create(generalTemplate).Error; err != nil {
			return err
		}

		// 2. SpringBoot 模板
		springbootTemplate := &model.DocumentTemplate{
			Key:         "springboot",
			Name:        "SpringBoot 模板",
			Description: "专为 SpringBoot 后端项目设计的文档模板",
			IsSystem:    true,
			SortOrder:   2,
			Chapters: []model.TemplateChapter{
				{
					Title:     "项目概览",
					SortOrder: 1,
					Documents: []model.TemplateDocument{
						{
							Title:         "项目概览",
							Filename:      "overview.md",
							ContentPrompt: "生成 SpringBoot 项目的整体概览，包括项目背景、主要功能、技术栈和依赖版本。",
							SortOrder:     1,
						},
					},
				},
				{
					Title:     "架构分析",
					SortOrder: 2,
					Documents: []model.TemplateDocument{
						{
							Title:         "数据架构",
							Filename:      "data_architecture.md",
							ContentPrompt: "分析 SpringBoot 项目的数据模型、JPA/Hibernate 实体设计、数据库表结构和关系。",
							SortOrder:     1,
						},
						{
							Title:         "业务架构",
							Filename:      "business_architecture.md",
							ContentPrompt: "分析 SpringBoot 项目的业务模块划分、领域模型、分层架构。",
							SortOrder:     2,
						},
					},
				},
				{
					Title:     "核心接口",
					SortOrder: 3,
					Documents: []model.TemplateDocument{
						{
							Title:         "Controller 层分析",
							Filename:      "controller.md",
							ContentPrompt: "分析 SpringBoot 项目的 REST API 接口，包括 URL 映射、请求/响应参数、参数校验。",
							SortOrder:     1,
						},
						{
							Title:         "Service 层分析",
							Filename:      "service.md",
							ContentPrompt: "分析 SpringBoot 项目的 Service 层，包括业务逻辑实现、事务管理、服务间调用。",
							SortOrder:     2,
						},
						{
							Title:         "Repository 层分析",
							Filename:      "repository.md",
							ContentPrompt: "分析 SpringBoot 项目的 Repository 层，包括数据访问接口、自定义查询、分页实现。",
							SortOrder:     3,
						},
					},
				},
				{
					Title:     "业务流程",
					SortOrder: 4,
					Documents: []model.TemplateDocument{
						{
							Title:         "业务流程",
							Filename:      "business_flow.md",
							ContentPrompt: "分析 SpringBoot 项目的核心业务逻辑流程。",
							SortOrder:     1,
						},
					},
				},
				{
					Title:     "部署配置",
					SortOrder: 5,
					Documents: []model.TemplateDocument{
						{
							Title:         "部署配置",
							Filename:      "deployment.md",
							ContentPrompt: "分析 SpringBoot 项目的部署方式、配置文件、环境变量和启动脚本。",
							SortOrder:     1,
						},
					},
				},
			},
		}

		if err := tx.Create(springbootTemplate).Error; err != nil {
			return err
		}

		// 3. 前端模板
		frontendTemplate := &model.DocumentTemplate{
			Key:         "frontend",
			Name:        "前端模板",
			Description: "专为前端项目设计的文档模板",
			IsSystem:    true,
			SortOrder:   3,
			Chapters: []model.TemplateChapter{
				{
					Title:     "项目概览",
					SortOrder: 1,
					Documents: []model.TemplateDocument{
						{
							Title:         "项目概览",
							Filename:      "overview.md",
							ContentPrompt: "生成前端项目的整体概览，包括项目背景、主要功能、技术栈和依赖版本。",
							SortOrder:     1,
						},
					},
				},
				{
					Title:     "架构分析",
					SortOrder: 2,
					Documents: []model.TemplateDocument{
						{
							Title:         "组件分析",
							Filename:      "components.md",
							ContentPrompt: "分析前端项目的组件结构，包括通用组件、业务组件、组件间通信方式。",
							SortOrder:     1,
						},
						{
							Title:         "路由配置",
							Filename:      "routing.md",
							ContentPrompt: "分析前端项目的路由配置，包括路由定义、嵌套路由、路由守卫。",
							SortOrder:     2,
						},
					},
				},
				{
					Title:     "核心接口",
					SortOrder: 3,
					Documents: []model.TemplateDocument{
						{
							Title:         "API 接口",
							Filename:      "api.md",
							ContentPrompt: "分析前端项目调用的 API 接口，包括请求封装、响应处理、错误处理。",
							SortOrder:     1,
						},
					},
				},
				{
					Title:     "状态管理",
					SortOrder: 4,
					Documents: []model.TemplateDocument{
						{
							Title:         "状态管理",
							Filename:      "state_management.md",
							ContentPrompt: "分析前端项目的状态管理方案，包括全局状态、局部状态、状态流转。",
							SortOrder:     1,
						},
					},
				},
				{
					Title:     "部署配置",
					SortOrder: 5,
					Documents: []model.TemplateDocument{
						{
							Title:         "部署配置",
							Filename:      "deployment.md",
							ContentPrompt: "分析前端项目的构建配置、打包方式、部署流程和环境变量。",
							SortOrder:     1,
						},
					},
				},
			},
		}

		if err := tx.Create(frontendTemplate).Error; err != nil {
			return err
		}

		return nil
	})
}
