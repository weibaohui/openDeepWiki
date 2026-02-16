package service

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jung-kurt/gofpdf"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

//go:embed assets/fonts/JetBrainsMono-Regular.ttf
var jetBrainsMonoRegular []byte

//go:embed assets/fonts/JetBrainsMono-Bold.ttf
var jetBrainsMonoBold []byte

//go:embed assets/fonts/NotoSansCJKsc-Regular.ttf
var notoSansCJKRegular []byte

//go:embed assets/fonts/NotoSansCJKsc-Bold.ttf
var notoSansCJKBold []byte

type DocumentService struct {
	cfg        *config.Config
	docRepo    repository.DocumentRepository
	repoRepo   repository.RepoRepository
	ratingRepo repository.DocumentRatingRepository
}

// NewDocumentService 创建文档服务
func NewDocumentService(cfg *config.Config, docRepo repository.DocumentRepository, repoRepo repository.RepoRepository, ratingRepo repository.DocumentRatingRepository) *DocumentService {
	return &DocumentService{
		cfg:        cfg,
		docRepo:    docRepo,
		repoRepo:   repoRepo,
		ratingRepo: ratingRepo,
	}
}

type CreateDocumentRequest struct {
	RepositoryID uint   `json:"repository_id"`
	TaskID       uint   `json:"task_id"`
	Title        string `json:"title"`
	Filename     string `json:"filename"`
	Content      string `json:"content"`
	SortOrder    int    `json:"sort_order"`
}

func (s *DocumentService) UpdateTaskID(docID uint, taskID uint) error {
	return s.docRepo.UpdateTaskID(docID, taskID)
}
func (s *DocumentService) TransferLatest(oldDocID uint, newDocID uint) error {
	return s.docRepo.TransferLatest(oldDocID, newDocID)
}

func (s *DocumentService) Create(req CreateDocumentRequest) (*model.Document, error) {
	doc := &model.Document{
		RepositoryID: req.RepositoryID,
		TaskID:       req.TaskID,
		Title:        req.Title,
		Filename:     req.Filename,
		Content:      req.Content,
		SortOrder:    req.SortOrder,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.docRepo.CreateVersioned(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *DocumentService) GetByRepository(repoID uint) ([]model.Document, error) {
	return s.docRepo.GetByRepository(repoID)
}

func (s *DocumentService) Get(id uint) (*model.Document, error) {
	return s.docRepo.Get(id)
}

func (s *DocumentService) GetVersions(docID uint) ([]model.Document, error) {
	doc, err := s.docRepo.Get(docID)
	if err != nil {
		return nil, err
	}
	return s.docRepo.GetVersions(doc.RepositoryID, doc.Title)
}

func (s *DocumentService) Update(docID uint, content string) (*model.Document, error) {
	doc, err := s.docRepo.Get(docID)
	if err != nil {
		return nil, err
	}

	doc.Content = content
	doc.UpdatedAt = time.Now()
	if err := s.docRepo.Save(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *DocumentService) Delete(id uint) error {
	return s.docRepo.Delete(id)
}

func (s *DocumentService) DeleteByTaskID(taskID uint) error {
	return s.docRepo.DeleteByTaskID(taskID)
}

func (s *DocumentService) ExportAll(repoID uint) ([]byte, string, error) {
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return nil, "", err
	}

	docs, err := s.docRepo.GetByRepository(repoID)
	if err != nil {
		return nil, "", err
	}

	if len(docs) == 0 {
		return nil, "", fmt.Errorf("no documents to export")
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	indexContent := s.generateIndex(repo.Name, docs)
	indexFile, err := zipWriter.Create("index.md")
	if err != nil {
		return nil, "", err
	}
	indexFile.Write([]byte(indexContent))

	for _, doc := range docs {
		f, err := zipWriter.Create(doc.Filename)
		if err != nil {
			return nil, "", err
		}
		f.Write([]byte(doc.Content))
	}

	if err := zipWriter.Close(); err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("%s-docs.zip", repo.Name)
	return buf.Bytes(), filename, nil
}

// ExportPDF 导出仓库下所有文档为PDF
func (s *DocumentService) ExportPDF(repoID uint) ([]byte, string, error) {
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return nil, "", err
	}

	docs, err := s.docRepo.GetByRepository(repoID)
	if err != nil {
		return nil, "", err
	}

	if len(docs) == 0 {
		return nil, "", fmt.Errorf("no documents to export")
	}

	klog.V(6).Infof("开始导出PDF: repoID=%d, 文档数量=%d", repoID, len(docs))

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	bodyFont, monoFont := registerPDFFonts(pdf)

	for _, doc := range docs {
		pdf.AddPage()
		pdf.SetFont(bodyFont, "B", 16)
		cleanTitle := sanitizeInlineMarkdown(stripHTMLTags(strings.TrimSpace(doc.Title)))
		pdf.Bookmark(cleanTitle, 0, -1)
		pdf.MultiCell(0, 8, cleanTitle, "", "L", false)
		pdf.Ln(2)

		content := strings.TrimSpace(doc.Content)
		renderMarkdownToPDF(pdf, content, bodyFont, monoFont, cleanTitle)
	}

	buf := new(bytes.Buffer)
	if err := pdf.Output(buf); err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("%s-docs.pdf", repo.Name)
	klog.V(6).Infof("导出PDF完成: repoID=%d, 文件大小=%d", repoID, buf.Len())
	return buf.Bytes(), filename, nil
}

// registerPDFFonts 注册PDF字体并返回可用字体族名
func registerPDFFonts(pdf *gofpdf.Fpdf) (string, string) {
	pdf.AddUTF8FontFromBytes("NotoSansCJK", "", notoSansCJKRegular)
	if err := pdf.Error(); err != nil {
		klog.V(6).Infof("注册PDF中文字体失败，尝试JetBrains Mono: %v", err)
		pdf.SetError(nil)
		pdf.AddUTF8FontFromBytes("JetBrainsMono", "", jetBrainsMonoRegular)
		if err := pdf.Error(); err != nil {
			klog.V(6).Infof("注册PDF字体失败，回退Helvetica: %v", err)
			pdf.SetError(nil)
			return "Helvetica", "Helvetica"
		}
		pdf.AddUTF8FontFromBytes("JetBrainsMono", "B", jetBrainsMonoBold)
		if err := pdf.Error(); err != nil {
			klog.V(6).Infof("注册PDF字体失败，回退Helvetica: %v", err)
			pdf.SetError(nil)
			return "Helvetica", "Helvetica"
		}
		return "JetBrainsMono", "JetBrainsMono"
	}
	pdf.AddUTF8FontFromBytes("NotoSansCJK", "B", notoSansCJKBold)
	if err := pdf.Error(); err != nil {
		klog.V(6).Infof("注册PDF中文字体失败，尝试JetBrains Mono: %v", err)
		pdf.SetError(nil)
		pdf.AddUTF8FontFromBytes("JetBrainsMono", "", jetBrainsMonoRegular)
		if err := pdf.Error(); err != nil {
			klog.V(6).Infof("注册PDF字体失败，回退Helvetica: %v", err)
			pdf.SetError(nil)
			return "Helvetica", "Helvetica"
		}
		pdf.AddUTF8FontFromBytes("JetBrainsMono", "B", jetBrainsMonoBold)
		if err := pdf.Error(); err != nil {
			klog.V(6).Infof("注册PDF字体失败，回退Helvetica: %v", err)
			pdf.SetError(nil)
			return "Helvetica", "Helvetica"
		}
		return "JetBrainsMono", "JetBrainsMono"
	}
	pdf.AddUTF8FontFromBytes("JetBrainsMono", "", jetBrainsMonoRegular)
	if err := pdf.Error(); err != nil {
		klog.V(6).Infof("注册PDF等宽字体失败，回退为中文字体: %v", err)
		pdf.SetError(nil)
		return "NotoSansCJK", "NotoSansCJK"
	}
	pdf.AddUTF8FontFromBytes("JetBrainsMono", "B", jetBrainsMonoBold)
	if err := pdf.Error(); err != nil {
		klog.V(6).Infof("注册PDF等宽字体失败，回退为中文字体: %v", err)
		pdf.SetError(nil)
		return "NotoSansCJK", "NotoSansCJK"
	}
	return "NotoSansCJK", "JetBrainsMono"
}

// renderMarkdownToPDF 渲染Markdown内容到PDF
func renderMarkdownToPDF(pdf *gofpdf.Fpdf, content string, bodyFont string, monoFont string, docTitle string) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	leftMargin, _, _, _ := pdf.GetMargins()
	lineHeight := 6.0
	inCodeBlock := false
	var codeLines []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				renderCodeBlock(pdf, codeLines, monoFont, bodyFont, leftMargin)
				codeLines = nil
				inCodeBlock = false
			} else {
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			codeLines = append(codeLines, line)
			continue
		}

		if isTableStart(lines, i) {
			tableLines := collectTableLines(lines, i)
			renderMarkdownTable(pdf, tableLines, bodyFont, leftMargin)
			i += len(tableLines) - 1
			continue
		}

		if trimmed == "" {
			pdf.Ln(3)
			continue
		}

		if level, heading := parseHeading(trimmed); level > 0 {
			headingText := sanitizeInlineMarkdown(stripHTMLTags(strings.TrimSpace(heading)))
			size := 14.0
			switch level {
			case 1:
				size = 18
			case 2:
				size = 16
			case 3:
				size = 14
			case 4:
				size = 13
			default:
				size = 12
			}
			pdf.SetFont(bodyFont, "B", size)
			if !(level == 1 && headingText == docTitle) {
				pdf.Bookmark(headingText, level, -1)
			}
			pdf.MultiCell(0, 7, headingText, "", "L", false)
			pdf.Ln(1)
			continue
		}

		if strings.HasPrefix(trimmed, "> ") {
			pdf.SetTextColor(90, 90, 90)
			renderInlineText(pdf, strings.TrimSpace(trimmed[2:]), bodyFont, 12, lineHeight, leftMargin+4)
			pdf.SetTextColor(0, 0, 0)
			continue
		}

		if ordered, item := parseOrderedList(trimmed); ordered {
			renderInlineText(pdf, item, bodyFont, 12, lineHeight, leftMargin+4)
			continue
		}

		if unordered, item := parseUnorderedList(trimmed); unordered {
			renderInlineText(pdf, item, bodyFont, 12, lineHeight, leftMargin+4)
			continue
		}

		renderInlineText(pdf, trimmed, bodyFont, 12, lineHeight, leftMargin)
	}

	if inCodeBlock && len(codeLines) > 0 {
		renderCodeBlock(pdf, codeLines, monoFont, bodyFont, leftMargin)
	}
}

func renderCodeBlock(pdf *gofpdf.Fpdf, lines []string, monoFont string, bodyFont string, leftMargin float64) {
	if len(lines) == 0 {
		return
	}
	pdf.SetFillColor(245, 245, 245)
	pdf.SetTextColor(50, 50, 50)
	for _, line := range lines {
		font := monoFont
		if containsCJK(line) {
			font = bodyFont
		}
		pdf.SetFont(font, "", 10)
		pdf.SetX(leftMargin + 2)
		pdf.MultiCell(0, 5, line, "", "L", true)
	}
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(1)
}

func parseHeading(line string) (int, string) {
	count := 0
	for count < len(line) && line[count] == '#' {
		count++
	}
	if count == 0 || count > 6 {
		return 0, ""
	}
	if len(line) <= count || line[count] != ' ' {
		return 0, ""
	}
	return count, strings.TrimSpace(line[count+1:])
}

// parseOrderedList 解析有序列表项
func parseOrderedList(line string) (bool, string) {
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == 0 || i+1 >= len(line) {
		return false, ""
	}
	if line[i] != '.' || line[i+1] != ' ' {
		return false, ""
	}
	number, err := strconv.Atoi(line[:i])
	if err != nil {
		return false, ""
	}
	item := strings.TrimSpace(line[i+2:])
	item = normalizeTaskMarker(item)
	return true, fmt.Sprintf("%d. %s", number, item)
}

// parseUnorderedList 解析无序列表项
func parseUnorderedList(line string) (bool, string) {
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "+ ") {
		item := strings.TrimSpace(line[2:])
		item = normalizeTaskMarker(item)
		return true, fmt.Sprintf("• %s", item)
	}
	return false, ""
}

func normalizeTaskMarker(item string) string {
	if strings.HasPrefix(item, "[ ] ") || strings.HasPrefix(item, "[x] ") || strings.HasPrefix(item, "[X] ") {
		return strings.TrimSpace(item[4:])
	}
	return item
}

func sanitizeInlineMarkdown(text string) string {
	text = stripMarkdownLinks(text)
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ReplaceAll(text, "`", "")
	return text
}

// renderInlineText 渲染包含链接的单行文本
func renderInlineText(pdf *gofpdf.Fpdf, text string, font string, fontSize float64, lineHeight float64, indent float64) {
	pdf.SetFont(font, "", fontSize)
	pdf.SetX(indent)
	segments := parseInlineLinks(text)
	for _, segment := range segments {
		segmentText := sanitizeInlineMarkdown(stripHTMLTags(segment.Text))
		if segmentText == "" {
			continue
		}
		if segment.Link != "" {
			pdf.WriteLinkString(lineHeight, segment.Link, segmentText)
			continue
		}
		pdf.Write(lineHeight, segmentText)
	}
	pdf.Ln(lineHeight)
}

// parseInlineLinks 解析文本中的HTML与Markdown链接
func parseInlineLinks(text string) []inlineLinkSegment {
	segments := parseHTMLLinks(text)
	var result []inlineLinkSegment
	for _, segment := range segments {
		if segment.Link != "" {
			result = append(result, segment)
			continue
		}
		result = append(result, parseMarkdownLinks(segment.Text)...)
	}
	if len(result) == 0 {
		return []inlineLinkSegment{{Text: text}}
	}
	return result
}

// parseHTMLLinks 解析文本中的a标签链接
func parseHTMLLinks(text string) []inlineLinkSegment {
	var segments []inlineLinkSegment
	lower := strings.ToLower(text)
	for {
		start := strings.Index(lower, "<a")
		if start == -1 {
			if text != "" {
				segments = append(segments, inlineLinkSegment{Text: text})
			}
			return segments
		}
		if start > 0 {
			segments = append(segments, inlineLinkSegment{Text: text[:start]})
		}
		openEnd := strings.Index(lower[start:], ">")
		if openEnd == -1 {
			segments = append(segments, inlineLinkSegment{Text: text[start:]})
			return segments
		}
		openEnd += start
		href := extractHref(text[start:openEnd])
		closeIndex := strings.Index(strings.ToLower(text[openEnd:]), "</a>")
		if closeIndex == -1 {
			segments = append(segments, inlineLinkSegment{Text: text[openEnd+1:]})
			return segments
		}
		closeIndex += openEnd
		linkText := text[openEnd+1 : closeIndex]
		segments = append(segments, inlineLinkSegment{Text: linkText, Link: href})
		text = text[closeIndex+4:]
		lower = strings.ToLower(text)
	}
}

// parseMarkdownLinks 解析Markdown链接
func parseMarkdownLinks(text string) []inlineLinkSegment {
	var segments []inlineLinkSegment
	for {
		start := strings.Index(text, "[")
		if start == -1 {
			if text != "" {
				segments = append(segments, inlineLinkSegment{Text: text})
			}
			return segments
		}
		mid := strings.Index(text[start:], "]")
		if mid == -1 {
			segments = append(segments, inlineLinkSegment{Text: text})
			return segments
		}
		mid += start
		if mid+1 >= len(text) || text[mid+1] != '(' {
			segments = append(segments, inlineLinkSegment{Text: text[:mid+1]})
			text = text[mid+1:]
			continue
		}
		end := strings.Index(text[mid+2:], ")")
		if end == -1 {
			segments = append(segments, inlineLinkSegment{Text: text})
			return segments
		}
		end += mid + 2
		if start > 0 {
			segments = append(segments, inlineLinkSegment{Text: text[:start]})
		}
		label := text[start+1 : mid]
		link := text[mid+2 : end]
		segments = append(segments, inlineLinkSegment{Text: label, Link: link})
		text = text[end+1:]
	}
}

// extractHref 提取a标签中的href地址
func extractHref(text string) string {
	lower := strings.ToLower(text)
	hrefIndex := strings.Index(lower, "href=")
	if hrefIndex == -1 {
		return ""
	}
	rest := text[hrefIndex+5:]
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ""
	}
	if rest[0] == '"' || rest[0] == '\'' {
		quote := rest[0]
		end := strings.IndexByte(rest[1:], quote)
		if end == -1 {
			return ""
		}
		return rest[1 : end+1]
	}
	for i, r := range rest {
		if r == ' ' || r == '>' {
			return rest[:i]
		}
	}
	return rest
}

// stripHTMLTags 移除文本中的HTML标签
func stripHTMLTags(text string) string {
	var builder strings.Builder
	inTag := false
	for _, r := range text {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func stripMarkdownLinks(text string) string {
	for {
		start := strings.Index(text, "[")
		if start == -1 {
			return text
		}
		mid := strings.Index(text[start:], "]")
		if mid == -1 {
			return text
		}
		mid += start
		if mid+1 >= len(text) || text[mid+1] != '(' {
			return text
		}
		end := strings.Index(text[mid+2:], ")")
		if end == -1 {
			return text
		}
		end += mid + 2
		label := text[start+1 : mid]
		text = text[:start] + label + text[end+1:]
	}
}

// inlineLinkSegment 表示文本中的链接片段
type inlineLinkSegment struct {
	Text string
	Link string
}

func containsCJK(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func isTableStart(lines []string, index int) bool {
	if index+1 >= len(lines) {
		return false
	}
	return isTableLine(lines[index]) && isTableSeparatorLine(lines[index+1])
}

func collectTableLines(lines []string, start int) []string {
	var tableLines []string
	for i := start; i < len(lines); i++ {
		if !isTableLine(lines[i]) {
			break
		}
		tableLines = append(tableLines, lines[i])
	}
	return tableLines
}

func isTableLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	return strings.Contains(trimmed, "|")
}

func isTableSeparatorLine(line string) bool {
	cells := parseTableRow(line)
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			return false
		}
		dashCount := 0
		for _, r := range cell {
			if r == '-' {
				dashCount++
				continue
			}
			if r == ':' {
				continue
			}
			return false
		}
		if dashCount < 3 {
			return false
		}
	}
	return true
}

func parseTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, sanitizeInlineMarkdown(strings.TrimSpace(part)))
	}
	return cells
}

func renderMarkdownTable(pdf *gofpdf.Fpdf, lines []string, bodyFont string, leftMargin float64) {
	if len(lines) < 2 {
		return
	}
	header := parseTableRow(lines[0])
	if !isTableSeparatorLine(lines[1]) {
		return
	}
	rows := [][]string{header}
	for _, line := range lines[2:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		rows = append(rows, parseTableRow(line))
	}
	colCount := 0
	for _, row := range rows {
		if len(row) > colCount {
			colCount = len(row)
		}
	}
	if colCount == 0 {
		return
	}
	for i, row := range rows {
		if len(row) < colCount {
			padded := make([]string, colCount)
			copy(padded, row)
			rows[i] = padded
		}
	}
	pageWidth, pageHeight := pdf.GetPageSize()
	_, _, rightMargin, bottomMargin := pdf.GetMargins()
	tableWidth := pageWidth - leftMargin - rightMargin
	colWidth := tableWidth / float64(colCount)
	lineHeight := 4.2
	pdf.SetCellMargin(0)
	startY := pdf.GetY()
	for rowIndex, row := range rows {
		pdf.SetFont(bodyFont, "", 10)
		pdf.SetFillColor(255, 255, 255)
		if rowIndex == 0 {
			pdf.SetFont(bodyFont, "B", 10)
			pdf.SetFillColor(235, 235, 235)
		}
		rowHeight := calcTableRowHeight(pdf, row, colWidth, lineHeight)
		if startY+rowHeight > pageHeight-bottomMargin {
			pdf.AddPage()
			startY = pdf.GetY()
		}
		x := leftMargin
		y := startY
		for _, cell := range row {
			drawStyle := "D"
			if rowIndex == 0 {
				drawStyle = "DF"
			}
			pdf.Rect(x, y, colWidth, rowHeight, drawStyle)
			lines := pdf.SplitLines([]byte(cell), colWidth)
			textHeight := float64(len(lines)) * lineHeight
			offsetY := (rowHeight - textHeight) / 2
			if offsetY < 0 {
				offsetY = 0
			}
			pdf.SetXY(x, y+offsetY)
			pdf.MultiCell(colWidth, lineHeight, cell, "", "L", false)
			pdf.SetXY(x+colWidth, y)
			x += colWidth
		}
		startY = y + rowHeight
		pdf.SetY(startY)
	}
	pdf.SetCellMargin(1)
	pdf.Ln(1)
}

func calcTableRowHeight(pdf *gofpdf.Fpdf, row []string, colWidth float64, lineHeight float64) float64 {
	maxLines := 1
	for _, cell := range row {
		lines := pdf.SplitLines([]byte(cell), colWidth)
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}
	return float64(maxLines) * lineHeight
}

func (s *DocumentService) generateIndex(repoName string, docs []model.Document) string {
	content := fmt.Sprintf("# %s - 项目文档\n\n", repoName)
	content += "## 目录\n\n"

	for _, doc := range docs {
		content += fmt.Sprintf("- [%s](%s)\n", doc.Title, doc.Filename)
	}

	return content
}

func (s *DocumentService) GetIndex(repoID uint) (string, error) {
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return "", err
	}

	docs, err := s.docRepo.GetByRepository(repoID)
	if err != nil {
		return "", err
	}

	return s.generateIndex(repo.Name, docs), nil
}

// GetRedirectURL 获取代码跳转链接
func (s *DocumentService) GetRedirectURL(docID uint, filePath string) (string, error) {
	doc, err := s.docRepo.Get(docID)
	if err != nil {
		return "", err
	}

	repo, err := s.repoRepo.GetBasic(doc.RepositoryID)
	if err != nil {
		return "", err
	}

	// 处理仓库 URL
	repoURL := repo.URL
	if before, ok := strings.CutSuffix(repoURL, ".git"); ok {
		repoURL = before
	}

	branch := repo.CloneBranch
	if branch == "" {
		branch = "main" // 默认回退
	}

	// 清理文件路径 (移除开头的 /)
	filePath = strings.TrimPrefix(filePath, "/")

	// 构造 URL
	// 假设是 GitHub/GitLab 风格: base/blob/branch/path
	// TODO 以后要兼容各种类型
	return fmt.Sprintf("%s/blob/%s/%s", repoURL, branch, filePath), nil
}

// SubmitRating 提交文档评分并返回统计信息
func (s *DocumentService) SubmitRating(documentID uint, score int) (*model.DocumentRatingStats, error) {
	if s.ratingRepo == nil {
		return nil, fmt.Errorf("rating repository not configured")
	}
	if score < 1 || score > 5 {
		return nil, fmt.Errorf("score must be between 1 and 5")
	}

	klog.V(6).Infof("SubmitRating: document_id=%d score=%d", documentID, score)

	latest, err := s.ratingRepo.GetLatestByDocumentID(documentID)
	if err != nil {
		return nil, err
	}
	if latest != nil && latest.Score == score && time.Since(latest.CreatedAt) <= 10*time.Second {
		klog.V(6).Infof("SubmitRating: duplicate ignored document_id=%d score=%d", documentID, score)
		return s.GetRatingStats(documentID)
	}

	now := time.Now()
	rating := &model.DocumentRating{
		DocumentID: documentID,
		Score:      score,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.ratingRepo.Create(rating); err != nil {
		return nil, err
	}

	klog.V(6).Infof("SubmitRating: created rating_id=%d document_id=%d", rating.ID, documentID)
	return s.GetRatingStats(documentID)
}

// GetRatingStats 获取文档评分统计信息
func (s *DocumentService) GetRatingStats(documentID uint) (*model.DocumentRatingStats, error) {
	if s.ratingRepo == nil {
		return nil, fmt.Errorf("rating repository not configured")
	}
	stats, err := s.ratingRepo.GetStatsByDocumentID(documentID)
	if err != nil {
		return nil, err
	}
	stats.AverageScore = math.Round(stats.AverageScore*10) / 10
	klog.V(6).Infof("GetRatingStats: document_id=%d average=%.1f count=%d", documentID, stats.AverageScore, stats.RatingCount)
	return stats, nil
}

// GetTokenUsage 获取文档的 Token 用量数据
func (s *DocumentService) GetTokenUsage(docID uint) (*model.TaskUsage, error) {
	return s.docRepo.GetTokenUsageByDocID(docID)
}
