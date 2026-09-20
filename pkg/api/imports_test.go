package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	pgRepo "github.com/stanterprise/observer/internal/repository/postgres"
	attachmentimport "github.com/stanterprise/observer/pkg/attachments"
	"github.com/stanterprise/observer/pkg/importer"
	"github.com/stanterprise/observer/pkg/importer/playwrightblob"
)

func TestImportHandlerCapabilitiesAndIdempotentUpload(t *testing.T) {
	_, db := setupPostgresHandler(t)
	repo := pgRepo.NewPostgresRepository(db, nil)
	parser := playwrightblob.New(playwrightblob.DefaultLimits(false))
	registry, err := importer.NewRegistry(parser)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewImportHandler(registry, repo, attachmentimport.NewImportService(nil, parser.Limits().InlineAttachmentBytes), nil, t.TempDir(), 1)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	capabilityResponse := httptest.NewRecorder()
	router.ServeHTTP(capabilityResponse, httptest.NewRequest(http.MethodGet, "/api/v1/imports", nil))
	if capabilityResponse.Code != http.StatusOK {
		t.Fatalf("capabilities status = %d", capabilityResponse.Code)
	}
	if !bytes.Contains(capabilityResponse.Body.Bytes(), []byte(`"uploadPath":"/v1/imports/playwright/blob/v2"`)) {
		t.Fatalf("capabilities = %s", capabilityResponse.Body.String())
	}

	archive := minimalPlaywrightBlob(t)
	first := uploadRequest(t, archive, "Release import")
	firstResponse := httptest.NewRecorder()
	router.ServeHTTP(firstResponse, first)
	if firstResponse.Code != http.StatusCreated {
		t.Fatalf("first status=%d body=%s", firstResponse.Code, firstResponse.Body.String())
	}
	var firstPayload struct {
		Data struct {
			RunID   string `json:"runId"`
			Created bool   `json:"created"`
		} `json:"data"`
	}
	if err := json.Unmarshal(firstResponse.Body.Bytes(), &firstPayload); err != nil {
		t.Fatal(err)
	}
	if firstPayload.Data.RunID == "" || !firstPayload.Data.Created {
		t.Fatalf("payload = %#v", firstPayload)
	}

	secondResponse := httptest.NewRecorder()
	router.ServeHTTP(secondResponse, uploadRequest(t, archive, "Different ignored name"))
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("second status=%d body=%s", secondResponse.Code, secondResponse.Body.String())
	}
	var runCount int64
	if err := db.Table("runs").Count(&runCount).Error; err != nil {
		t.Fatal(err)
	}
	if runCount != 1 {
		t.Fatalf("run count = %d", runCount)
	}
}

func uploadRequest(t *testing.T, archive, name string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("name", name)
	part, err := writer.CreateFormFile("reports", "report.zip")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(archive)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, file); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/imports/playwright/blob/v2", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func minimalPlaywrightBlob(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "report.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zipWriter := zip.NewWriter(file)
	report, err := zipWriter.Create("report.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	testCase := map[string]interface{}{"testId": "test-1", "title": "works", "location": map[string]interface{}{"file": "a.spec.ts", "line": 1, "column": 1}, "retries": 0}
	suite := map[string]interface{}{"title": "a.spec.ts", "location": map[string]interface{}{"file": "a.spec.ts", "line": 1, "column": 1}, "entries": []interface{}{testCase}}
	events := []map[string]interface{}{
		{"method": "onBlobReportMetadata", "params": map[string]interface{}{"version": 2, "name": "blob name"}},
		{"method": "onConfigure", "params": map[string]interface{}{"config": map[string]interface{}{"rootDir": "/repo", "version": "1.50.0"}}},
		{"method": "onProject", "params": map[string]interface{}{"project": map[string]interface{}{"name": "chromium", "timeout": 30000, "suites": []interface{}{suite}}}},
		{"method": "onTestBegin", "params": map[string]interface{}{"testId": "test-1", "result": map[string]interface{}{"id": "result-1", "retry": 0, "startTime": 1700000000000}}},
		{"method": "onTestEnd", "params": map[string]interface{}{"test": map[string]interface{}{"testId": "test-1", "expectedStatus": "passed"}, "result": map[string]interface{}{"id": "result-1", "duration": 12, "status": "passed", "errors": []interface{}{}}}},
		{"method": "onEnd", "params": map[string]interface{}{"result": map[string]interface{}{"status": "passed", "startTime": 1700000000000, "duration": 12}}},
	}
	for _, event := range events {
		line, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := report.Write(append(line, '\n')); err != nil {
			t.Fatal(err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
