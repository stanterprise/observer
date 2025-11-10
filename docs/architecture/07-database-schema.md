# Database Schema

## Core Tables

| Table | Purpose |
|--------|----------|
| `runs` | Test suite executions |
| `tests` | Individual test cases |
| `steps` | Detailed steps of tests |
| `artifacts` | Screenshots, logs, etc. |
| `signals` | Metrics and counters |

## Example Columns

**runs**  
- id, suite, branch, sha, actor  
- start_time, end_time, status, duration

**tests**  
- id, run_id, name, file, status, duration

**artifacts**  
- id, test_id, kind, uri, sha256

## Indexes
- `tests(run_id)`  
- `runs(branch, created_at DESC)`  
- `artifacts(test_id)`

## Migrations
Handled via `golang-migrate` or `atlas`.
