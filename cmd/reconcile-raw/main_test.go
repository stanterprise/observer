package main

import "testing"

func TestKindFromPath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"/x/ReportRunStart", "run.start"},
		{"/x/ReportRunEnd", "run.end"},
		{"/x/ReportTestBegin", "test.begin"},
		{"/x/ReportTestEnd", "test.end"},
		{"/x/ReportStepBegin", "step.begin"},
		{"/x/ReportStepEnd", "step.end"},
		{"/x/ReportSuiteBegin", "suite.begin"},
		{"/x/ReportSuiteEnd", "suite.end"},
	}

	for _, tc := range cases {
		if got := kindFromPath(tc.in); got != tc.want {
			t.Fatalf("kindFromPath(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeKV(t *testing.T) {
	input := []any{
		map[string]any{"Key": "type", "Value": "test.begin"},
		map[string]any{"Key": "data", "Value": []any{
			map[string]any{"Key": "test_case", "Value": []any{
				map[string]any{"Key": "id", "Value": "abc"},
				map[string]any{"Key": "run_id", "Value": "run-1"},
			}},
		}},
	}

	m := asMap(normalizeKV(input))
	if m == nil {
		t.Fatal("expected map")
	}
	if m["type"] != "test.begin" {
		t.Fatalf("bad type: %v", m["type"])
	}
	tc := asMap(asMap(m["data"])["test_case"])
	if asString(tc["id"]) != "abc" {
		t.Fatalf("bad id: %v", tc["id"])
	}
}

func TestNormalizeProtoTime(t *testing.T) {
	got := normalizeProtoTime(map[string]any{"seconds": float64(1775444514), "nanos": float64(705000000)})
	if got == "" {
		t.Fatal("expected non-empty normalized time")
	}
}
