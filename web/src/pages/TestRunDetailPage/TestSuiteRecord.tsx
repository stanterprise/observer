import type { TestSuite } from "@/types/testSuite";
import TestCaseRecord from "./TestCaseRecord";

type TestSuiteRecordProps = {
  suite: TestSuite;
  filterSuiteTypes?: ("ROOT" | "PROJECT" | "SUBSUITE")[];
};

const TestSuiteRecord = ({ suite }: TestSuiteRecordProps) => {
  return (
    <div className="border border-gray-300 rounded-lg p-4 mb-4 pr-0">
      <div>{suite.name}</div>
      <div className="space-y-3">
        {suite.tests?.map((test) => (
          <TestCaseRecord key={test.id} test={test} runId={suite.runId} />
        ))}
      </div>
      <div>
        {suite.suites?.map((subsuite) => (
          <TestSuiteRecord key={subsuite.id} suite={subsuite} />
        ))}
      </div>
    </div>
  );
};

export default TestSuiteRecord;
