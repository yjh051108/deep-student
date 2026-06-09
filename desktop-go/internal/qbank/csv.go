package qbank

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

type CsvDuplicateStrategy string

const (
	CsvDuplicateSkip      CsvDuplicateStrategy = "skip"
	CsvDuplicateOverwrite CsvDuplicateStrategy = "overwrite"
	CsvDuplicateMerge     CsvDuplicateStrategy = "merge"
)

type CsvPreviewResult struct {
	Headers   []string   `json:"headers"`
	Rows      [][]string `json:"rows"`
	TotalRows int        `json:"total_rows"`
	Encoding  string     `json:"encoding"`
}

type CsvImportRequest struct {
	FilePath          string               `json:"file_path"`
	ExamID            string               `json:"exam_id"`
	FieldMapping      map[string]string    `json:"field_mapping"`
	DuplicateStrategy CsvDuplicateStrategy `json:"duplicate_strategy,omitempty"`
	FolderID          *string              `json:"folder_id,omitempty"`
	ExamName          *string              `json:"exam_name,omitempty"`
}

type CsvImportError struct {
	Row     int     `json:"row"`
	Message string  `json:"message"`
	RawData *string `json:"raw_data,omitempty"`
}

type CsvImportResult struct {
	SuccessCount int              `json:"success_count"`
	SkippedCount int              `json:"skipped_count"`
	FailedCount  int              `json:"failed_count"`
	Errors       []CsvImportError `json:"errors"`
	ExamID       string           `json:"exam_id"`
	TotalRows    int              `json:"total_rows"`
}

type CsvExportRequest struct {
	ExamID         string           `json:"exam_id"`
	FilePath       string           `json:"file_path"`
	Fields         []string         `json:"fields,omitempty"`
	Filters        *QuestionFilters `json:"filters,omitempty"`
	IncludeAnswers bool             `json:"include_answers,omitempty"`
	Encoding       string           `json:"encoding,omitempty"`
}

type CsvExportResult struct {
	ExportedCount int    `json:"exported_count"`
	FilePath      string `json:"file_path"`
	FileSize      int64  `json:"file_size"`
}

var csvImportTargetFields = map[string]bool{
	"question_label": true,
	"content":        true,
	"options":        true,
	"answer":         true,
	"explanation":    true,
	"question_type":  true,
	"difficulty":     true,
	"tags":           true,
	"images":         true,
}

var csvExportableFields = [][2]string{
	{"content", "题干内容"},
	{"question_type", "题目类型"},
	{"options", "选项"},
	{"answer", "答案"},
	{"explanation", "解析"},
	{"difficulty", "难度"},
	{"tags", "标签"},
	{"images", "关联图片"},
	{"question_label", "题号"},
	{"user_answer", "用户答案"},
	{"is_correct", "是否正确"},
	{"attempt_count", "答题次数"},
	{"correct_count", "正确次数"},
	{"status", "学习状态"},
	{"is_favorite", "收藏"},
	{"user_note", "笔记"},
	{"created_at", "创建时间"},
	{"updated_at", "更新时间"},
}

func (s *Service) GetCsvPreview(filePath string, rows int) (CsvPreviewResult, error) {
	if rows < 1 {
		rows = 5
	}
	content, encoding, err := readCsvTextFile(filePath)
	if err != nil {
		return CsvPreviewResult{}, err
	}
	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return CsvPreviewResult{}, errors.New("CSV file is empty")
		}
		return CsvPreviewResult{}, fmt.Errorf("read CSV headers: %w", err)
	}
	headers = trimCsvRecord(headers)

	previewRows := make([][]string, 0, rows)
	totalRows := 0
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return CsvPreviewResult{}, fmt.Errorf("read CSV row %d: %w", totalRows+2, err)
		}
		totalRows++
		if len(previewRows) < rows {
			previewRows = append(previewRows, trimCsvRecord(record))
		}
	}

	return CsvPreviewResult{
		Headers:   headers,
		Rows:      previewRows,
		TotalRows: totalRows,
		Encoding:  encoding,
	}, nil
}

func (s *Service) ImportQuestionsCsv(request CsvImportRequest) (CsvImportResult, error) {
	if strings.TrimSpace(request.ExamID) == "" {
		return CsvImportResult{}, errors.New("exam_id is required")
	}
	if request.DuplicateStrategy == "" {
		request.DuplicateStrategy = CsvDuplicateSkip
	}
	if request.DuplicateStrategy != CsvDuplicateSkip &&
		request.DuplicateStrategy != CsvDuplicateOverwrite &&
		request.DuplicateStrategy != CsvDuplicateMerge {
		return CsvImportResult{}, fmt.Errorf("unsupported duplicate_strategy: %s", request.DuplicateStrategy)
	}

	content, _, err := readCsvTextFile(request.FilePath)
	if err != nil {
		return CsvImportResult{}, err
	}
	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return CsvImportResult{}, errors.New("CSV file is empty")
		}
		return CsvImportResult{}, fmt.Errorf("read CSV headers: %w", err)
	}
	headers = trimCsvRecord(headers)
	if err := validateCsvFieldMapping(headers, request.FieldMapping); err != nil {
		return CsvImportResult{}, err
	}

	records := make([][]string, 0)
	rowNumbers := make([]int, 0)
	parseErrors := make([]error, 0)
	for rowNumber := 2; ; rowNumber++ {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			records = append(records, nil)
			rowNumbers = append(rowNumbers, rowNumber)
			parseErrors = append(parseErrors, err)
			continue
		}
		records = append(records, trimCsvRecord(record))
		rowNumbers = append(rowNumbers, rowNumber)
		parseErrors = append(parseErrors, nil)
	}

	result := CsvImportResult{
		ExamID:    request.ExamID,
		TotalRows: len(records),
		Errors:    []CsvImportError{},
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existingByHash := s.existingQuestionHashesLocked(request.ExamID)
	for i, record := range records {
		rowNumber := rowNumbers[i]
		if record == nil {
			result.FailedCount++
			message := "parse CSV row failed"
			if parseErrors[i] != nil {
				message = parseErrors[i].Error()
			}
			result.Errors = append(result.Errors, CsvImportError{
				Row:     rowNumber,
				Message: message,
			})
			continue
		}

		outcome, err := s.importCsvRecordLocked(request, headers, record, existingByHash, rowNumber)
		if err != nil {
			result.FailedCount++
			raw := strings.Join(record, ",")
			result.Errors = append(result.Errors, CsvImportError{
				Row:     rowNumber,
				Message: err.Error(),
				RawData: &raw,
			})
			continue
		}
		switch outcome {
		case "skipped":
			result.SkippedCount++
		default:
			result.SuccessCount++
		}
	}
	if err := s.flushLocked(); err != nil {
		return CsvImportResult{}, err
	}
	return result, nil
}

func (s *Service) ExportQuestionsCsv(request CsvExportRequest) (CsvExportResult, error) {
	if strings.TrimSpace(request.ExamID) == "" {
		return CsvExportResult{}, errors.New("exam_id is required")
	}
	if strings.TrimSpace(request.FilePath) == "" {
		return CsvExportResult{}, errors.New("file_path is required")
	}
	if isVirtualCsvPath(request.FilePath) {
		return CsvExportResult{}, errors.New("virtual export paths are not available in the Go CSV bridge yet")
	}
	localPath := normalizeCsvLocalPath(request.FilePath)
	if hasParentTraversal(localPath) {
		return CsvExportResult{}, errors.New("file_path must not contain '..'")
	}

	fields := request.Fields
	if len(fields) == 0 {
		fields = defaultCsvExportFields(request.IncludeAnswers)
	}
	if err := validateCsvExportFields(fields); err != nil {
		return CsvExportResult{}, err
	}

	s.mu.RLock()
	questions := make([]Question, 0)
	for _, question := range s.state.Questions {
		if question.ExamID != request.ExamID || !matchesFilters(question, request.Filters) {
			continue
		}
		questions = append(questions, question)
	}
	sortQuestions(questions)
	s.mu.RUnlock()

	if len(questions) == 0 {
		return CsvExportResult{}, errors.New("no questions to export")
	}

	var buffer bytes.Buffer
	if normalizeCsvEncoding(request.Encoding) == "utf8_bom" {
		buffer.Write([]byte{0xEF, 0xBB, 0xBF})
	}

	writer := csv.NewWriter(&buffer)
	writer.UseCRLF = false
	headers := make([]string, 0, len(fields))
	for _, field := range fields {
		headers = append(headers, csvExportDisplayName(field))
	}
	if err := writer.Write(headers); err != nil {
		return CsvExportResult{}, err
	}
	for _, question := range questions {
		row := make([]string, 0, len(fields))
		for _, field := range fields {
			row = append(row, csvFieldValue(question, field))
		}
		if err := writer.Write(neutralizeCsvFormulaCells(row)); err != nil {
			return CsvExportResult{}, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return CsvExportResult{}, err
	}

	output := buffer.Bytes()
	if normalizeCsvEncoding(request.Encoding) == "gbk" {
		encoded, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(buffer.String()))
		if err != nil {
			return CsvExportResult{}, fmt.Errorf("GBK encode failed: %w", err)
		}
		output = encoded
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return CsvExportResult{}, err
	}
	if err := os.WriteFile(localPath, output, 0o600); err != nil {
		return CsvExportResult{}, err
	}
	info, err := os.Stat(localPath)
	if err != nil {
		return CsvExportResult{}, err
	}
	return CsvExportResult{
		ExportedCount: len(questions),
		FilePath:      localPath,
		FileSize:      info.Size(),
	}, nil
}

func (s *Service) GetCsvExportableFields() [][]string {
	out := make([][]string, 0, len(csvExportableFields))
	for _, field := range csvExportableFields {
		out = append(out, []string{field[0], field[1]})
	}
	return out
}

func (s *Service) existingQuestionHashesLocked(examID string) map[string]int {
	existing := map[string]int{}
	for index, question := range s.state.Questions {
		if question.ExamID != examID {
			continue
		}
		existing[contentHash(question.Content)] = index
	}
	return existing
}

func (s *Service) importCsvRecordLocked(
	request CsvImportRequest,
	headers []string,
	record []string,
	existingByHash map[string]int,
	rowNumber int,
) (string, error) {
	values := mapCsvFields(headers, record, request.FieldMapping)
	content := strings.TrimSpace(values["content"])
	if content == "" {
		return "", fmt.Errorf("row %d: content is empty", rowNumber)
	}

	hash := contentHash(content)
	if index, exists := existingByHash[hash]; exists {
		switch request.DuplicateStrategy {
		case CsvDuplicateSkip:
			return "skipped", nil
		case CsvDuplicateOverwrite:
			previous := s.state.Questions[index]
			updated := applyCsvOverwrite(previous, values)
			updated.UpdatedAt = nowISO()
			s.state.Questions[index] = updated
			if err := s.syncQuestionResourceLocked(index); err != nil {
				s.state.Questions[index] = previous
				return "", err
			}
			existingByHash[contentHash(updated.Content)] = index
			return "updated", nil
		case CsvDuplicateMerge:
			previous := s.state.Questions[index]
			updated := applyCsvMerge(previous, values)
			updated.UpdatedAt = nowISO()
			s.state.Questions[index] = updated
			if err := s.syncQuestionResourceLocked(index); err != nil {
				s.state.Questions[index] = previous
				return "", err
			}
			existingByHash[contentHash(updated.Content)] = index
			return "updated", nil
		}
	}

	sourceType := "imported"
	sourceRef := "csv"
	questionType := parseCsvQuestionType(values["question_type"])
	difficulty := parseCsvDifficulty(values["difficulty"])
	label := strings.TrimSpace(values["question_label"])
	if label == "" {
		label = fmt.Sprintf("Q%d", rowNumber-1)
	}
	parentID := request.FolderID
	question := Question{
		ID:            "q_" + randomToken(16),
		ExamID:        request.ExamID,
		QuestionLabel: &label,
		Content:       content,
		Options:       parseCsvOptions(values["options"]),
		Answer:        strings.TrimSpace(values["answer"]),
		Explanation:   strings.TrimSpace(values["explanation"]),
		QuestionType:  questionType,
		Difficulty:    difficulty,
		Tags:          parseCsvTags(values["tags"]),
		Status:        "new",
		Images:        parseCsvImages(values["images"]),
		SourceType:    sourceType,
		SourceRef:     &sourceRef,
		ParentID:      parentID,
		CreatedAt:     nowISO(),
		UpdatedAt:     nowISO(),
	}
	s.state.Questions = append(s.state.Questions, question)
	index := len(s.state.Questions) - 1
	if err := s.syncQuestionResourceLocked(index); err != nil {
		s.state.Questions = s.state.Questions[:index]
		return "", err
	}
	existingByHash[hash] = index
	return "created", nil
}

func readCsvTextFile(filePath string) (string, string, error) {
	if strings.TrimSpace(filePath) == "" {
		return "", "", errors.New("file_path is required")
	}
	if isVirtualCsvPath(filePath) {
		return "", "", errors.New("virtual CSV paths are not available in the Go bridge yet")
	}
	filePath = normalizeCsvLocalPath(filePath)
	if hasParentTraversal(filePath) {
		return "", "", errors.New("file_path must not contain '..'")
	}
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", err
	}
	if len(bytes) >= 3 && bytes[0] == 0xEF && bytes[1] == 0xBB && bytes[2] == 0xBF {
		return string(bytes[3:]), "UTF-8 BOM", nil
	}
	if utf8.Valid(bytes) {
		return string(bytes), "UTF-8", nil
	}
	if decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(bytes); err == nil && utf8.Valid(decoded) {
		return string(decoded), "GBK", nil
	}
	if decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(bytes); err == nil && utf8.Valid(decoded) {
		return string(decoded), "GB18030", nil
	}
	return "", "", errors.New("unsupported CSV encoding; expected UTF-8, UTF-8 BOM, GBK, or GB18030")
}

func isVirtualCsvPath(filePath string) bool {
	lower := strings.ToLower(strings.TrimSpace(filePath))
	if strings.HasPrefix(lower, "content://") || strings.HasPrefix(lower, "ph://") {
		return true
	}
	if strings.HasPrefix(lower, "file://") {
		parsed, err := url.Parse(filePath)
		return err != nil || parsed.Path == ""
	}
	return false
}

func normalizeCsvLocalPath(filePath string) string {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(filePath)), "file://") {
		return filePath
	}
	parsed, err := url.Parse(filePath)
	if err != nil {
		return filePath
	}
	pathValue := parsed.Path
	if runtime.GOOS == "windows" && strings.HasPrefix(pathValue, "/") && len(pathValue) >= 3 && pathValue[2] == ':' {
		pathValue = pathValue[1:]
	}
	if pathValue == "" {
		return filePath
	}
	return filepath.FromSlash(pathValue)
}

func hasParentTraversal(filePath string) bool {
	cleaned := filepath.Clean(filePath)
	for _, part := range strings.FieldsFunc(cleaned, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		if part == ".." {
			return true
		}
	}
	return false
}

func trimCsvRecord(record []string) []string {
	out := make([]string, len(record))
	for index, value := range record {
		out[index] = strings.TrimSpace(value)
	}
	return out
}

func validateCsvFieldMapping(headers []string, mapping map[string]string) error {
	if len(mapping) == 0 {
		return errors.New("field_mapping is required")
	}
	headerSet := map[string]bool{}
	for _, header := range headers {
		headerSet[header] = true
	}
	hasContent := false
	seenTargets := map[string]bool{}
	for csvColumn, targetField := range mapping {
		if !headerSet[csvColumn] {
			return fmt.Errorf("CSV column does not exist: %s", csvColumn)
		}
		if !csvImportTargetFields[targetField] {
			return fmt.Errorf("unsupported target field: %s", targetField)
		}
		if seenTargets[targetField] {
			return fmt.Errorf("target field is mapped more than once: %s", targetField)
		}
		seenTargets[targetField] = true
		if targetField == "content" {
			hasContent = true
		}
	}
	if !hasContent {
		return errors.New("field_mapping must include content")
	}
	return nil
}

func mapCsvFields(headers []string, record []string, mapping map[string]string) map[string]string {
	values := map[string]string{}
	for column, target := range mapping {
		for index, header := range headers {
			if header == column && index < len(record) {
				values[target] = strings.TrimSpace(record[index])
				break
			}
		}
	}
	return values
}

func contentHash(content string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(content), ""))
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:16])
}

func applyCsvOverwrite(question Question, values map[string]string) Question {
	if value := strings.TrimSpace(values["content"]); value != "" {
		question.Content = value
	}
	if value := strings.TrimSpace(values["question_label"]); value != "" {
		question.QuestionLabel = &value
	}
	if value, ok := values["answer"]; ok {
		question.Answer = strings.TrimSpace(value)
	}
	if value, ok := values["explanation"]; ok {
		question.Explanation = strings.TrimSpace(value)
	}
	if value := strings.TrimSpace(values["question_type"]); value != "" {
		question.QuestionType = parseCsvQuestionType(value)
	}
	if value := strings.TrimSpace(values["difficulty"]); value != "" {
		question.Difficulty = parseCsvDifficulty(value)
	}
	if value := strings.TrimSpace(values["options"]); value != "" {
		question.Options = parseCsvOptions(value)
	}
	if value := strings.TrimSpace(values["tags"]); value != "" {
		question.Tags = parseCsvTags(value)
	}
	if value := strings.TrimSpace(values["images"]); value != "" {
		question.Images = parseCsvImages(value)
	}
	return question
}

func applyCsvMerge(question Question, values map[string]string) Question {
	if question.Answer == "" {
		if value, ok := values["answer"]; ok {
			question.Answer = strings.TrimSpace(value)
		}
	}
	if question.Explanation == "" {
		if value, ok := values["explanation"]; ok {
			question.Explanation = strings.TrimSpace(value)
		}
	}
	if len(question.Options) == 0 {
		if value, ok := values["options"]; ok {
			question.Options = parseCsvOptions(value)
		}
	}
	if len(question.Tags) == 0 {
		if value, ok := values["tags"]; ok {
			question.Tags = parseCsvTags(value)
		}
	}
	if len(question.Images) == 0 {
		if value, ok := values["images"]; ok {
			question.Images = parseCsvImages(value)
		}
	}
	if question.QuestionLabel == nil || strings.TrimSpace(*question.QuestionLabel) == "" {
		if value := strings.TrimSpace(values["question_label"]); value != "" {
			question.QuestionLabel = &value
		}
	}
	return question
}

func parseCsvOptions(value string) []QuestionOption {
	value = strings.TrimSpace(value)
	if value == "" {
		return []QuestionOption{}
	}
	var parsed []QuestionOption
	if strings.HasPrefix(value, "[") && json.Unmarshal([]byte(value), &parsed) == nil {
		return nonNilOptions(parsed)
	}
	separators := []string{";", "\n", "|"}
	for _, separator := range separators {
		parts := strings.Split(value, separator)
		if len(parts) < 2 {
			continue
		}
		options := make([]QuestionOption, 0, len(parts))
		for index, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			key, content := parseCsvOptionPart(part, index)
			options = append(options, QuestionOption{Key: key, Content: content})
		}
		if len(options) > 0 {
			return options
		}
	}
	key, content := parseCsvOptionPart(value, 0)
	return []QuestionOption{{Key: key, Content: content}}
}

func parseCsvOptionPart(part string, index int) (string, string) {
	trimmed := strings.TrimSpace(part)
	if len(trimmed) >= 2 {
		first := strings.ToUpper(trimmed[:1])
		rest := strings.TrimSpace(trimmed[1:])
		if first >= "A" && first <= "Z" {
			rest = strings.TrimLeft(rest, ".、:：)） \t")
			if rest != "" {
				return first, rest
			}
		}
	}
	return string(rune('A' + index)), trimmed
}

func parseCsvQuestionType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "single_choice", "single", "单选", "单选题":
		return "single_choice"
	case "multiple_choice", "multiple", "多选", "多选题":
		return "multiple_choice"
	case "indefinite_choice", "indefinite", "不定项", "不定项选择题":
		return "indefinite_choice"
	case "fill_blank", "fill", "填空", "填空题":
		return "fill_blank"
	case "short_answer", "short", "简答", "简答题":
		return "short_answer"
	case "essay", "论述", "论述题":
		return "essay"
	case "calculation", "calc", "计算", "计算题":
		return "calculation"
	case "proof", "证明", "证明题":
		return "proof"
	default:
		return "other"
	}
}

func parseCsvDifficulty(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "easy", "1", "简单":
		return "easy"
	case "hard", "3", "困难", "难":
		return "hard"
	case "very_hard", "4", "极难":
		return "very_hard"
	case "medium", "2", "中等":
		return "medium"
	default:
		return "medium"
	}
}

func parseCsvTags(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}
	}
	var parsed []string
	if strings.HasPrefix(value, "[") && json.Unmarshal([]byte(value), &parsed) == nil {
		return nonNilStrings(cleanStringSlice(parsed))
	}
	for _, separator := range []string{",", ";", "|", "、"} {
		if strings.Contains(value, separator) {
			return cleanStringSlice(strings.Split(value, separator))
		}
	}
	return []string{value}
}

func parseCsvImages(value string) []QuestionImage {
	value = strings.TrimSpace(value)
	if value == "" {
		return []QuestionImage{}
	}
	var images []QuestionImage
	if strings.HasPrefix(value, "[") && json.Unmarshal([]byte(value), &images) == nil {
		return nonNilImages(images)
	}
	return []QuestionImage{}
}

func cleanStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func defaultCsvExportFields(includeAnswers bool) []string {
	fields := []string{
		"content",
		"question_type",
		"options",
		"answer",
		"explanation",
		"difficulty",
		"tags",
		"question_label",
	}
	if includeAnswers {
		fields = append(fields, "user_answer", "is_correct", "attempt_count", "correct_count", "status")
	}
	return fields
}

func validateCsvExportFields(fields []string) error {
	for _, field := range fields {
		if !isCsvExportableField(field) {
			return fmt.Errorf("unsupported export field: %s", field)
		}
	}
	return nil
}

func isCsvExportableField(field string) bool {
	for _, pair := range csvExportableFields {
		if pair[0] == field {
			return true
		}
	}
	return false
}

func csvExportDisplayName(field string) string {
	for _, pair := range csvExportableFields {
		if pair[0] == field {
			return pair[1]
		}
	}
	return field
}

func csvFieldValue(question Question, field string) string {
	switch field {
	case "content":
		return question.Content
	case "question_type":
		return formatCsvQuestionType(question.QuestionType)
	case "options":
		return formatCsvOptions(question.Options)
	case "answer":
		return question.Answer
	case "explanation":
		return question.Explanation
	case "difficulty":
		return formatCsvDifficulty(question.Difficulty)
	case "tags":
		return strings.Join(question.Tags, ",")
	case "images":
		bytes, err := json.Marshal(question.Images)
		if err != nil {
			return "[]"
		}
		return string(bytes)
	case "question_label":
		if question.QuestionLabel == nil {
			return ""
		}
		return *question.QuestionLabel
	case "user_answer":
		return question.UserAnswer
	case "is_correct":
		if question.IsCorrect == nil {
			return ""
		}
		if *question.IsCorrect {
			return "正确"
		}
		return "错误"
	case "attempt_count":
		return fmt.Sprintf("%d", question.AttemptCount)
	case "correct_count":
		return fmt.Sprintf("%d", question.CorrectCount)
	case "status":
		return formatCsvStatus(question.Status)
	case "is_favorite":
		if question.IsFavorite {
			return "是"
		}
		return "否"
	case "user_note":
		return question.UserNote
	case "created_at":
		return question.CreatedAt
	case "updated_at":
		return question.UpdatedAt
	default:
		return ""
	}
}

func formatCsvOptions(options []QuestionOption) string {
	parts := make([]string, 0, len(options))
	for _, option := range options {
		key := strings.TrimSpace(option.Key)
		if key == "" {
			key = "-"
		}
		parts = append(parts, fmt.Sprintf("%s. %s", key, strings.TrimSpace(option.Content)))
	}
	return strings.Join(parts, "; ")
}

func formatCsvQuestionType(value string) string {
	switch value {
	case "single_choice":
		return "单选题"
	case "multiple_choice":
		return "多选题"
	case "indefinite_choice":
		return "不定项选择题"
	case "fill_blank":
		return "填空题"
	case "short_answer":
		return "简答题"
	case "essay":
		return "论述题"
	case "calculation":
		return "计算题"
	case "proof":
		return "证明题"
	default:
		return "其他"
	}
}

func formatCsvDifficulty(value string) string {
	switch value {
	case "easy":
		return "简单"
	case "hard":
		return "困难"
	case "very_hard":
		return "极难"
	default:
		return "中等"
	}
}

func formatCsvStatus(value string) string {
	switch value {
	case "in_progress":
		return "学习中"
	case "mastered":
		return "已掌握"
	case "review":
		return "需复习"
	default:
		return "新题"
	}
}

func neutralizeCsvFormulaCells(row []string) []string {
	out := make([]string, len(row))
	for index, value := range row {
		if startsWithCsvFormulaPrefix(value) {
			out[index] = "\t" + value
		} else {
			out[index] = value
		}
	}
	return out
}

func startsWithCsvFormulaPrefix(value string) bool {
	if value == "" {
		return false
	}
	switch []rune(value)[0] {
	case '=', '+', '-', '@', '\t', '\r', '\n', '\uFF1D', '\uFF0B', '\uFF0D', '\uFF20':
		return true
	default:
		return false
	}
}

func normalizeCsvEncoding(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "gbk":
		return "gbk"
	case "utf8_bom", "utf-8-bom", "utf_8_bom":
		return "utf8_bom"
	default:
		return "utf8"
	}
}
