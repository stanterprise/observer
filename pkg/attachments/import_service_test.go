package attachments

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/pkg/importer"
	"github.com/stanterprise/observer/pkg/storage"
)

func TestImportServiceMaterializesInlineAndExternalAttachments(t *testing.T) {
	driver := &fakeDriver{}
	attempt := &m.TestAttempt{ID: "attempt", TestID: "test", RunID: "run"}
	bundle := &importer.RunImportBundle{
		Run: &m.TestRun{ID: "run"}, Attempts: []*m.TestAttempt{attempt},
		Attachments: []*importer.Attachment{
			{AttemptID: "attempt", Name: "small.txt", MimeType: "text/plain", Size: 5, Available: true, Open: stringOpener("hello")},
			{AttemptID: "attempt", Name: "large.zip", MimeType: "application/zip", Size: 20, Available: true, Open: stringOpener("this is external data")},
			{AttemptID: "attempt", Name: "missing.png", MimeType: "image/png", Available: false},
		},
	}
	keys, err := NewImportService(driver, 10).Materialize(context.Background(), bundle)
	if err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}
	if len(keys) != 1 || keys[0] != "key-1" {
		t.Fatalf("keys = %#v", keys)
	}
	if len(attempt.Attachments) != 3 {
		t.Fatalf("attachments = %#v", attempt.Attachments)
	}
	if attempt.Attachments[0]["storage"] != "inline" || attempt.Attachments[1]["storage"] != "fake" || attempt.Attachments[2]["storage"] != "missing" {
		t.Fatalf("attachments = %#v", attempt.Attachments)
	}
	if len(bundle.AttachmentRows) != 1 || bundle.AttachmentRows[0].StorageKey != "key-1" {
		t.Fatalf("rows = %#v", bundle.AttachmentRows)
	}
	NewImportService(driver, 10).Cleanup(context.Background(), keys)
	if len(driver.deleted) != 1 || driver.deleted[0] != "key-1" {
		t.Fatalf("deleted = %#v", driver.deleted)
	}
}

func stringOpener(value string) importer.ContentOpener {
	return func(context.Context) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(value)), nil }
}

type fakeDriver struct{ deleted []string }

func (*fakeDriver) Upload(_ context.Context, name, mime string, content io.Reader) (*storage.AttachmentMetadata, error) {
	data, _ := io.ReadAll(content)
	return &storage.AttachmentMetadata{Name: name, MimeType: mime, Size: int64(len(data)), StorageKey: "key-1", StorageURI: "fake://key-1"}, nil
}
func (*fakeDriver) Download(context.Context, string) (io.ReadCloser, error) { return nil, nil }
func (*fakeDriver) GetMetadata(context.Context, string) (*storage.AttachmentMetadata, error) {
	return nil, nil
}
func (d *fakeDriver) Delete(_ context.Context, key string) error {
	d.deleted = append(d.deleted, key)
	return nil
}
func (*fakeDriver) GetSignedURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}
func (*fakeDriver) Name() string { return "fake" }
func (*fakeDriver) Close() error { return nil }
