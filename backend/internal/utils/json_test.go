package utils

import (
	"strings"
	"testing"
)

// TestExtractYAMLFromCodeBlock 验证从代码块中提取 YAML
func TestExtractYAMLFromCodeBlock(t *testing.T) {
	content := "说明文本\n```yaml\n" +
		"dirs:\n" +
		"  - type: project_overview\n" +
		"    title: 项目简介\n" +
		"    sort_order: 1\n" +
		"    evidence:\n" +
		"      - aspect: 目录结构\n" +
		"        source: |\n" +
		"          - /README.md\n" +
		"        detail: |\n" +
		"          依据 README\n" +
		"analysis_summary: \"总结\"\n" +
		"```\n结尾文本"
	extracted := ExtractYAML(content)
	if extracted == "" {
		t.Fatalf("expected yaml content, got empty")
	}
	if extracted[0:5] != "dirs:" {
		t.Fatalf("unexpected yaml prefix: %s", extracted)
	}
}

// TestExtractYAMLFromDirsLine 验证从普通文本中定位 dirs 开始位置
func TestExtractYAMLFromDirsLine(t *testing.T) {
	content := "前置说明\ndirs:\n  - type: project_overview\nanalysis_summary: \"总结\""
	extracted := ExtractYAML(content)
	if extracted[0:5] != "dirs:" {
		t.Fatalf("unexpected yaml prefix: %s", extracted)
	}
}

func TestExtractYAMLWithoutWrapper(t *testing.T) {
	content := "说明文本\n" +
		"dirs:\n" +
		"  - type: project_overview\n" +
		"    title: 项目简介\n" +
		"    sort_order: 1\n" +
		"analysis_summary: \"总结\"\n" +
		"额外说明"
	extracted := ExtractYAML(content)
	if !strings.HasPrefix(extracted, "dirs:") {
		t.Fatalf("unexpected yaml prefix: %s", extracted)
	}
	if strings.Contains(extracted, "额外说明") {
		t.Fatalf("unexpected trailing text: %s", extracted)
	}
}
