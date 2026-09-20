package playwrightblob

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/stanterprise/observer/pkg/importer"
)

const blobVersion = 2

var reportType = importer.ReportType{Producer: "playwright", Format: "blob", Version: "v2"}

type Parser struct {
	limits importer.Limits
}

func DefaultLimits(externalAttachments bool) importer.Limits {
	return importer.Limits{
		FileExtensions:        []string{".zip"},
		MultipleFiles:         true,
		MaxFiles:              32,
		MaxRequestBytes:       512 << 20,
		MaxUncompressedBytes:  2 << 30,
		MaxJSONLRecordBytes:   32 << 20,
		InlineAttachmentBytes: 100 << 10,
		ExternalAttachments:   externalAttachments,
	}
}

func New(limits importer.Limits) *Parser {
	if limits.MaxFiles <= 0 {
		limits = DefaultLimits(limits.ExternalAttachments)
	}
	return &Parser{limits: limits}
}

func (p *Parser) ReportType() importer.ReportType { return reportType }
func (p *Parser) Limits() importer.Limits         { return p.limits }

func (p *Parser) Probe(ctx context.Context, files []importer.SourceFile) (importer.ProbeResult, error) {
	if len(files) != 1 {
		return importer.ProbeResult{}, importer.NewError("invalid_file_count", "probe requires exactly one report", nil)
	}
	opened, err := p.openBlob(ctx, files[0])
	if err != nil {
		return importer.ProbeResult{}, err
	}
	defer opened.close()
	metadata, _, err := p.readMetadata(opened.report)
	if err != nil {
		return importer.ProbeResult{}, err
	}
	if metadata.Version != blobVersion {
		return importer.ProbeResult{}, versionError(metadata.Version)
	}
	return importer.ProbeResult{ReportType: reportType, Metadata: map[string]interface{}{
		"blobVersion": metadata.Version,
		"userAgent":   metadata.UserAgent,
	}}, nil
}

func (p *Parser) Parse(ctx context.Context, files []importer.SourceFile) (*importer.RunImportBundle, error) {
	if len(files) == 0 {
		return nil, importer.NewError("missing_files", "at least one report file is required", nil)
	}
	if len(files) > p.limits.MaxFiles {
		return nil, importer.NewError("upload_too_large", fmt.Sprintf("too many report files: got %d, maximum is %d", len(files), p.limits.MaxFiles), nil)
	}

	sources := append([]importer.SourceFile(nil), files...)
	for i := range sources {
		if sources[i].SHA256 == "" {
			digest, size, err := hashFile(ctx, sources[i].Path)
			if err != nil {
				return nil, importer.NewError("invalid_archive", "failed to hash report", err)
			}
			sources[i].SHA256 = digest
			if sources[i].Size == 0 {
				sources[i].Size = size
			}
		}
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].SHA256 < sources[j].SHA256 })
	for i := 1; i < len(sources); i++ {
		if sources[i].SHA256 == sources[i-1].SHA256 {
			return nil, importer.NewError("duplicate_report", "the same report was provided more than once", nil)
		}
	}

	sourceDigest := combinedDigest(sources)
	runID := "pwb-" + sourceDigest[:32]
	usedTests := make(map[string]struct{})
	opened := make([]*openedBlob, 0, len(sources))
	parsed := make([]*blobRun, 0, len(sources))
	cleanup := func() {
		for _, blob := range opened {
			_ = blob.close()
		}
	}

	for _, source := range sources {
		if err := ctx.Err(); err != nil {
			cleanup()
			return nil, err
		}
		blob, err := p.openBlob(ctx, source)
		if err != nil {
			cleanup()
			return nil, err
		}
		opened = append(opened, blob)
		result, err := p.parseBlob(ctx, runID, blob, usedTests)
		if err != nil {
			cleanup()
			return nil, err
		}
		parsed = append(parsed, result)
	}

	if err := validateBlobSet(parsed); err != nil {
		cleanup()
		return nil, err
	}
	bundle := mergeRuns(runID, sourceDigest, parsed)
	bundle.Cleanup = cleanup
	return bundle, nil
}

type openedBlob struct {
	source  importer.SourceFile
	file    *os.File
	reader  *zip.Reader
	report  *zip.File
	entries map[string]*zip.File
}

func (b *openedBlob) close() error {
	if b == nil || b.file == nil {
		return nil
	}
	err := b.file.Close()
	b.file = nil
	return err
}

func (p *Parser) openBlob(ctx context.Context, source importer.SourceFile) (*openedBlob, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.Open(source.Path)
	if err != nil {
		return nil, importer.NewError("invalid_archive", "failed to open report archive", err)
	}
	stat, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, importer.NewError("invalid_archive", "failed to inspect report archive", err)
	}
	zr, err := zip.NewReader(f, stat.Size())
	if err != nil {
		_ = f.Close()
		return nil, importer.NewError("invalid_archive", "file is not a valid ZIP archive", err)
	}

	entries := make(map[string]*zip.File, len(zr.File))
	var report *zip.File
	var total uint64
	for _, entry := range zr.File {
		name, err := safeEntryName(entry)
		if err != nil {
			_ = f.Close()
			return nil, importer.NewError("invalid_archive", err.Error(), nil)
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		if _, exists := entries[name]; exists {
			_ = f.Close()
			return nil, importer.NewError("invalid_archive", "archive contains duplicate entry names", nil)
		}
		entries[name] = entry
		total += entry.UncompressedSize64
		if p.limits.MaxUncompressedBytes > 0 && total > uint64(p.limits.MaxUncompressedBytes) {
			_ = f.Close()
			return nil, importer.NewError("upload_too_large", "archive exceeds the uncompressed size limit", nil)
		}
		if name == "report.jsonl" {
			report = entry
		}
	}
	if report == nil {
		_ = f.Close()
		return nil, importer.NewError("missing_report_jsonl", "archive does not contain root-level report.jsonl", nil)
	}
	return &openedBlob{source: source, file: f, reader: zr, report: report, entries: entries}, nil
}

func safeEntryName(entry *zip.File) (string, error) {
	name := entry.Name
	if name == "" || strings.ContainsRune(name, 0) || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("archive contains an unsafe entry path")
	}
	compareName := name
	if entry.FileInfo().IsDir() {
		compareName = strings.TrimSuffix(name, "/")
	}
	clean := path.Clean(compareName)
	if clean == "." || clean != compareName || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("archive contains an unsafe entry path")
	}
	if entry.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("archive contains a symbolic link")
	}
	return clean, nil
}

func (p *Parser) readMetadata(report *zip.File) (blobMetadata, io.ReadCloser, error) {
	rc, err := report.Open()
	if err != nil {
		return blobMetadata{}, nil, importer.NewError("invalid_archive", "failed to open report.jsonl", err)
	}
	reader := bufio.NewReaderSize(rc, 64<<10)
	line, err := readBoundedLine(reader, p.limits.MaxJSONLRecordBytes)
	if err != nil {
		_ = rc.Close()
		return blobMetadata{}, nil, importer.NewError("invalid_report_jsonl", "failed to read blob metadata", err)
	}
	var event jsonEvent
	if err := json.Unmarshal(line, &event); err != nil || event.Method != "onBlobReportMetadata" {
		_ = rc.Close()
		return blobMetadata{}, nil, importer.NewError("missing_blob_metadata", "first report event must be onBlobReportMetadata", err)
	}
	var metadata blobMetadata
	if err := json.Unmarshal(event.Params, &metadata); err != nil {
		_ = rc.Close()
		return blobMetadata{}, nil, importer.NewError("invalid_report_jsonl", "invalid blob metadata", err)
	}
	return metadata, &bufferedReadCloser{Reader: reader, Closer: rc}, nil
}

type bufferedReadCloser struct {
	*bufio.Reader
	io.Closer
}

func readBoundedLine(reader *bufio.Reader, max int64) ([]byte, error) {
	if max <= 0 {
		max = 32 << 20
	}
	var line []byte
	for {
		fragment, err := reader.ReadSlice('\n')
		if int64(len(line)+len(fragment)) > max {
			return nil, fmt.Errorf("JSONL record exceeds %d bytes", max)
		}
		line = append(line, fragment...)
		switch {
		case err == nil:
			return bytes.TrimSuffix(line, []byte{'\n'}), nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF) && len(line) > 0:
			return line, nil
		default:
			return nil, err
		}
	}
}

func versionError(version int) error {
	return &importer.Error{
		Code:    "report_version_mismatch",
		Message: fmt.Sprintf("Playwright blob schema version %d is not supported by the v2 endpoint", version),
		Details: map[string]interface{}{"requestedVersion": "v2", "detectedVersion": fmt.Sprintf("v%d", version)},
	}
}

func combinedDigest(files []importer.SourceFile) string {
	h := sha256.New()
	_, _ = io.WriteString(h, reportType.Key())
	for _, file := range files {
		_, _ = io.WriteString(h, "\x00"+file.SHA256)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func hashFile(ctx context.Context, filePath string) (string, int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := copyWithContext(ctx, h, f)
	if err != nil {
		return "", n, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 64<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, readErr := src.Read(buf)
		if n > 0 {
			written, writeErr := dst.Write(buf[:n])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return total, nil
			}
			return total, readErr
		}
	}
}

func hashID(prefix string, parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = io.WriteString(h, part)
		_, _ = io.WriteString(h, "\x00")
	}
	return prefix + hex.EncodeToString(h.Sum(nil))[:32]
}

func durationNanos(ms float64) int64 { return int64(ms * float64(time.Millisecond)) }

func locationString(location *jsonLocation) string {
	if location == nil || location.File == "" {
		return ""
	}
	if location.Line <= 0 {
		return location.File
	}
	if location.Column <= 0 {
		return fmt.Sprintf("%s:%d", location.File, location.Line)
	}
	return fmt.Sprintf("%s:%d:%d", location.File, location.Line, location.Column)
}

func decodeBase64Opener(value string) (importer.ContentOpener, int64, error) {
	// Validate now while retaining streaming decode for materialization.
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(value))
	decodedSize, err := io.Copy(io.Discard, decoder)
	if err != nil {
		return nil, 0, err
	}
	return func(ctx context.Context) (io.ReadCloser, error) {
		return &contextReadCloser{ctx: ctx, reader: base64.NewDecoder(base64.StdEncoding, strings.NewReader(value))}, nil
	}, decodedSize, nil
}

type contextReadCloser struct {
	ctx    context.Context
	reader io.Reader
	closer io.Closer
}

func (r *contextReadCloser) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func (r *contextReadCloser) Close() error {
	if r.closer != nil {
		return r.closer.Close()
	}
	return nil
}

func zipOpener(entry *zip.File) importer.ContentOpener {
	return func(ctx context.Context) (io.ReadCloser, error) {
		rc, err := entry.Open()
		if err != nil {
			return nil, err
		}
		return &contextReadCloser{ctx: ctx, reader: rc, closer: rc}, nil
	}
}
