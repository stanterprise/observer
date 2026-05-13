


-- Backfill existing runs with run_stats records based on actual test statuses
INSERT INTO run_stats (run_id, name, total, passed, failed, skipped, flaky, broken, timedout, interrupted, unknown, not_run, running, duration, created_at, updated_at)
SELECT
    r.id,
    r.name,
    COUNT(*) as total,
    COUNT(CASE WHEN t.status = 'PASSED' THEN 1 END) as passed,
    COUNT(CASE WHEN t.status = 'FAILED' THEN 1 END) as failed,
    COUNT(CASE WHEN t.status = 'SKIPPED' THEN 1 END) as skipped,
    COUNT(CASE WHEN t.status = 'FLAKY' THEN 1 END) as flaky,
    COUNT(CASE WHEN t.status = 'BROKEN' THEN 1 END) as broken,
    COUNT(CASE WHEN t.status = 'TIMEDOUT' THEN 1 END) as timedout,
    COUNT(CASE WHEN t.status = 'INTERRUPTED' THEN 1 END) as interrupted,
    COUNT(CASE WHEN t.status = 'UNKNOWN' THEN 1 END) as unknown,
    COUNT(CASE WHEN t.status = 'NOT_RUN' THEN 1 END) as not_run,
    COUNT(CASE WHEN t.status = 'RUNNING' THEN 1 END) as running,
    COALESCE(
        EXTRACT(
            EPOCH FROM (r.finished_at - r.started_at)
            ) * 1000,
            COALESCE(
                EXTRACT(
                    EPOCH FROM (r.updated_at - r.created_at)
                    ) * 1000, 0
                )
        )::BIGINT as duration,
    r.created_at,
    r.updated_at
FROM runs r
LEFT JOIN tests t ON t.run_id = r.id
GROUP BY r.id, r.name
ON CONFLICT (run_id) DO NOTHING;
