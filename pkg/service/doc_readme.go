package service

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"
)

type docReadmeService struct {
	parent *docService
}

func (s *docService) ReadmeService() *docReadmeService {
	return &docReadmeService{
		parent: s,
	}
}
func (s *docReadmeService) prompt(ctx context.Context) string {
	prompt := `
		/no_thinking
		你是一名专业的代码分析专家，任务是为一个 GitHub 仓库创建 README.md 文档。你的目标是基于提供的目录结构分析仓库内容，并生成一份高质量的 README，突出项目的关键特性，风格应参考 GitHub 上的高级开源项目。

		仓库名称是%s。
		仓库存放路径=%s.这是一个相对路径。请注意在后面读取文件时先拼接相对路径。
		请你根据存放路径，使用 [list_directory] 方法 先读取仓库文件夹根目录结构，再根据目录结构，按需读取仓库的中的必要文件，然后根据文件内容，生成一个README.md文件。

		请遵循以下步骤生成 中文版 README.md 文档：
	
	
1. 核心文件分析

使用 READ_FILE 函数检查以下关键文件，请先拼接仓库存储的相对路径%s：
	•	不要读取'.'dot点开头的文件或目录
	•	不要读取 'node_modules' 、'.git' 等辅助目录
	•	主项目文件（通常位于根目录）
	•	配置文件（如 package.json、setup.py 等）
	•	文档文件（位于根目录或 /docs 目录中）
	•	示例文件或用法演示

2. 分节信息提取
针对 README 的每个部分，从指定文件中提取准确信息：
a. 项目名称 / 描述
	•	读取主文件与配置文件
	•	查找项目描述（如在 package.json、setup.py、或主要实现文件中）
b. 特性
	•	阅读主要实现文件，识别项目的能力与功能
	•	研究代码结构以提炼功能特性
	•	查阅专门的功能文档（如有）
c. 安装说明
	•	阅读安装相关文件（如 package.json、requirements.txt、或安装指南）
	•	提取依赖信息和安装步骤
d. 使用说明
	•	阅读示例文件、文档、或主实现文件
	•	提取展示项目使用方式的代码示例
e. 贡献说明
	•	阅读 CONTRIBUTING.md 或类似的贡献指南文件
f. 许可证
	•	如果仓库中存在 LICENSE 文件，请读取该文件内容

3. README 结构规范
请按照以下结构组织 README.md：
a. 项目名称与描述
	•	简洁明了的项目名称
	•	项目的目的与价值主张简介
	•	如适用，包含徽章或状态指示器
b. 特性
	•	项目的关键能力以项目符号列出
	•	对主要功能进行简要解释
	•	项目的独特性与优势
c. 安装
	•	安装步骤详解
	•	依赖项与环境要求
	•	如适用，提供平台特定说明
d. 使用
	•	基本用法示例与代码片段
	•	常见使用场景
	•	如有，提供 API 概览
e. 贡献指南
	•	面向贡献者的指南
	•	开发环境配置说明
	•	Pull Request 提交流程
f. 许可证（仅当存在 LICENSE 文件时）
	•	简要说明许可证类型与影响

重要准则：
	•	所有 README 中的信息 必须通过读取实际文件内容（使用 READ_FILE ，相对路径）获取
	•	不得对项目内容做任何未经验证的假设
	•	使用 Markdown 格式优化可读性（如标题、代码块、列表等）
	•	专注于创建一份专业、有吸引力的 README，突出项目优势
	•	对话记录中，如有 已归纳 的信息，可以合理使用。
	•	确保 README 结构清晰、内容准确
	•	请使用中文书写文档。

请将最终的 README.md 内容，使用<readme></readme>进行包裹。调用 [write_file] 函数写入 %s 目录下的 README.md 文件。
请在结束对话前，务必检查将最终的 README.md 内容，使用<readme></readme>进行包裹。调用 [write_file] 函数写入 %s 目录下的 README.md 文件。		
`

	folder, _ := s.parent.GetRuntimeFolder()
	path, _ := s.parent.RepoService().GetRepoPath(ctx)
	repName := s.parent.RepoService().GetRepoName(ctx)
	return fmt.Sprintf(prompt, repName, path, path, folder, folder)
}

func (s *docReadmeService) finalCheck(ctx context.Context) string {
	prompt := `
		/no_thinking
		仓库信息:
		仓库名称是%s。
		仓库存放路径=%s.这是一个相对路径。请注意在后面读取文件时先拼接相对路径。
		Readme.md 存为路径[%s]/readme.md
	
		任务要求：
		在对话结束前，请确认已经在目录[%s]下生成Readme.md.
		请使用[READ_FILE]尝试读取确认改文件已经生成。

		确认结果处理：
		1. 如果没有生成，请利用历史对话信息重新生成。
		2. 如果已经生成，并且历史对话信息中含有归纳信息，可以利用归纳信息，更新Readme.md文档
		3. 如果已经生成，并且不需要更新，请输出 <确认结束> , 结束对话。
		
`

	folder, _ := s.parent.GetRuntimeFolder()
	repName := s.parent.RepoService().GetRepoName(ctx)
	return fmt.Sprintf(prompt, repName, folder, folder, folder)
}
func (s *docReadmeService) Generate(ctx context.Context) error {
	// 计时
	start := time.Now()
	defer func() {
		klog.V(6).Infof("生成README.md文件耗时: %0.2f 秒", time.Since(start).Seconds())
	}()

	if err := s.parent.MustHaveAnalysisInstance(); err != nil {
		return err
	}
	reader, err := s.parent.chat(ctx, s.prompt(ctx), "", s.finalCheck(ctx))
	if err != nil {
		return err

	}
	all, err := s.parent.readAndWrite(ctx, reader)
	if err != nil {
		return err
	}
	klog.V(6).Infof("生成README.md文件成功: \n%s\n\n", all)
	return nil
}
