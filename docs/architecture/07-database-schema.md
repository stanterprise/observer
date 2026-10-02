# Database Schema

PostgreSQL is the source of truth for durable run data and API reads. Current models are defined in [`internal/models/relational.go`](../../internal/models/relational.go), and SQL migrations under [`migrations/`](../../migrations/) are authoritative. MongoDB is used only for transient `live_step_buffers` data.

## Core Tables

| Table            | Purpose                                                                                |
| ---------------- | -------------------------------------------------------------------------------------- |
| `runs`           | Logical test-run executions                                                            |
| `run_executions` | Individual executions and shard metadata                                               |
| `suites`         | Test suite hierarchy                                                                   |
| `tests`          | Individual logical test cases                                                          |
| `test_attempts`  | Test attempt status, timing, steps, failures, and output                               |
| `attachments`    | Attachment metadata and storage keys                                                   |
| `run_stats`      | Aggregated run statistics                                                              |
| `steps`          | Conceptual test-step data; stored in `test_attempts.steps` JSONB, not a separate table |
| `signals`        | Metrics/counters were a design concept; no dedicated table exists in the current model |

## Example Columns

**runs**

- id, name, status, metadata, initiated_by, project_name
- started_at, finished_at, duration
- Earlier examples referred to branch, sha, and actor; these are not dedicated columns in the current `runs` model and may be supplied through metadata.

**tests**

- id, run_id, suite_id, name, title, status, duration
- location, retry_count, retry_index, metadata

**test_attempts**

- id, run_id, test_id, execution_id, attempt_index
- status, started_at, finished_at, duration, steps

**attachments**

- id, test_id, test_attempt_id, kind, storage_key, checksum
- Earlier design notes called the table `artifacts` and used `uri` / `sha256`; current relational models use `attachments` and storage metadata.

## Indexes

- `tests(run_id)`
- `runs(status, started_at)` (the former `runs(branch, created_at)` example is not a current index)
- `attachments(test_id)`
- Additional indexes are maintained in model tags and SQL migrations.

## MongoDB Live-Step Buffer

MongoDB stores transient live step data in the `live_step_buffers` collection. It is not the durable run-data store; completed run records remain in PostgreSQL.

## Migrations

Migrations are checked-in SQL files applied by the repository migration runner (`cmd/migrate`). The schema is not managed by `golang-migrate` or Atlas.
