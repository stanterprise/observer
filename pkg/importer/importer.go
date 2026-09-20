package importer

import (
	"context"
	"fmt"
	"io"

	m "github.com/stanterprise/observer/internal/models"
)

type Error struct {
	Code    string
	Message string
	Details map[string]interface{}
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func (e *Error) Unwrap() error { return e.Err }

func NewError(code, message string, err error) error {
	if message == "" && err != nil {
		message = fmt.Sprintf("%v", err)
	}
	return &Error{Code: code, Message: message, Err: err}
}

// ReportType identifies one versioned report adapter.
type ReportType struct {
	Producer string `json:"producer"`
	Format   string `json:"format"`
	Version  string `json:"reportVersion"`
}

func (t ReportType) Key() string {
	return t.Producer + "/" + t.Format + "/" + t.Version
}

// Limits are advertised by an importer and enforced by the HTTP layer/parser.
type Limits struct {
	FileExtensions        []string `json:"fileExtensions"`
	MultipleFiles         bool     `json:"multipleFiles"`
	MaxFiles              int      `json:"maxFiles"`
	MaxRequestBytes       int64    `json:"maxRequestBytes"`
	MaxUncompressedBytes  int64    `json:"maxUncompressedBytes"`
	MaxJSONLRecordBytes   int64    `json:"maxJsonlRecordBytes"`
	InlineAttachmentBytes int64    `json:"inlineAttachmentBytes"`
	ExternalAttachments   bool     `json:"externalAttachments"`
}

type Capability struct {
	ReportType
	UploadPath string `json:"uploadPath"`
	Limits
}

// SourceFile is a request-scoped, seekable upload owned by the API handler.
type SourceFile struct {
	Name   string
	Path   string
	Size   int64
	SHA256 string
}

type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
}

// ContentOpener opens attachment content. Callers must close the returned reader.
type ContentOpener func(context.Context) (io.ReadCloser, error)

// Attachment is a normalized attachment waiting to be stored.
type Attachment struct {
	AttemptID  string
	StepID     string
	Name       string
	MimeType   string
	Size       int64
	Checksum   string
	SourcePath string
	Available  bool
	Open       ContentOpener
}

// RunImportBundle is the format-neutral result consumed by persistence.
type RunImportBundle struct {
	ReportType     ReportType
	SourceDigest   string
	Run            *m.TestRun
	Executions     []*m.RunExecution
	Suites         []*m.Suite
	Tests          []*m.Test
	Attempts       []*m.TestAttempt
	Attachments    []*Attachment
	AttachmentRows []*m.Attachment
	Warnings       []Warning
	Cleanup        func() `json:"-"`
}

type ProbeResult struct {
	ReportType ReportType
	Metadata   map[string]interface{}
}

type Importer interface {
	ReportType() ReportType
	Limits() Limits
	Probe(context.Context, []SourceFile) (ProbeResult, error)
	Parse(context.Context, []SourceFile) (*RunImportBundle, error)
}
