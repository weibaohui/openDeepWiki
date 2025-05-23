package service

import (
	"context"
	"fmt"

	"github.com/weibaohui/openDeepWiki/pkg/models"
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
		你是一个文档生成助手，你需要根据以下信息生成一个README.md文件。
		仓库存放路径在%s.请注意使用相对路径。
		仓库名称是%s。
		请你根据存放路径，先读取仓库文件夹目录结构，再根据目录结构，读取仓库的中的必要文件，然后根据文件内容，生成一个README.md文件。
		请你读取关键的代码目录结构下的文件，包括：
		1. 代码文件
		2. 配置文件
		3. 脚本文件 抽取关键信息，作为编写readme文档的依据。
		原仓库中的Readme文档，只能作为参考。	
		请你生成README.md文件，包含以下信息：
		1. 仓库名称
		2. 仓库描述
		3. 仓库的使用方法
		4. 仓库的依赖
		5. 仓库的安装方法
		6. 仓库的配置方法
		7. 仓库的使用示例
		8. 仓库的注意事项
		9. 仓库的贡献者
		10. 仓库的许可证
		11. 仓库的版本号
		12. 仓库的更新日志
		13. 仓库的问题反馈
		14. 仓库的贡献指南
		等相关信息。
		请务必使用<finalResult></finalResult>包裹最终结果。
		`
	prompt = `
		You are a professional code analysis expert tasked with creating a README.md document for a GitHub repository. Your goal is to analyze the content of the repository based on the provided catalogue structure and generate a high-quality README that highlights the project's key features and follows the style of advanced open-source projects on GitHub.

仓库存放路径在%s.请注意使用相对路径。
		仓库名称是%s。
		请你根据存放路径，先读取仓库文件夹目录结构，

 

To collect information about the files in the repository, you can use the READ_FILE function. This function accepts the file path as a parameter and returns the content of the file. Use this function to read the contents of specific files mentioned in the directory.

Follow these steps to generate the README:

1. Essential File Analysis
   - Examine key files by using the READ_FILE function on:
     - Main project file (typically in root directory)
     - Configuration files (package.json, setup.py, etc.)
     - Documentation files (in root or /docs directory)
     - Example files or usage demonstrations

2. Section-by-Section Information Gathering
   For each README section, READ specific files to extract accurate information:

   a. Project Title/Description
      - READ main files and configuration files
      - Look for project descriptions in package.json, setup.py, or main implementation files

   b. Features
      - READ implementation files to identify capabilities and functionality
      - Examine code structure to determine feature sets
      - Look for feature documentation in specialized files

   c. Installation
      - READ setup files like package.json, requirements.txt, or installation guides
      - Extract dependency information and setup requirements

   d. Usage
      - READ example files, documentation, or main implementation files
      - Extract code examples showing how to use the project

   e. Contributing
      - READ CONTRIBUTING.md or similar contribution guidelines

   f. License
      - READ the LICENSE file if it exists in the repository

3. README Structure
   Structure your README.md with the following sections:

   a. Project Title and Description
      - Clear, concise project name
      - Brief overview of purpose and value proposition
      - Any badges or status indicators if applicable

   b. Features
      - Bulleted list of key capabilities
      - Brief explanations of main functionality
      - What makes this project unique or valuable

   c. Installation
      - Step-by-step instructions
      - Dependencies and requirements
      - Platform-specific notes if applicable

   d. Usage
      - Basic examples with code snippets
      - Common use cases
      - API overview if applicable

   e. Contributing
      - Guidelines for contributors
      - Development setup
      - Pull request process

   f. License (ONLY if a LICENSE file exists)
      - Brief description of the license type and implications

Important Guidelines:
- ALL information in the README MUST be obtained by READING actual file contents using the READ_FILE function
- Do NOT make assumptions about the project without verifying through file contents
- Use Markdown formatting to enhance readability (headings, code blocks, lists, etc.)
- Focus on creating a professional, engaging README that highlights the project's strengths
- Ensure the README is well-structured and provides clear, accurate information

Provide your final README.md content within <readme> tags. Include no explanations or comments outside of these tags.
	
	每次执行Function时，我都会把执行结果放在对话历史的最后面，发送给你。请你根据对话历史，生成最终的README.md文件。
	`
	path, _ := s.parent.RepoService().GetRepoPath(ctx)
	repName := s.parent.RepoService().GetRepoName(ctx)
	return fmt.Sprintf(prompt, path, repName)
}
func (s *docReadmeService) Generate(ctx context.Context, analysis *models.DocAnalysis) error {
	reader, err := s.parent.chat(ctx, s.prompt(ctx), "")
	if err != nil {
		return err

	}
	_, err = s.parent.readAndWrite(ctx, reader, analysis)
	if err != nil {
		return err
	}
	return nil
}
