package playwrightblob

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/pkg/importer"
)

type blobRun struct {
	opened      *openedBlob
	metadata    blobMetadata
	config      jsonConfig
	execution   *m.RunExecution
	suites      []*m.Suite
	tests       []*m.Test
	attempts    []*m.TestAttempt
	attachments []*importer.Attachment
	warnings    []importer.Warning
	result      jsonFullResult
	hasEnd      bool
}

type attemptState struct {
	attempt         *m.TestAttempt
	test            *m.Test
	resultID        string
	steps           map[string]*m.StepDocument
	stepOrder       []string
	attachments     []*importer.Attachment
	stepAttachments map[int]string
}

func (p *Parser) parseBlob(ctx context.Context, runID string, blob *openedBlob, usedTests map[string]struct{}) (*blobRun, error) {
	metadata, rc, err := p.readMetadata(blob.report)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	if metadata.Version != blobVersion {
		return nil, versionError(metadata.Version)
	}

	executionID := hashID("pwbx-", runID, blob.source.SHA256)
	result := &blobRun{
		opened:   blob,
		metadata: metadata,
		execution: &m.RunExecution{
			ID: executionID, RunID: runID, Name: blob.source.Name, Status: "NOT_RUN",
			Metadata: map[string]interface{}{"sourceFile": blob.source.Name, "sourceSha256": blob.source.SHA256, "playwrightUserAgent": metadata.UserAgent},
		},
	}
	if metadata.Name != "" {
		result.execution.Name = metadata.Name
	}
	if metadata.Shard != nil {
		current, total := int32(metadata.Shard.Current), int32(metadata.Shard.Total)
		result.execution.IsShard = true
		result.execution.ShardIndex = &current
		result.execution.ShardCountExpected = &total
	}

	testsByExternal := make(map[string]*m.Test)
	attemptsByResult := make(map[string]*attemptState)
	scanner := bufio.NewReaderSize(rc, 64<<10)
	lineNumber := 1
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line, readErr := readBoundedLine(scanner, p.limits.MaxJSONLRecordBytes)
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return nil, importer.NewError("invalid_report_jsonl", fmt.Sprintf("failed to read report event at line %d", lineNumber+1), readErr)
		}
		lineNumber++
		if len(line) == 0 {
			continue
		}
		var event jsonEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, importer.NewError("invalid_report_jsonl", fmt.Sprintf("invalid JSON at line %d", lineNumber), err)
		}
		switch event.Method {
		case "onConfigure":
			var params configureParams
			if err := decodeParams(event, &params); err != nil {
				return nil, err
			}
			result.config = params.Config
			result.execution.Metadata["playwrightVersion"] = params.Config.Version
			result.execution.Metadata["workers"] = params.Config.Workers
		case "onProject":
			var params projectParams
			if err := decodeParams(event, &params); err != nil {
				return nil, err
			}
			addProject(runID, blob.source.SHA256, result, params.Project, testsByExternal, usedTests)
		case "onTestBegin":
			var params testBeginParams
			if err := decodeParams(event, &params); err != nil {
				return nil, err
			}
			test := testsByExternal[params.TestID]
			if test == nil {
				return nil, importer.NewError("invalid_report_jsonl", "onTestBegin references an unknown test", nil)
			}
			start := unixMillis(params.Result.StartTime)
			attemptID := hashID("pwba-", executionID, params.Result.ID, test.ID)
			attempt := &m.TestAttempt{
				ID: attemptID, RunID: runID, ExecutionID: executionID, TestID: test.ID,
				AttemptIndex: int32(params.Result.Retry), Status: "RUNNING", StartTime: &start,
				Metadata: map[string]interface{}{"playwrightResultId": params.Result.ID, "workerIndex": params.Result.WorkerIndex, "parallelIndex": params.Result.ParallelIndex},
			}
			state := &attemptState{attempt: attempt, test: test, resultID: params.Result.ID, steps: make(map[string]*m.StepDocument), stepAttachments: make(map[int]string)}
			attemptsByResult[params.Result.ID] = state
			result.attempts = append(result.attempts, attempt)
		case "onStepBegin":
			var params stepBeginParams
			if err := decodeParams(event, &params); err != nil {
				return nil, err
			}
			state := attemptsByResult[params.ResultID]
			if state == nil {
				return nil, importer.NewError("invalid_report_jsonl", "onStepBegin references an unknown result", nil)
			}
			started := unixMillis(params.Step.StartTime)
			stepID := hashID("pwbs-", state.attempt.ID, params.Step.ID)
			parentID := ""
			if params.Step.ParentStepID != "" {
				parentID = hashID("pwbs-", state.attempt.ID, params.Step.ParentStepID)
			}
			step := &m.StepDocument{
				ID: stepID, RunID: runID, ExecutionID: executionID, TestCaseRunID: state.test.ID,
				ParentStepID: parentID, Title: params.Step.Title, StartTime: &started, Status: "RUNNING",
				Category: params.Step.Category, Type: params.Step.Category, Location: locationString(params.Step.Location),
				RetryIndex: state.attempt.AttemptIndex, Metadata: map[string]interface{}{"playwrightStepId": params.Step.ID},
			}
			state.steps[params.Step.ID] = step
			state.stepOrder = append(state.stepOrder, params.Step.ID)
		case "onStepEnd":
			var params stepEndParams
			if err := decodeParams(event, &params); err != nil {
				return nil, err
			}
			state := attemptsByResult[params.ResultID]
			if state == nil || state.steps[params.Step.ID] == nil {
				return nil, importer.NewError("invalid_report_jsonl", "onStepEnd references an unknown step", nil)
			}
			step := state.steps[params.Step.ID]
			duration := durationNanos(params.Step.Duration)
			step.Duration = &duration
			step.Status = "PASSED"
			if params.Step.Error != nil {
				step.Status = "FAILED"
				step.Error = params.Step.Error.Message
				step.Errors = errorStrings([]jsonError{*params.Step.Error})
			}
			for _, index := range params.Step.Attachments {
				if index >= 0 {
					state.stepAttachments[index] = step.ID
					if index < len(state.attachments) {
						state.attachments[index].StepID = step.ID
					}
				}
			}
		case "onStdIO":
			var params stdIOParams
			if err := decodeParams(event, &params); err != nil {
				return nil, err
			}
			state := attemptsByResult[params.ResultID]
			if state == nil {
				continue
			}
			message := params.Data
			if params.IsBase64 {
				decoded, err := base64.StdEncoding.DecodeString(params.Data)
				if err != nil {
					return nil, importer.NewError("invalid_report_jsonl", "invalid base64 output", err)
				}
				message = string(decoded)
			}
			output := &m.OutputDocument{Message: message}
			if params.Type == "stderr" {
				state.attempt.StdErr = append(state.attempt.StdErr, output)
			} else {
				state.attempt.StdOut = append(state.attempt.StdOut, output)
			}
		case "onAttach":
			var params attachParams
			if err := decodeParams(event, &params); err != nil {
				return nil, err
			}
			state := attemptsByResult[params.ResultID]
			if state == nil {
				continue
			}
			for _, attachment := range params.Attachments {
				index := len(state.attachments)
				normalized := result.addAttachment(state, attachment, state.stepAttachments[index])
				state.attachments = append(state.attachments, normalized)
			}
		case "onTestEnd":
			var params testEndParams
			if err := decodeParams(event, &params); err != nil {
				return nil, err
			}
			state := attemptsByResult[params.Result.ID]
			if state == nil {
				return nil, importer.NewError("invalid_report_jsonl", "onTestEnd references an unknown result", nil)
			}
			finishAttempt(result, state, params)
			for _, attachment := range params.Result.Attachments {
				index := len(state.attachments)
				normalized := result.addAttachment(state, attachment, state.stepAttachments[index])
				state.attachments = append(state.attachments, normalized)
			}
		case "onError":
			var params errorParams
			if err := decodeParams(event, &params); err != nil {
				return nil, err
			}
			globalErrors, _ := result.execution.Metadata["globalErrors"].([]map[string]interface{})
			globalErrors = append(globalErrors, map[string]interface{}{"message": params.Error.Message, "stack": params.Error.Stack, "location": locationString(params.Error.Location)})
			result.execution.Metadata["globalErrors"] = globalErrors
			result.warnings = append(result.warnings, importer.Warning{Code: "playwright_global_error", Message: params.Error.Message, File: blob.source.Name})
		case "onEnd":
			var params endParams
			if err := decodeParams(event, &params); err != nil {
				return nil, err
			}
			result.result, result.hasEnd = params.Result, true
		case "onBegin", "onExit", "onTestPaused":
			// The event carries no additional domain data.
		default:
			result.warnings = append(result.warnings, importer.Warning{Code: "unknown_playwright_event", Message: "ignored event " + event.Method, File: blob.source.Name})
		}
	}

	if !result.hasEnd {
		return nil, importer.NewError("incomplete_report", "report does not contain onEnd", nil)
	}
	finalizeBlob(result, attemptsByResult)
	return result, nil
}

func decodeParams(event jsonEvent, target interface{}) error {
	if err := json.Unmarshal(event.Params, target); err != nil {
		return importer.NewError("invalid_report_jsonl", "invalid "+event.Method+" event", err)
	}
	return nil
}

func addProject(runID, sourceHash string, result *blobRun, project jsonProject, testsByExternal map[string]*m.Test, usedTests map[string]struct{}) {
	rootID := hashID("pwbu-", runID, project.Name)
	root := &m.Suite{ID: rootID, RunID: runID, ExternalSuiteID: project.Name, Name: project.Name, Type: "project", ProjectName: project.Name, Status: "NOT_RUN", Metadata: project.Metadata}
	result.suites = append(result.suites, root)
	for index := range project.Suites {
		addSuite(runID, sourceHash, result, project, &project.Suites[index], root, []string{project.Name}, testsByExternal, usedTests)
	}
}

func addSuite(runID, sourceHash string, result *blobRun, project jsonProject, input *jsonSuite, parent *m.Suite, ancestry []string, testsByExternal map[string]*m.Test, usedTests map[string]struct{}) {
	parts := append(append([]string{}, ancestry...), input.Title, locationString(input.Location))
	suiteID := hashID("pwbu-", append([]string{runID}, parts...)...)
	parentID := parent.ID
	suite := &m.Suite{ID: suiteID, RunID: runID, ExternalSuiteID: strings.Join(parts, " > "), ParentSuiteID: &parentID, Name: input.Title, Type: "suite", ProjectName: project.Name, Location: locationString(input.Location), Status: "NOT_RUN"}
	parent.SubSuiteIDs = appendUnique(parent.SubSuiteIDs, suiteID)
	result.suites = append(result.suites, suite)
	for _, raw := range input.Entries {
		var shape map[string]json.RawMessage
		if json.Unmarshal(raw, &shape) != nil {
			continue
		}
		if _, ok := shape["testId"]; ok {
			var testCase jsonTestCase
			if json.Unmarshal(raw, &testCase) != nil || testCase.TestID == "" {
				continue
			}
			testID := hashID("pwbt-", runID, testCase.TestID)
			if _, exists := usedTests[testID]; exists {
				testID = hashID("pwbt-", runID, sourceHash, testCase.TestID)
			}
			usedTests[testID] = struct{}{}
			suiteRef := suite.ID
			retries, repeat, timeout := int32(testCase.Retries), int32(testCase.RepeatEachIndex), int32(project.Timeout)
			test := &m.Test{
				ID: testID, RunID: runID, ExternalTestID: testCase.TestID, SuiteID: &suiteRef,
				Name: testCase.Title, Title: testCase.Title, Status: "NOT_RUN", Location: locationString(&testCase.Location),
				Tags: testCase.Tags, RetryCount: &retries, RetryIndex: &repeat, Timeout: &timeout,
				Metadata: map[string]interface{}{"project": project.Name, "annotations": testCase.Annotations, "repeatEachIndex": testCase.RepeatEachIndex},
			}
			result.tests = append(result.tests, test)
			testsByExternal[testCase.TestID] = test
			suite.TestCaseIDs = appendUnique(suite.TestCaseIDs, testID)
			continue
		}
		var child jsonSuite
		if json.Unmarshal(raw, &child) == nil && child.Title != "" {
			addSuite(runID, sourceHash, result, project, &child, suite, append(ancestry, input.Title), testsByExternal, usedTests)
		}
	}
}

func finishAttempt(result *blobRun, state *attemptState, params testEndParams) {
	attempt := state.attempt
	duration := durationNanos(params.Result.Duration)
	attempt.Duration = &duration
	if attempt.StartTime != nil {
		end := attempt.StartTime.Add(time.Duration(duration))
		attempt.EndTime = &end
	}
	attempt.Status = normalizeTestStatus(params.Result.Status, params.Test.ExpectedStatus)
	attempt.Metadata["playwrightStatus"] = params.Result.Status
	attempt.Metadata["playwrightExpectedStatus"] = params.Test.ExpectedStatus
	attempt.Metadata["annotations"] = append(params.Test.Annotations, params.Result.Annotations...)
	if len(params.Result.Errors) > 0 {
		attempt.ErrorMessage = params.Result.Errors[0].Message
		attempt.StackTrace = params.Result.Errors[0].Stack
		attempt.ErrorList = errorStrings(params.Result.Errors)
		for _, item := range params.Result.Errors {
			attempt.Errors = append(attempt.Errors, &m.TestErrorDocument{ErrorMessage: item.Message, StackTrace: item.Stack, Timestamp: attempt.EndTime})
		}
	}
	steps := buildStepTree(state)
	attempt.Steps, _ = m.StepFromDocuments(steps)
	attempt.StepsCount = int32(len(state.steps))
}

func buildStepTree(state *attemptState) []*m.StepDocument {
	roots := make([]*m.StepDocument, 0)
	for _, upstreamID := range state.stepOrder {
		step := state.steps[upstreamID]
		if step.Status == "RUNNING" {
			step.Status = "INTERRUPTED"
		}
		if step.ParentStepID == "" {
			roots = append(roots, step)
			continue
		}
		var parent *m.StepDocument
		for _, candidate := range state.steps {
			if candidate.ID == step.ParentStepID {
				parent = candidate
				break
			}
		}
		if parent == nil {
			roots = append(roots, step)
		} else {
			parent.Steps = append(parent.Steps, step)
		}
	}
	return roots
}

func (r *blobRun) addAttachment(state *attemptState, input jsonAttachment, stepID string) *importer.Attachment {
	attachment := &importer.Attachment{AttemptID: state.attempt.ID, StepID: stepID, Name: input.Name, MimeType: input.ContentType, SourcePath: input.Path, Available: true}
	if input.Base64 != "" {
		opener, size, err := decodeBase64Opener(input.Base64)
		if err != nil {
			r.warnings = append(r.warnings, importer.Warning{Code: "invalid_attachment", Message: "attachment " + input.Name + " has invalid base64", File: r.opened.source.Name})
			attachment.Available = false
		} else {
			attachment.Open, attachment.Size = opener, size
		}
	} else {
		resourcePath := path.Clean(strings.TrimPrefix(strings.ReplaceAll(input.Path, "\\", "/"), "/"))
		entry := r.opened.entries[resourcePath]
		if !strings.HasPrefix(resourcePath, "resources/") || entry == nil || entry.FileInfo().IsDir() {
			r.warnings = append(r.warnings, importer.Warning{Code: "missing_attachment_resource", Message: "attachment resource is unavailable: " + input.Path, File: r.opened.source.Name})
			attachment.Available = false
		} else {
			attachment.Open = zipOpener(entry)
			attachment.Size = int64(entry.UncompressedSize64)
		}
	}
	r.attachments = append(r.attachments, attachment)
	return attachment
}

func finalizeBlob(result *blobRun, states map[string]*attemptState) {
	byTest := make(map[string][]*m.TestAttempt)
	for _, attempt := range result.attempts {
		if attempt.Status == "RUNNING" {
			attempt.Status = "INTERRUPTED"
			state := states[attempt.Metadata["playwrightResultId"].(string)]
			attempt.Steps, _ = m.StepFromDocuments(buildStepTree(state))
			attempt.StepsCount = int32(len(state.steps))
		}
		byTest[attempt.TestID] = append(byTest[attempt.TestID], attempt)
	}
	for _, test := range result.tests {
		attempts := byTest[test.ID]
		if len(attempts) == 0 {
			continue
		}
		sort.SliceStable(attempts, func(i, j int) bool { return attempts[i].AttemptIndex < attempts[j].AttemptIndex })
		test.Status = aggregateStatuses(attempts)
		latestRetry := attempts[len(attempts)-1].AttemptIndex
		test.RetryIndex = &latestRetry
		test.StartTime, test.EndTime, test.Duration = attemptTimes(attempts)
	}
	status := normalizeRunStatus(result.result.Status)
	start := unixMillis(result.result.StartTime)
	duration := durationNanos(result.result.Duration)
	end := start.Add(time.Duration(duration))
	result.execution.Status, result.execution.StartTime, result.execution.EndTime, result.execution.Duration = status, &start, &end, &duration
	aggregateSuites(result.suites, result.tests)
}

func validateBlobSet(runs []*blobRun) error {
	if len(runs) == 0 {
		return importer.NewError("missing_files", "at least one report is required", nil)
	}
	sharded := runs[0].metadata.Shard != nil
	seen := make(map[int]struct{})
	total := 0
	for _, run := range runs {
		if (run.metadata.Shard != nil) != sharded {
			return importer.NewError("incompatible_report_set", "sharded and unsharded reports cannot be mixed", nil)
		}
		if run.config.RootDir != runs[0].config.RootDir {
			return importer.NewError("incompatible_report_set", "reports use different Playwright root directories", nil)
		}
		if sharded {
			if total == 0 {
				total = run.metadata.Shard.Total
			}
			if run.metadata.Shard.Total != total || run.metadata.Shard.Current < 1 || run.metadata.Shard.Current > total {
				return importer.NewError("invalid_shard_set", "reports contain inconsistent shard metadata", nil)
			}
			if _, exists := seen[run.metadata.Shard.Current]; exists {
				return importer.NewError("duplicate_shard", "the same shard index was provided more than once", nil)
			}
			seen[run.metadata.Shard.Current] = struct{}{}
		}
	}
	if sharded && len(seen) != total {
		missing := make([]int, 0)
		for i := 1; i <= total; i++ {
			if _, ok := seen[i]; !ok {
				missing = append(missing, i)
			}
		}
		return &importer.Error{Code: "incomplete_shard_set", Message: "all shards must be uploaded in one request", Details: map[string]interface{}{"missingShards": missing, "expected": total}}
	}
	return nil
}

func mergeRuns(runID, sourceDigest string, runs []*blobRun) *importer.RunImportBundle {
	bundle := &importer.RunImportBundle{ReportType: reportType, SourceDigest: sourceDigest}
	run := &m.TestRun{ID: runID, Name: "Imported Playwright run", Status: "NOT_RUN", InitiatedBy: "import", Metadata: map[string]interface{}{
		"source": map[string]interface{}{"producer": "playwright", "format": "blob", "version": "v2", "digest": sourceDigest},
	}}
	bundle.Run = run
	suites := make(map[string]*m.Suite)
	var starts, ends []time.Time
	sourceFiles := make([]map[string]interface{}, 0, len(runs))
	projectNames := make(map[string]struct{})
	for _, parsed := range runs {
		sourceFiles = append(sourceFiles, map[string]interface{}{"name": parsed.opened.source.Name, "sha256": parsed.opened.source.SHA256, "size": parsed.opened.source.Size, "userAgent": parsed.metadata.UserAgent})
		bundle.Executions = append(bundle.Executions, parsed.execution)
		bundle.Tests = append(bundle.Tests, parsed.tests...)
		for _, test := range parsed.tests {
			if project, ok := test.Metadata["project"].(string); ok && project != "" {
				projectNames[project] = struct{}{}
			}
		}
		bundle.Attempts = append(bundle.Attempts, parsed.attempts...)
		bundle.Attachments = append(bundle.Attachments, parsed.attachments...)
		bundle.Warnings = append(bundle.Warnings, parsed.warnings...)
		if parsed.execution.StartTime != nil {
			starts = append(starts, *parsed.execution.StartTime)
		}
		if parsed.execution.EndTime != nil {
			ends = append(ends, *parsed.execution.EndTime)
		}
		for _, suite := range parsed.suites {
			if existing := suites[suite.ID]; existing != nil {
				existing.TestCaseIDs = mergeStrings(existing.TestCaseIDs, suite.TestCaseIDs)
				existing.SubSuiteIDs = mergeStrings(existing.SubSuiteIDs, suite.SubSuiteIDs)
			} else {
				suites[suite.ID] = suite
				bundle.Suites = append(bundle.Suites, suite)
			}
		}
	}
	run.Metadata["source"].(map[string]interface{})["files"] = sourceFiles
	if len(projectNames) == 1 {
		for project := range projectNames {
			run.ProjectName = project
		}
	} else if len(projectNames) > 1 {
		run.ProjectName = "playwright"
	}
	aggregateSuites(bundle.Suites, bundle.Tests)
	if len(starts) > 0 {
		sort.Slice(starts, func(i, j int) bool { return starts[i].Before(starts[j]) })
		run.StartTime = &starts[0]
	}
	if len(ends) > 0 {
		sort.Slice(ends, func(i, j int) bool { return ends[i].Before(ends[j]) })
		run.EndTime = &ends[len(ends)-1]
	}
	if run.StartTime != nil && run.EndTime != nil {
		d := run.EndTime.Sub(*run.StartTime).Nanoseconds()
		run.Duration = &d
	}
	run.Status = aggregateRunStatuses(bundle.Executions)
	if len(runs) == 1 && runs[0].metadata.Name != "" {
		run.Name = runs[0].metadata.Name
	}
	return bundle
}

func normalizeTestStatus(actual, expected string) string {
	if actual == "skipped" {
		if expected == "skipped" {
			return "SKIPPED"
		}
		return "NOT_RUN"
	}
	if actual == "timedOut" {
		return "TIMEDOUT"
	}
	if actual == "interrupted" {
		return "INTERRUPTED"
	}
	if expected == "failed" {
		if actual == "failed" {
			return "PASSED"
		}
		if actual == "passed" {
			return "FAILED"
		}
	}
	if actual == "passed" {
		return "PASSED"
	}
	if actual == "failed" {
		return "FAILED"
	}
	return "UNKNOWN"
}

func normalizeRunStatus(status string) string {
	switch strings.ToLower(status) {
	case "passed":
		return "PASSED"
	case "failed":
		return "FAILED"
	case "timedout":
		return "TIMEDOUT"
	case "interrupted":
		return "INTERRUPTED"
	default:
		return "UNKNOWN"
	}
}

func aggregateStatuses(attempts []*m.TestAttempt) string {
	latest := attempts[len(attempts)-1].Status
	if latest == "PASSED" {
		for _, attempt := range attempts[:len(attempts)-1] {
			if attempt.Status == "FAILED" || attempt.Status == "TIMEDOUT" || attempt.Status == "INTERRUPTED" || attempt.Status == "BROKEN" {
				return "FLAKY"
			}
		}
	}
	return latest
}

func aggregateRunStatuses(executions []*m.RunExecution) string {
	status := "PASSED"
	for _, execution := range executions {
		if execution.Status == "FAILED" {
			return "FAILED"
		}
		if execution.Status != "PASSED" {
			status = execution.Status
		}
	}
	return status
}

func aggregateSuites(suites []*m.Suite, tests []*m.Test) {
	testsBySuite := make(map[string][]*m.Test)
	for _, test := range tests {
		if test.SuiteID != nil {
			testsBySuite[*test.SuiteID] = append(testsBySuite[*test.SuiteID], test)
		}
	}
	children := make(map[string][]*m.Suite)
	for _, suite := range suites {
		if suite.ParentSuiteID != nil {
			children[*suite.ParentSuiteID] = append(children[*suite.ParentSuiteID], suite)
		}
	}
	var visit func(*m.Suite) string
	visit = func(suite *m.Suite) string {
		statuses := make([]string, 0)
		var starts, ends []time.Time
		for _, test := range testsBySuite[suite.ID] {
			statuses = append(statuses, test.Status)
			if test.StartTime != nil {
				starts = append(starts, *test.StartTime)
			}
			if test.EndTime != nil {
				ends = append(ends, *test.EndTime)
			}
		}
		for _, child := range children[suite.ID] {
			statuses = append(statuses, visit(child))
			if child.StartTime != nil {
				starts = append(starts, *child.StartTime)
			}
			if child.EndTime != nil {
				ends = append(ends, *child.EndTime)
			}
		}
		suite.Status = aggregateSimpleStatuses(statuses)
		if len(starts) > 0 {
			sort.Slice(starts, func(i, j int) bool { return starts[i].Before(starts[j]) })
			suite.StartTime = &starts[0]
		}
		if len(ends) > 0 {
			sort.Slice(ends, func(i, j int) bool { return ends[i].Before(ends[j]) })
			suite.EndTime = &ends[len(ends)-1]
		}
		if suite.StartTime != nil && suite.EndTime != nil {
			d := suite.EndTime.Sub(*suite.StartTime).Nanoseconds()
			suite.Duration = &d
		}
		return suite.Status
	}
	for _, suite := range suites {
		if suite.ParentSuiteID == nil {
			visit(suite)
		}
	}
}

func aggregateSimpleStatuses(statuses []string) string {
	if len(statuses) == 0 {
		return "NOT_RUN"
	}
	allSkipped, allNotRun := true, true
	for _, status := range statuses {
		if status == "FAILED" || status == "BROKEN" || status == "TIMEDOUT" || status == "INTERRUPTED" {
			return "FAILED"
		}
		if status != "SKIPPED" {
			allSkipped = false
		}
		if status != "NOT_RUN" {
			allNotRun = false
		}
	}
	if allSkipped {
		return "SKIPPED"
	}
	if allNotRun {
		return "NOT_RUN"
	}
	return "PASSED"
}

func attemptTimes(attempts []*m.TestAttempt) (*time.Time, *time.Time, *int64) {
	var start, end *time.Time
	var duration int64
	for _, attempt := range attempts {
		if attempt.StartTime != nil && (start == nil || attempt.StartTime.Before(*start)) {
			value := *attempt.StartTime
			start = &value
		}
		if attempt.EndTime != nil && (end == nil || attempt.EndTime.After(*end)) {
			value := *attempt.EndTime
			end = &value
		}
		if attempt.Duration != nil {
			duration += *attempt.Duration
		}
	}
	return start, end, &duration
}

func unixMillis(value int64) time.Time { return time.UnixMilli(value).UTC() }

func errorStrings(input []jsonError) []string {
	result := make([]string, 0, len(input))
	for _, item := range input {
		if item.Message != "" {
			result = append(result, item.Message)
		}
	}
	return result
}

func appendUnique(values []string, value string) []string {
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}
func mergeStrings(left, right []string) []string {
	for _, value := range right {
		left = appendUnique(left, value)
	}
	return left
}
