package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"gorm.io/gorm"
)

// TestReadDocToolInvokableRun 验证 read_doc 工具按ID读取文档内容
func TestReadDocToolInvokableRun(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db error: %v", err)
	}
	if err := db.AutoMigrate(&model.Document{}); err != nil {
		t.Fatalf("migrate error: %v", err)
	}

	docRepo := repository.NewDocumentRepository(db)
	doc := &model.Document{
		RepositoryID: 1,
		TaskID:       1,
		Title:        "示例文档",
		Filename:     "demo.md",
		Content:      "这里是全文内容",
	}
	if err := docRepo.Create(doc); err != nil {
		t.Fatalf("create doc error: %v", err)
	}

	readTool := NewReadDocTool(docRepo)
	argsJSON, _ := json.Marshal(struct {
		DocID uint `json:"doc_id"`
	}{DocID: doc.ID})

	result, err := readTool.InvokableRun(context.Background(), string(argsJSON))
	if err != nil {
		t.Fatalf("InvokableRun error: %v", err)
	}
	if result != doc.Content {
		t.Fatalf("unexpected content: %s", result)
	}
}
