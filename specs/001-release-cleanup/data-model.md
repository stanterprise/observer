# Data Model: Release Readiness Cleanup

## Entities

### ReleaseReadinessChecklist

- **Fields**
  - id (string, required)
  - version (string, required)
  - generated_at (datetime, required)
  - scope (string, required) — bounded cleanup scope definition
  - overall_status (enum: pass | fail | blocked | approve-with-risk, required)
  - decision (object, required)
    - status (enum: approve | block | approve-with-risk)
    - rationale (string, required)
    - decided_by (string, required)
    - decided_at (datetime, required)
  - gates (array of Gate, required, min 1)
  - evidence_index_path (string, required)

- **Validation Rules**
  - Every functional requirement maps to at least one gate.
  - overall_status must align with gate statuses (any fail => fail/blocked).

### Gate

- **Fields**
  - id (string, required)
  - name (string, required)
  - description (string, required)
  - status (enum: pass | fail | na, required)
  - owner (string, required)
  - evidence_paths (array of strings, required)
  - remediation_task_ids (array of string, optional)
  - risk_ids (array of string, optional)

- **Validation Rules**
  - status=fail requires at least one remediation_task_id.

### CleanupTask

- **Fields**
  - id (string, required)
  - title (string, required)
  - priority (enum: P0 | P1 | P2 | P3, required)
  - owner (string, required)
  - status (enum: open | in-progress | blocked | done, required)
  - acceptance_criteria (array of string, required)
  - related_gate_id (string, required)
  - due_date (date, optional)

### RiskRegisterItem

- **Fields**
  - id (string, required)
  - severity (enum: low | medium | high | critical, required)
  - impact (string, required)
  - mitigation (string, optional)
  - acceptance_rationale (string, optional)
  - related_gate_ids (array of string, required)

### DocumentationSet

- **Fields**
  - id (string, required)
  - name (string, required)
  - paths (array of string, required)
  - owner (string, required)
  - last_verified_at (datetime, optional)

### LicenseInventory

- **Fields**
  - id (string, required)
  - generated_at (datetime, required)
  - dependencies (array of DependencyLicense, required)

### DependencyLicense

- **Fields**
  - name (string, required)
  - version (string, required)
  - license (string, required)
  - obligations (array of string, optional)
  - source (enum: go.mod | package.json | other, required)

## Relationships

- ReleaseReadinessChecklist.gates references Gate entries.
- Gate.remediation_task_ids references CleanupTask.
- Gate.risk_ids references RiskRegisterItem.
- CleanupTask.related_gate_id references Gate.
- RiskRegisterItem.related_gate_ids references Gate.
- DocumentationSet and LicenseInventory are referenced by the checklist evidence index.
