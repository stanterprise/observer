import type { TestStatus } from "./common";
import type { Test } from "./testCase";

export interface TestSuite {
  id: string;
  name: string;
  runId: string;
  description?: string;
  parentSuiteId?: string;
  startTime?: string;
  endTime?: string;
  duration?: number; // in milliseconds
  status?: TestStatus;
  metadata?: Record<string, string>;
  projectName?: string;
  location?: string;
  type?: string;
  testCaseIds?: string[];
  subSuiteIds?: string[];
  project?: string;
  initiatedBy?: string;
  author?: string;
  owner?: string;
  tests?: Test[];
  suites?: TestSuite[];
  createdAt?: string;
  updatedAt?: string;
}

export type SuiteType = "root" | "project" | "subsuite";

//     string id = 1; // Unique identifier for the test suite run
// string name = 2; // Name of the test suite
// string description = 3; // Description of the test suite
// string run_id = 4; // Identifier for the global test run this suite belongs to
// google.protobuf.Timestamp start_time = 5; // Start time of the suite execution
// google.protobuf.Timestamp end_time = 6; // End time of the suite execution
// google.protobuf.Duration duration = 7; // Duration of the suite execution
// testsystem.v1.common.TestStatus status = 8; // Overall status of the suite run
// map<string, string> metadata = 9; // Additional metadata for the suite
// string location = 10; // Location or path of the test suite definition
// SuiteType type = 11; // Type of the test suite (e.g., "root", "project", "subsuite")
// string parent_suite_id = 12; // Reference to the parent suite, if any
// repeated string test_case_ids = 13; // List of test case IDs in this suite
// repeated string sub_suite_ids = 14; // Nested test suite IDs
// string project = 15; // Project identifier (e.g., browser/device configuration for Playwright)
// string initiated_by = 16; // Identifier for who initiated the test run
// string author = 17; // Author of the test suite
// string owner = 18; // Team or individual responsible for the test suite
// repeated testsystem.v1.entities.TestCaseRun test_cases = 19; // Nested test case objects (optional, for sending full structure)
// repeated TestSuiteRun sub_suites = 20; // Nested test suite objects (optional, for sending full structure)

// type SuiteDocument struct {
// 	ID              string                 `bson:"id" json:"id"`
// 	RunID           string                 `bson:"run_id,omitempty" json:"runId,omitempty"`
// 	ParentSuiteID   string                 `bson:"parent_suite_id,omitempty" json:"parentSuiteId,omitempty"`
// 	Name            string                 `bson:"name,omitempty" json:"name,omitempty"`
// 	Description     string                 `bson:"description,omitempty" json:"description,omitempty"`
// 	Status          string                 `bson:"status,omitempty" json:"status,omitempty"`
// 	Metadata        map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
// 	Duration        *int64                 `bson:"duration,omitempty" json:"duration,omitempty"`
// 	Location        string                 `bson:"location,omitempty" json:"location,omitempty"`
// 	Type            string                 `bson:"type,omitempty" json:"type,omitempty"`
// 	TestSuiteSpecID string                 `bson:"test_suite_spec_id,omitempty" json:"testSuiteSpecId,omitempty"`
// 	InitiatedBy     string                 `bson:"initiated_by,omitempty" json:"initiatedBy,omitempty"`
// 	ProjectName     string                 `bson:"project_name,omitempty" json:"projectName,omitempty"`
// 	Author          string                 `bson:"author,omitempty" json:"author,omitempty"`
// 	Owner           string                 `bson:"owner,omitempty" json:"owner,omitempty"`
// 	TestCaseIds     []string               `bson:"test_case_ids,omitempty" json:"testCaseIds,omitempty"`
// 	SubSuiteIds     []string               `bson:"sub_suite_ids,omitempty" json:"subSuiteIds,omitempty"`
// 	StartTime       *time.Time             `bson:"start_time,omitempty" json:"startTime,omitempty"`
// 	EndTime         *time.Time             `bson:"end_time,omitempty" json:"endTime,omitempty"`
// 	CreatedAt       time.Time              `bson:"created_at" json:"createdAt"`
// 	UpdatedAt       time.Time              `bson:"updated_at" json:"updatedAt"`

// 	// Nested child suites
// 	Suites []*SuiteDocument `bson:"suites,omitempty" json:"suites,omitempty"`
// 	// Test cases within this suite
// 	Tests []*TestDocument `bson:"tests,omitempty" json:"tests,omitempty"`
// }
