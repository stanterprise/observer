package playwrightblob

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stanterprise/observer/pkg/importer"
)

func TestParseNormalizesRetriesExpectedFailuresAndAttachments(t *testing.T) {
	project := jsonProject{Name: "chromium", TestDir: "/repo/tests", Timeout: 30_000}
	tests := []json.RawMessage{
		mustJSON(t, jsonTestCase{TestID: "flaky-test", Title: "eventually works", Location: jsonLocation{File: "a.spec.ts", Line: 3, Column: 1}, Retries: 1}),
		mustJSON(t, jsonTestCase{TestID: "expected-failure", Title: "known issue", Location: jsonLocation{File: "a.spec.ts", Line: 9, Column: 1}}),
		mustJSON(t, jsonTestCase{TestID: "not-run", Title: "not selected", Location: jsonLocation{File: "a.spec.ts", Line: 12, Column: 1}}),
	}
	project.Suites = []jsonSuite{{Title: "a.spec.ts", Location: &jsonLocation{File: "a.spec.ts", Line: 1, Column: 1}, Entries: tests}}
	events := []testEvent{
		{"onBlobReportMetadata", blobMetadata{Version: 2, UserAgent: "Playwright/1.50.0", Name: "CI run"}},
		{"onConfigure", configureParams{Config: jsonConfig{RootDir: "/repo", Version: "1.50.0", Workers: 2}}},
		{"onProject", projectParams{Project: project}},
		{"onTestBegin", testBeginParams{TestID: "flaky-test", Result: jsonResultStart{ID: "r1", Retry: 0, StartTime: 1_700_000_000_000}}},
		{"onStepBegin", stepBeginParams{TestID: "flaky-test", ResultID: "r1", Step: jsonStepStart{ID: "s1", Title: "click", Category: "pw:api", StartTime: 1_700_000_000_001}}},
		{"onStepEnd", stepEndParams{TestID: "flaky-test", ResultID: "r1", Step: jsonStepEnd{ID: "s1", Duration: 5, Error: &jsonError{Message: "locator failed"}}}},
		{"onTestEnd", testEndParams{Test: jsonTestEnd{TestID: "flaky-test", ExpectedStatus: "passed"}, Result: jsonResultEnd{ID: "r1", Duration: 10, Status: "failed", Errors: []jsonError{{Message: "first failure", Stack: "stack"}}}}},
		{"onTestBegin", testBeginParams{TestID: "flaky-test", Result: jsonResultStart{ID: "r2", Retry: 1, StartTime: 1_700_000_000_020}}},
		{"onStdIO", stdIOParams{Type: "stdout", TestID: "flaky-test", ResultID: "r2", Data: "hello"}},
		{"onStepBegin", stepBeginParams{TestID: "flaky-test", ResultID: "r2", Step: jsonStepStart{ID: "s2", Title: "attach", Category: "test.attach", StartTime: 1_700_000_000_021}}},
		{"onAttach", attachParams{TestID: "flaky-test", ResultID: "r2", Attachments: []jsonAttachment{{Name: "note.txt", ContentType: "text/plain", Base64: "aGVsbG8="}, {Name: "missing.png", ContentType: "image/png", Path: "resources/missing.png"}}}},
		{"onStepEnd", stepEndParams{TestID: "flaky-test", ResultID: "r2", Step: jsonStepEnd{ID: "s2", Duration: 1, Attachments: []int{0}}}},
		{"onTestEnd", testEndParams{Test: jsonTestEnd{TestID: "flaky-test", ExpectedStatus: "passed"}, Result: jsonResultEnd{ID: "r2", Duration: 8, Status: "passed"}}},
		{"onTestBegin", testBeginParams{TestID: "expected-failure", Result: jsonResultStart{ID: "r3", StartTime: 1_700_000_000_040}}},
		{"onTestEnd", testEndParams{Test: jsonTestEnd{TestID: "expected-failure", ExpectedStatus: "failed"}, Result: jsonResultEnd{ID: "r3", Duration: 4, Status: "failed"}}},
		{"onEnd", endParams{Result: jsonFullResult{Status: "passed", StartTime: 1_700_000_000_000, Duration: 50}}},
	}
	file := writeBlob(t, events, nil)
	parser := New(DefaultLimits(true))
	bundle, err := parser.Parse(context.Background(), []importer.SourceFile{{Name: "report.zip", Path: file}})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	defer bundle.Cleanup()

	if bundle.Run.Name != "CI run" || bundle.Run.Status != "PASSED" {
		t.Fatalf("run = %#v", bundle.Run)
	}
	if len(bundle.Tests) != 3 || len(bundle.Attempts) != 3 {
		t.Fatalf("tests=%d attempts=%d", len(bundle.Tests), len(bundle.Attempts))
	}
	statuses := map[string]string{}
	for _, test := range bundle.Tests {
		statuses[test.ExternalTestID] = test.Status
	}
	if statuses["flaky-test"] != "FLAKY" {
		t.Errorf("flaky status = %q", statuses["flaky-test"])
	}
	if statuses["expected-failure"] != "PASSED" {
		t.Errorf("expected failure status = %q", statuses["expected-failure"])
	}
	if statuses["not-run"] != "NOT_RUN" {
		t.Errorf("not-run status = %q", statuses["not-run"])
	}
	if len(bundle.Attachments) != 2 || !bundle.Attachments[0].Available || bundle.Attachments[1].Available {
		t.Fatalf("attachments = %#v", bundle.Attachments)
	}
	if bundle.Attachments[0].StepID == "" {
		t.Fatal("step attachment was not associated with its step")
	}
	if len(bundle.Warnings) != 1 || bundle.Warnings[0].Code != "missing_attachment_resource" {
		t.Fatalf("warnings = %#v", bundle.Warnings)
	}
	if bundle.Attempts[0].StepsCount != 1 || len(bundle.Attempts[1].StdOut) != 1 {
		t.Fatalf("attempt normalization incomplete")
	}
}

func TestParseRejectsIncompleteShardSet(t *testing.T) {
	metadata := blobMetadata{Version: 2, Shard: &struct {
		Total   int `json:"total"`
		Current int `json:"current"`
	}{Total: 2, Current: 1}}
	events := []testEvent{{"onBlobReportMetadata", metadata}, {"onConfigure", configureParams{Config: jsonConfig{RootDir: "/repo"}}}, {"onEnd", endParams{Result: jsonFullResult{Status: "passed", StartTime: 1, Duration: 1}}}}
	file := writeBlob(t, events, nil)
	bundle, err := New(DefaultLimits(false)).Parse(context.Background(), []importer.SourceFile{{Name: "one.zip", Path: file}})
	if bundle != nil {
		bundle.Cleanup()
	}
	var typed *importer.Error
	if !errors.As(err, &typed) || typed.Code != "incomplete_shard_set" {
		t.Fatalf("error = %#v", err)
	}
}

func TestParseRejectsUnsafeZipEntry(t *testing.T) {
	file := writeBlob(t, []testEvent{{"onBlobReportMetadata", blobMetadata{Version: 2}}}, map[string][]byte{"../escape": []byte("bad")})
	_, err := New(DefaultLimits(false)).Parse(context.Background(), []importer.SourceFile{{Name: "bad.zip", Path: file}})
	var typed *importer.Error
	if !errors.As(err, &typed) || typed.Code != "invalid_archive" {
		t.Fatalf("error = %#v", err)
	}
}

func TestNormalizeTestStatusPreservesPlaywrightExpectedSemantics(t *testing.T) {
	cases := map[string]struct{ actual, expected, want string }{
		"expected failure": {"failed", "failed", "PASSED"},
		"unexpected pass":  {"passed", "failed", "FAILED"},
		"declared skip":    {"skipped", "skipped", "SKIPPED"},
		"did not run":      {"skipped", "passed", "NOT_RUN"},
		"timeout":          {"timedOut", "passed", "TIMEDOUT"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			if got := normalizeTestStatus(test.actual, test.expected); got != test.want {
				t.Fatalf("status=%q, want %q", got, test.want)
			}
		})
	}
}

type testEvent struct {
	method string
	params interface{}
}

func writeBlob(t *testing.T, events []testEvent, resources map[string][]byte) string {
	t.Helper()
	filePath := filepath.Join(t.TempDir(), "report.zip")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	report, err := writer.Create("report.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		line, err := json.Marshal(map[string]interface{}{"method": event.method, "params": event.params})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := report.Write(append(line, '\n')); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range resources {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return filePath
}

func mustJSON(t *testing.T, value interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
