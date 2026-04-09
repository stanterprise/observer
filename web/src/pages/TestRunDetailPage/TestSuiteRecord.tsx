import type { TestSuite } from "@/types/testSuite";
import { TagList } from "@/components/TagList";
import TestCaseRecord from "./TestCaseRecord";

type TestSuiteRecordProps = {
  suite: TestSuite;
  hiddenSuiteTypes?: Set<string>;
};

const TestSuiteRecord = ({ suite, hiddenSuiteTypes }: TestSuiteRecordProps) => {
  const isHidden =
    hiddenSuiteTypes &&
    suite.type &&
    hiddenSuiteTypes.has(suite.type.toUpperCase());

  // If suite is hidden, render children directly without the suite wrapper
  if (isHidden) {
    return (
      <>
        {/* Render tests from hidden suite */}
        {suite.tests?.map((test) => (
          <div key={test.id} className="mb-3">
            <TestCaseRecord test={test} runId={suite.runId} />
          </div>
        ))}
        {/* Render child suites (which may also be hidden) */}
        {suite.suites?.map((subsuite) => (
          <TestSuiteRecord
            key={subsuite.id}
            suite={subsuite}
            hiddenSuiteTypes={hiddenSuiteTypes}
          />
        ))}
      </>
    );
  }

  // Normal rendering when suite is visible
  return (
    <div className="mb-4 rounded-xl border border-(--stitch-outline) bg-(--stitch-surface-card) p-6 transition-colors hover:bg-(--stitch-surface-low)">
      <div className="mb-4 border-b border-(--stitch-outline) pb-3">
        <div className="mb-2 text-base font-semibold text-(--stitch-on-surface)">
          {suite.name}
        </div>
        {suite.tags && suite.tags.length > 0 && (
          <TagList tags={suite.tags} />
        )}
      </div>
      <div className="space-y-3">
        {suite.tests?.map((test) => (
          <TestCaseRecord key={test.id} test={test} runId={suite.runId} />
        ))}
      </div>
      <div className="mt-4 space-y-3">
        {suite.suites?.map((subsuite) => (
          <TestSuiteRecord
            key={subsuite.id}
            suite={subsuite}
            hiddenSuiteTypes={hiddenSuiteTypes}
          />
        ))}
      </div>
    </div>
  );
};

export default TestSuiteRecord;
