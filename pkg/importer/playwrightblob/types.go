package playwrightblob

import "encoding/json"

type jsonEvent struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type blobMetadata struct {
	Version       int    `json:"version"`
	UserAgent     string `json:"userAgent"`
	Name          string `json:"name,omitempty"`
	PathSeparator string `json:"pathSeparator,omitempty"`
	Shard         *struct {
		Total   int `json:"total"`
		Current int `json:"current"`
	} `json:"shard,omitempty"`
}

type jsonLocation struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type jsonConfig struct {
	ConfigFile    string                 `json:"configFile"`
	GlobalTimeout int64                  `json:"globalTimeout"`
	MaxFailures   int                    `json:"maxFailures"`
	Metadata      map[string]interface{} `json:"metadata"`
	RootDir       string                 `json:"rootDir"`
	Version       string                 `json:"version"`
	Workers       int                    `json:"workers"`
	Tags          []string               `json:"tags"`
}

type jsonProject struct {
	Name         string                 `json:"name"`
	Metadata     map[string]interface{} `json:"metadata"`
	RepeatEach   int                    `json:"repeatEach"`
	Retries      int                    `json:"retries"`
	TestDir      string                 `json:"testDir"`
	Timeout      int                    `json:"timeout"`
	Dependencies []string               `json:"dependencies"`
	Suites       []jsonSuite            `json:"suites"`
}

type jsonSuite struct {
	Title    string            `json:"title"`
	Location *jsonLocation     `json:"location,omitempty"`
	Entries  []json.RawMessage `json:"entries"`
}

type jsonTestCase struct {
	TestID          string                   `json:"testId"`
	Title           string                   `json:"title"`
	Location        jsonLocation             `json:"location"`
	Retries         int                      `json:"retries"`
	Tags            []string                 `json:"tags"`
	RepeatEachIndex int                      `json:"repeatEachIndex"`
	Annotations     []map[string]interface{} `json:"annotations"`
}

type jsonResultStart struct {
	ID            string `json:"id"`
	Retry         int    `json:"retry"`
	WorkerIndex   int    `json:"workerIndex"`
	ParallelIndex int    `json:"parallelIndex"`
	StartTime     int64  `json:"startTime"`
}

type jsonError struct {
	Message  string        `json:"message"`
	Stack    string        `json:"stack"`
	Snippet  string        `json:"snippet"`
	Location *jsonLocation `json:"location,omitempty"`
}

type jsonAttachment struct {
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Path        string `json:"path,omitempty"`
	Base64      string `json:"base64,omitempty"`
}

type jsonResultEnd struct {
	ID          string                   `json:"id"`
	Duration    float64                  `json:"duration"`
	Status      string                   `json:"status"`
	Errors      []jsonError              `json:"errors"`
	Attachments []jsonAttachment         `json:"attachments,omitempty"`
	Annotations []map[string]interface{} `json:"annotations,omitempty"`
}

type jsonTestEnd struct {
	TestID         string                   `json:"testId"`
	ExpectedStatus string                   `json:"expectedStatus"`
	Timeout        int                      `json:"timeout"`
	Annotations    []map[string]interface{} `json:"annotations"`
}

type jsonStepStart struct {
	ID           string        `json:"id"`
	ParentStepID string        `json:"parentStepId,omitempty"`
	Title        string        `json:"title"`
	Category     string        `json:"category"`
	StartTime    int64         `json:"startTime"`
	Location     *jsonLocation `json:"location,omitempty"`
}

type jsonStepEnd struct {
	ID          string                   `json:"id"`
	Duration    float64                  `json:"duration"`
	Error       *jsonError               `json:"error,omitempty"`
	Attachments []int                    `json:"attachments,omitempty"`
	Annotations []map[string]interface{} `json:"annotations,omitempty"`
}

type jsonFullResult struct {
	Status    string  `json:"status"`
	StartTime int64   `json:"startTime"`
	Duration  float64 `json:"duration"`
}

type configureParams struct {
	Config jsonConfig `json:"config"`
}

type projectParams struct {
	Project jsonProject `json:"project"`
}

type testBeginParams struct {
	TestID string          `json:"testId"`
	Result jsonResultStart `json:"result"`
}

type testEndParams struct {
	Test   jsonTestEnd   `json:"test"`
	Result jsonResultEnd `json:"result"`
}

type stepBeginParams struct {
	TestID   string        `json:"testId"`
	ResultID string        `json:"resultId"`
	Step     jsonStepStart `json:"step"`
}

type stepEndParams struct {
	TestID   string      `json:"testId"`
	ResultID string      `json:"resultId"`
	Step     jsonStepEnd `json:"step"`
}

type stdIOParams struct {
	Type     string `json:"type"`
	TestID   string `json:"testId"`
	ResultID string `json:"resultId"`
	Data     string `json:"data"`
	IsBase64 bool   `json:"isBase64"`
}

type attachParams struct {
	TestID      string           `json:"testId"`
	ResultID    string           `json:"resultId"`
	Attachments []jsonAttachment `json:"attachments"`
}

type errorParams struct {
	Error jsonError `json:"error"`
}

type endParams struct {
	Result jsonFullResult `json:"result"`
}
