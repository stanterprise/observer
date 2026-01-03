import type { TestSuite } from "@/types/testSuite";
import TestCaseRecord from "./TestCaseRecord";

type TestSuiteRecordProps = {
  suite: TestSuite;
  filterSuiteTypes?: ("ROOT" | "PROJECT" | "SUBSUITE")[];
};

const TestSuiteRecord = ({ suite }: TestSuiteRecordProps) => {
  return (
    <div className="border border-gray-200 rounded-xl p-6 mb-4 bg-white shadow-sm hover:shadow-md transition-all duration-200">
      <div className="text-base font-semibold text-gray-900 mb-4 pb-3 border-b border-gray-200">
        {suite.name}
      </div>
      <div className="space-y-3">
        {suite.tests?.map((test) => (
          <TestCaseRecord key={test.id} test={test} runId={suite.runId} />
        ))}
      </div>
      <div className="mt-4 space-y-3">
        {suite.suites?.map((subsuite) => (
          <TestSuiteRecord key={subsuite.id} suite={subsuite} />
        ))}
      </div>
    </div>
  );
};

export default TestSuiteRecord;
