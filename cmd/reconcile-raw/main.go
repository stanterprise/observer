package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Event struct {
	Source       string
	Line         int
	Kind         string
	RunID        string
	EntityType   string
	EntityID     string
	RetryIndex   string
	Status       string
	ShardCurrent string
	StartTime    string
	Key          string
}

type KeyDiff struct {
	Key      string
	Sent     int
	Received int
	Kind     string
}

type KindStats struct {
	Sent               int
	Received           int
	Matched            int
	MissingInReceived  int
	UnexpectedReceived int
}

type Report struct {
	SentTotal               int
	ReceivedTotal           int
	MatchedTotal            int
	MissingInReceivedTotal  int
	UnexpectedReceivedTotal int
	KindStats               map[string]KindStats
	Passes                  []PassStats
	MissingEvents           []Event
	UnexpectedEvents        []Event
	ReconciledPairs         []ReconciledPair
	MissingInReceived       []KeyDiff
	UnexpectedInReceived    []KeyDiff
}

type PassStats struct {
	Name    string
	Matched int
}

type ReconciledPair struct {
	Pass     string
	MatchKey string
	Sent     Event
	Received Event
}

type ReconciledRecord struct {
	Pass     string `json:"pass"`
	MatchKey string `json:"match_key"`
	Sent     Event  `json:"sent"`
	Received Event  `json:"received"`
}

type TriageSummaryRecord struct {
	Side        string   `json:"side"`
	Kind        string   `json:"kind"`
	EntityType  string   `json:"entity_type"`
	RetryIndex  string   `json:"retry_index"`
	Status      string   `json:"status"`
	Count       int      `json:"count"`
	SampleIDs   []string `json:"sample_ids"`
	SampleLines []int    `json:"sample_lines"`
}

func main() {
	var sentPath string
	var receivedPath string
	var top int
	var saveRecords bool
	var outPrefix string

	flag.StringVar(&sentPath, "sent", "stanterprise-debug.jsonl", "Path to sent raw JSONL file")
	flag.StringVar(&receivedPath, "received", "raw-messages-03CA3D78-41CD-435B-8C67-EAE0EF364DA8-2026-04-06T03-03-15-847Z.jsonl", "Path to received raw JSONL file")
	flag.IntVar(&top, "top", 20, "How many unmatched keys to print per side")
	flag.BoolVar(&saveRecords, "save-records", true, "Write reconciled/missing/unexpected records to JSONL files")
	flag.StringVar(&outPrefix, "out-prefix", "reconcile-raw", "Prefix for output JSONL files")
	flag.Parse()

	sentEvents, err := parseSentFile(sentPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse sent file: %v\n", err)
		os.Exit(1)
	}

	receivedEvents, err := parseReceivedFile(receivedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse received file: %v\n", err)
		os.Exit(1)
	}

	report := reconcile(sentEvents, receivedEvents)
	printReport(report, top)

	if saveRecords {
		if err := writeRetainedRecords(report, outPrefix); err != nil {
			fmt.Fprintf(os.Stderr, "write retained records: %v\n", err)
			os.Exit(1)
		}
	}
}

func parseSentFile(path string) ([]Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 32*1024*1024)

	events := make([]Event, 0, 1024)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := scanner.Bytes()

		var raw map[string]any
		if err := json.Unmarshal(line, &raw); err != nil {
			return nil, fmt.Errorf("line %d: invalid json: %w", lineNo, err)
		}

		event := normalizeSent(raw, lineNo)
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func parseReceivedFile(path string) ([]Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 32*1024*1024)

	events := make([]Event, 0, 1024)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := scanner.Bytes()

		var raw map[string]any
		if err := json.Unmarshal(line, &raw); err != nil {
			return nil, fmt.Errorf("line %d: invalid json: %w", lineNo, err)
		}

		event := normalizeReceived(raw, lineNo)
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func normalizeSent(raw map[string]any, line int) Event {
	kind := kindFromPath(asString(raw["path"]))
	msg := asMap(raw["message"])

	event := Event{Source: "sent", Line: line, Kind: kind}

	switch kind {
	case "test.begin", "test.end":
		tc := asMap(msg["test_case"])
		event.EntityType = "test"
		event.EntityID = asString(tc["id"])
		event.RunID = asString(tc["run_id"])
		event.RetryIndex = normalizeRetryIndex(tc["retry_index"])
		event.Status = normalizeStatus(tc["status"])
		event.StartTime = normalizeProtoTime(tc["start_time"])
	case "step.begin", "step.end":
		step := asMap(msg["step"])
		event.EntityType = "step"
		event.EntityID = asString(step["id"])
		event.RunID = asString(step["run_id"])
		event.RetryIndex = normalizeRetryIndex(step["retry_index"])
		event.Status = normalizeStatus(step["status"])
		event.StartTime = normalizeProtoTime(step["start_time"])
	case "suite.begin", "suite.end":
		suite := asMap(msg["test_suite"])
		if len(suite) == 0 {
			suite = asMap(msg["suite"])
		}
		event.EntityType = "suite"
		event.EntityID = asString(suite["id"])
		event.RunID = asString(suite["run_id"])
		event.Status = normalizeStatus(suite["status"])
	case "run.start", "run.end":
		event.EntityType = "run"
		event.RunID = asString(msg["run_id"])
		event.EntityID = event.RunID
		event.ShardCurrent = extractShardCurrent(msg)
	default:
		event.EntityType = "unknown"
		event.RunID = firstNonEmpty(asString(msg["run_id"]), findStringByKey(msg, "run_id"))
		event.EntityID = firstNonEmpty(findStringByKey(msg, "id"), fmt.Sprintf("line-%d", line))
	}

	if event.EntityID == "" {
		event.EntityID = firstNonEmpty(event.RunID, fmt.Sprintf("line-%d", line))
	}
	if event.RunID == "" {
		event.RunID = findStringByKey(msg, "run_id")
	}

	event.Key = buildEventKey(event)
	return event
}

func normalizeReceived(raw map[string]any, line int) Event {
	payload := normalizeKV(raw["payload"])
	payloadMap := asMap(payload)
	data := asMap(payloadMap["data"])

	kind := asString(raw["eventType"])
	if kind == "" {
		kind = asString(payloadMap["type"])
	}

	event := Event{Source: "received", Line: line, Kind: kind}

	switch kind {
	case "test.begin", "test.end":
		tc := asMap(data["test_case"])
		event.EntityType = "test"
		event.EntityID = asString(tc["id"])
		event.RunID = asString(tc["run_id"])
		event.RetryIndex = normalizeRetryIndex(tc["retry_index"])
		event.Status = normalizeStatus(tc["status"])
		event.StartTime = asString(tc["start_time"])
	case "step.begin", "step.end":
		step := asMap(data["step"])
		event.EntityType = "step"
		event.EntityID = asString(step["id"])
		event.RunID = asString(step["run_id"])
		event.RetryIndex = normalizeRetryIndex(step["retry_index"])
		event.Status = normalizeStatus(step["status"])
		event.StartTime = asString(step["start_time"])
	case "suite.begin", "suite.end":
		suite := asMap(data["test_suite"])
		if len(suite) == 0 {
			suite = asMap(data["suite"])
		}
		event.EntityType = "suite"
		event.EntityID = asString(suite["id"])
		event.RunID = asString(suite["run_id"])
		event.Status = normalizeStatus(suite["status"])
	case "run.start", "run.end":
		event.EntityType = "run"
		event.RunID = asString(data["run_id"])
		event.EntityID = event.RunID
		event.ShardCurrent = extractShardCurrent(data)
	default:
		event.EntityType = "unknown"
		event.RunID = firstNonEmpty(asString(data["run_id"]), findStringByKey(data, "run_id"))
		event.EntityID = firstNonEmpty(findStringByKey(data, "id"), fmt.Sprintf("line-%d", line))
	}

	if event.EntityID == "" {
		event.EntityID = firstNonEmpty(event.RunID, fmt.Sprintf("line-%d", line))
	}
	if event.RunID == "" {
		event.RunID = findStringByKey(data, "run_id")
	}

	event.Key = buildEventKey(event)
	return event
}

func reconcile(sent, received []Event) Report {
	kindStats := map[string]KindStats{}

	for _, e := range sent {
		ks := kindStats[e.Kind]
		ks.Sent++
		kindStats[e.Kind] = ks
	}
	for _, e := range received {
		ks := kindStats[e.Kind]
		ks.Received++
		kindStats[e.Kind] = ks
	}

	passes := []struct {
		name  string
		keyFn func(Event) string
	}{
		{name: "strict", keyFn: keyStrict},
		{name: "ignore_start_time", keyFn: keyNoStartTime},
		{name: "ignore_start_time_and_shard", keyFn: keyNoStartTimeNoShard},
		{name: "entity_level", keyFn: keyEntityOnly},
		{name: "temporal_fallback", keyFn: keyTemporalFallback},
		{name: "entity_cross_kind", keyFn: keyEntityCrossKind},
		{name: "run_cross_kind", keyFn: keyRunCrossKind},
		{name: "kind_run_bucket", keyFn: keyKindRunBucket},
		{name: "phase_insensitive_bucket", keyFn: keyPhaseInsensitiveBucket},
	}

	remainingSent := append([]Event(nil), sent...)
	remainingReceived := append([]Event(nil), received...)
	matchedTotal := 0
	passStats := make([]PassStats, 0, len(passes))
	reconciledPairs := make([]ReconciledPair, 0)

	for _, pass := range passes {
		var matchedInPass int
		var pairs []ReconciledPair
		remainingSent, remainingReceived, matchedInPass, pairs = matchPass(remainingSent, remainingReceived, pass.name, pass.keyFn)

		matchedTotal += matchedInPass
		passStats = append(passStats, PassStats{Name: pass.name, Matched: matchedInPass})
		reconciledPairs = append(reconciledPairs, pairs...)

		for _, pair := range pairs {
			ks := kindStats[pair.Sent.Kind]
			ks.Matched++
			kindStats[pair.Sent.Kind] = ks
		}
	}

	for _, e := range remainingSent {
		ks := kindStats[e.Kind]
		ks.MissingInReceived++
		kindStats[e.Kind] = ks
	}
	for _, e := range remainingReceived {
		ks := kindStats[e.Kind]
		ks.UnexpectedReceived++
		kindStats[e.Kind] = ks
	}

	missingInReceived, unexpectedInReceived := buildKeyDiffs(remainingSent, remainingReceived)

	return Report{
		SentTotal:               len(sent),
		ReceivedTotal:           len(received),
		MatchedTotal:            matchedTotal,
		MissingInReceivedTotal:  len(remainingSent),
		UnexpectedReceivedTotal: len(remainingReceived),
		KindStats:               kindStats,
		Passes:                  passStats,
		MissingEvents:           remainingSent,
		UnexpectedEvents:        remainingReceived,
		ReconciledPairs:         reconciledPairs,
		MissingInReceived:       missingInReceived,
		UnexpectedInReceived:    unexpectedInReceived,
	}
}

func matchPass(sent, received []Event, passName string, keyFn func(Event) string) ([]Event, []Event, int, []ReconciledPair) {
	sentByKey := map[string][]Event{}
	receivedByKey := map[string][]Event{}
	usedKeys := map[string]struct{}{}

	for _, e := range sent {
		k := keyFn(e)
		if k == "" {
			continue
		}
		sentByKey[k] = append(sentByKey[k], e)
		usedKeys[k] = struct{}{}
	}
	for _, e := range received {
		k := keyFn(e)
		if k == "" {
			continue
		}
		receivedByKey[k] = append(receivedByKey[k], e)
		usedKeys[k] = struct{}{}
	}

	matched := 0
	pairs := make([]ReconciledPair, 0)

	for key := range usedKeys {
		sList := sentByKey[key]
		rList := receivedByKey[key]
		n := min(len(sList), len(rList))
		matched += n
		for i := 0; i < n; i++ {
			pairs = append(pairs, ReconciledPair{
				Pass:     passName,
				MatchKey: key,
				Sent:     sList[i],
				Received: rList[i],
			})
		}
	}

	remainingSent := make([]Event, 0)
	for key := range usedKeys {
		sList := sentByKey[key]
		rList := receivedByKey[key]
		n := min(len(sList), len(rList))
		if len(sList) > n {
			remainingSent = append(remainingSent, sList[n:]...)
		}
	}
	remainingReceived := make([]Event, 0)
	for key := range usedKeys {
		sList := sentByKey[key]
		rList := receivedByKey[key]
		n := min(len(sList), len(rList))
		if len(rList) > n {
			remainingReceived = append(remainingReceived, rList[n:]...)
		}
	}

	// Events with empty match keys are retained for later/manual inspection.
	for _, e := range sent {
		if keyFn(e) == "" {
			remainingSent = append(remainingSent, e)
		}
	}
	for _, e := range received {
		if keyFn(e) == "" {
			remainingReceived = append(remainingReceived, e)
		}
	}

	return remainingSent, remainingReceived, matched, pairs
}

func buildKeyDiffs(missing []Event, unexpected []Event) ([]KeyDiff, []KeyDiff) {
	missingCounts := map[string]int{}
	unexpectedCounts := map[string]int{}
	kindForKey := map[string]string{}

	for _, e := range missing {
		k := keyStrict(e)
		missingCounts[k]++
		if kindForKey[k] == "" {
			kindForKey[k] = e.Kind
		}
	}
	for _, e := range unexpected {
		k := keyStrict(e)
		unexpectedCounts[k]++
		if kindForKey[k] == "" {
			kindForKey[k] = e.Kind
		}
	}

	missingDiffs := make([]KeyDiff, 0, len(missingCounts))
	for key, sentCount := range missingCounts {
		missingDiffs = append(missingDiffs, KeyDiff{Key: key, Sent: sentCount, Received: unexpectedCounts[key], Kind: kindForKey[key]})
	}

	unexpectedDiffs := make([]KeyDiff, 0, len(unexpectedCounts))
	for key, recvCount := range unexpectedCounts {
		unexpectedDiffs = append(unexpectedDiffs, KeyDiff{Key: key, Sent: missingCounts[key], Received: recvCount, Kind: kindForKey[key]})
	}

	sort.Slice(missingDiffs, func(i, j int) bool {
		di := missingDiffs[i].Sent - missingDiffs[i].Received
		dj := missingDiffs[j].Sent - missingDiffs[j].Received
		if di == dj {
			return missingDiffs[i].Key < missingDiffs[j].Key
		}
		return di > dj
	})
	sort.Slice(unexpectedDiffs, func(i, j int) bool {
		di := unexpectedDiffs[i].Received - unexpectedDiffs[i].Sent
		dj := unexpectedDiffs[j].Received - unexpectedDiffs[j].Sent
		if di == dj {
			return unexpectedDiffs[i].Key < unexpectedDiffs[j].Key
		}
		return di > dj
	})

	return missingDiffs, unexpectedDiffs
}

func keyStrict(e Event) string {
	return strings.Join([]string{e.Kind, e.RunID, e.EntityType, e.EntityID, e.RetryIndex, e.ShardCurrent, e.StartTime}, "|")
}

func keyNoStartTime(e Event) string {
	return strings.Join([]string{e.Kind, e.RunID, e.EntityType, e.EntityID, e.RetryIndex, e.ShardCurrent}, "|")
}

func keyNoStartTimeNoShard(e Event) string {
	return strings.Join([]string{e.Kind, e.RunID, e.EntityType, e.EntityID, e.RetryIndex}, "|")
}

func keyEntityOnly(e Event) string {
	return strings.Join([]string{e.Kind, e.RunID, e.EntityType, e.EntityID}, "|")
}

func keyTemporalFallback(e Event) string {
	if e.EntityType != "step" && e.EntityType != "test" {
		return ""
	}
	t := normalizeTimeToMillis(e.StartTime)
	if t == "" {
		return ""
	}
	return strings.Join([]string{e.Kind, e.RunID, e.EntityType, e.RetryIndex, t}, "|")
}

func keyRunCrossKind(e Event) string {
	if e.EntityType != "run" {
		return ""
	}
	return strings.Join([]string{"run", e.RunID}, "|")
}

func keyEntityCrossKind(e Event) string {
	if e.EntityType != "step" && e.EntityType != "test" && e.EntityType != "suite" {
		return ""
	}
	if e.RunID == "" || e.EntityID == "" {
		return ""
	}
	return strings.Join([]string{"cross", e.RunID, e.EntityType, e.EntityID, e.RetryIndex}, "|")
}

func keyKindRunBucket(e Event) string {
	if e.Kind == "" || e.RunID == "" || e.EntityType == "" {
		return ""
	}
	return strings.Join([]string{e.Kind, e.RunID, e.EntityType}, "|")
}

func keyPhaseInsensitiveBucket(e Event) string {
	if e.RunID == "" || e.EntityType == "" {
		return ""
	}
	phaseKind := e.Kind
	if strings.HasSuffix(phaseKind, ".begin") || strings.HasSuffix(phaseKind, ".end") {
		idx := strings.LastIndex(phaseKind, ".")
		if idx > 0 {
			phaseKind = phaseKind[:idx] + ".phase"
		}
	}
	return strings.Join([]string{phaseKind, e.RunID, e.EntityType}, "|")
}

func normalizeTimeToMillis(v string) string {
	if v == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return v
	}
	return t.UTC().Truncate(time.Millisecond).Format("2006-01-02T15:04:05.000Z")
}

func printReport(r Report, top int) {
	fmt.Println("Raw Message Reconciliation Report")
	fmt.Println("=================================")
	fmt.Printf("Sent events:             %d\n", r.SentTotal)
	fmt.Printf("Received events:         %d\n", r.ReceivedTotal)
	fmt.Printf("Matched events:          %d\n", r.MatchedTotal)
	fmt.Printf("Missing in received:     %d\n", r.MissingInReceivedTotal)
	fmt.Printf("Unexpected in received:  %d\n", r.UnexpectedReceivedTotal)
	if r.MissingInReceivedTotal == 0 && r.UnexpectedReceivedTotal == 0 {
		fmt.Println("Fully reconciled:        yes")
	} else {
		fmt.Println("Fully reconciled:        no")
	}
	fmt.Println()

	fmt.Println("Reconciliation passes")
	fmt.Println("--------------------")
	for _, pass := range r.Passes {
		fmt.Printf("%-30s %6d\n", pass.Name, pass.Matched)
	}
	fmt.Println()

	kinds := make([]string, 0, len(r.KindStats))
	for kind := range r.KindStats {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)

	fmt.Println("Per-kind summary")
	fmt.Println("----------------")
	fmt.Printf("%-12s %8s %10s %8s %10s %11s\n", "kind", "sent", "received", "matched", "missing", "unexpected")
	for _, kind := range kinds {
		ks := r.KindStats[kind]
		fmt.Printf("%-12s %8d %10d %8d %10d %11d\n", kind, ks.Sent, ks.Received, ks.Matched, ks.MissingInReceived, ks.UnexpectedReceived)
	}
	fmt.Println()

	printTopDiffs("Top missing in received", r.MissingInReceived, top, true)
	fmt.Println()
	printTopDiffs("Top unexpected in received", r.UnexpectedInReceived, top, false)
}

func writeRetainedRecords(r Report, outPrefix string) error {
	if err := writeJSONLines(outPrefix+"-missing.jsonl", r.MissingEvents); err != nil {
		return err
	}
	if err := writeJSONLines(outPrefix+"-unexpected.jsonl", r.UnexpectedEvents); err != nil {
		return err
	}

	pairs := make([]ReconciledRecord, 0, len(r.ReconciledPairs))
	for _, p := range r.ReconciledPairs {
		pairs = append(pairs, ReconciledRecord{
			Pass:     p.Pass,
			MatchKey: p.MatchKey,
			Sent:     p.Sent,
			Received: p.Received,
		})
	}
	if err := writeJSONLines(outPrefix+"-reconciled.jsonl", pairs); err != nil {
		return err
	}

	triage := buildTriageSummary(r.MissingEvents, r.UnexpectedEvents)
	if err := writeJSONLines(outPrefix+"-triage-summary.jsonl", triage); err != nil {
		return err
	}

	fmt.Printf("\nRetained records written:\n")
	fmt.Printf("  %s\n", outPrefix+"-missing.jsonl")
	fmt.Printf("  %s\n", outPrefix+"-unexpected.jsonl")
	fmt.Printf("  %s\n", outPrefix+"-reconciled.jsonl")
	fmt.Printf("  %s\n", outPrefix+"-triage-summary.jsonl")

	return nil
}

func writeJSONLines(path string, records any) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)

	switch typed := records.(type) {
	case []Event:
		for _, rec := range typed {
			if err := enc.Encode(rec); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}
		}
	case []ReconciledRecord:
		for _, rec := range typed {
			if err := enc.Encode(rec); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}
		}
	case []TriageSummaryRecord:
		for _, rec := range typed {
			if err := enc.Encode(rec); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}
		}
	default:
		return fmt.Errorf("unsupported record type for %s", path)
	}

	return nil
}

func buildTriageSummary(missing []Event, unexpected []Event) []TriageSummaryRecord {
	type acc struct {
		rec TriageSummaryRecord
	}

	appendEvent := func(m map[string]*acc, side string, e Event) {
		key := strings.Join([]string{side, e.Kind, e.EntityType, e.RetryIndex, e.Status}, "|")
		entry, ok := m[key]
		if !ok {
			entry = &acc{rec: TriageSummaryRecord{
				Side:       side,
				Kind:       e.Kind,
				EntityType: e.EntityType,
				RetryIndex: e.RetryIndex,
				Status:     e.Status,
			}}
			m[key] = entry
		}

		entry.rec.Count++
		if len(entry.rec.SampleIDs) < 5 {
			entry.rec.SampleIDs = append(entry.rec.SampleIDs, e.EntityID)
		}
		if len(entry.rec.SampleLines) < 5 {
			entry.rec.SampleLines = append(entry.rec.SampleLines, e.Line)
		}
	}

	bucket := map[string]*acc{}
	for _, e := range missing {
		appendEvent(bucket, "missing", e)
	}
	for _, e := range unexpected {
		appendEvent(bucket, "unexpected", e)
	}

	out := make([]TriageSummaryRecord, 0, len(bucket))
	for _, b := range bucket {
		out = append(out, b.rec)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			if out[i].Side == out[j].Side {
				if out[i].Kind == out[j].Kind {
					return out[i].RetryIndex < out[j].RetryIndex
				}
				return out[i].Kind < out[j].Kind
			}
			return out[i].Side < out[j].Side
		}
		return out[i].Count > out[j].Count
	})

	return out
}

func printTopDiffs(title string, diffs []KeyDiff, top int, missing bool) {
	fmt.Println(title)
	fmt.Println(strings.Repeat("-", len(title)))
	if len(diffs) == 0 {
		fmt.Println("none")
		return
	}
	if top <= 0 || top > len(diffs) {
		top = len(diffs)
	}
	for i := 0; i < top; i++ {
		d := diffs[i]
		delta := d.Received - d.Sent
		if missing {
			delta = d.Sent - d.Received
		}
		fmt.Printf("%2d) kind=%s delta=%d sent=%d received=%d key=%s\n", i+1, d.Kind, delta, d.Sent, d.Received, d.Key)
	}
}

func kindFromPath(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx >= 0 {
		path = path[idx+1:]
	}
	switch path {
	case "ReportRunStart":
		return "run.start"
	case "ReportRunEnd":
		return "run.end"
	case "ReportSuiteBegin", "ReportTestSuiteBegin":
		return "suite.begin"
	case "ReportSuiteEnd", "ReportTestSuiteEnd":
		return "suite.end"
	case "ReportTestBegin":
		return "test.begin"
	case "ReportTestEnd":
		return "test.end"
	case "ReportStepBegin":
		return "step.begin"
	case "ReportStepEnd":
		return "step.end"
	default:
		return normalizePathFallback(path)
	}
}

func normalizePathFallback(path string) string {
	if path == "" {
		return "unknown"
	}
	lower := strings.TrimPrefix(strings.ToLower(path), "report")
	if strings.HasSuffix(lower, "begin") {
		return strings.TrimSuffix(lower, "begin") + ".begin"
	}
	if strings.HasSuffix(lower, "end") {
		return strings.TrimSuffix(lower, "end") + ".end"
	}
	return lower
}

func normalizeKV(v any) any {
	switch t := v.(type) {
	case []any:
		if looksLikeKVArray(t) {
			out := map[string]any{}
			for _, item := range t {
				kv := asMap(item)
				k := asString(kv["Key"])
				val := normalizeKV(kv["Value"])
				if prev, ok := out[k]; ok {
					if arr, ok := prev.([]any); ok {
						out[k] = append(arr, val)
					} else {
						out[k] = []any{prev, val}
					}
				} else {
					out[k] = val
				}
			}
			return out
		}
		arr := make([]any, 0, len(t))
		for _, item := range t {
			arr = append(arr, normalizeKV(item))
		}
		return arr
	case map[string]any:
		out := map[string]any{}
		for k, v := range t {
			out[k] = normalizeKV(v)
		}
		return out
	default:
		return v
	}
}

func looksLikeKVArray(items []any) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		m := asMap(item)
		if m == nil {
			return false
		}
		if _, ok := m["Key"]; !ok {
			return false
		}
		if _, ok := m["Value"]; !ok {
			return false
		}
	}
	return true
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func scalarToString(v any) string { return asString(v) }

func normalizeRetryIndex(v any) string {
	s := scalarToString(v)
	if s == "" {
		return "0"
	}
	return s
}

func normalizeStatus(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return statusFromCode(int(t))
	case int:
		return statusFromCode(t)
	case int64:
		return statusFromCode(int(t))
	default:
		return ""
	}
}

func statusFromCode(code int) string {
	switch code {
	case 0:
		return "STATUS_UNSPECIFIED"
	case 1:
		return "PASSED"
	case 2:
		return "FAILED"
	case 3:
		return "SKIPPED"
	case 4:
		return "TIMED_OUT"
	case 5:
		return "INTERRUPTED"
	case 6:
		return "FLAKY"
	case 7:
		return "RUNNING"
	case 8:
		return "NOT_RUN"
	default:
		return strconv.Itoa(code)
	}
}

func normalizeProtoTime(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		sec, ok := toInt64(t["seconds"])
		if !ok {
			return ""
		}
		nsec, _ := toInt64(t["nanos"])
		return time.Unix(sec, nsec).UTC().Format(time.RFC3339Nano)
	default:
		return ""
	}
}

func toInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case int:
		return int64(t), true
	case int64:
		return t, true
	case json.Number:
		n, err := t.Int64()
		return n, err == nil
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func extractShardCurrent(obj map[string]any) string {
	metadata := asMap(obj["metadata"])
	if metadata == nil {
		return ""
	}
	return scalarToString(metadata["shard.current"])
}

func findStringByKey(v any, needle string) string {
	switch t := v.(type) {
	case map[string]any:
		if val, ok := t[needle]; ok {
			if s := asString(val); s != "" {
				return s
			}
		}
		for _, child := range t {
			if s := findStringByKey(child, needle); s != "" {
				return s
			}
		}
	case []any:
		for _, child := range t {
			if s := findStringByKey(child, needle); s != "" {
				return s
			}
		}
	}
	return ""
}

func buildEventKey(e Event) string {
	return keyStrict(e)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
