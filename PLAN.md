# DIGITAL TWIN VERSION CONTROL IMPLEMENTATION PLAN

## Goal
Evolve the current digital twin subsystem from:
- current materialized graph in `dt_entities`
- entity-only revision history in `dt_entity_revisions`
- twin-layer reconciliation metadata embedded in entity `ComputedValues`

into a full temporal version-control system with:
- sync-run version anchors
- standalone relationship history
- run checkpoints/snapshots
- point-in-time reconstruction
- provenance-rich diffs
- frontend/operator workflows for history browsing and state comparison

This plan assumes the recently landed foundations already exist:
- triggerable ingestion via manual/webhook pipeline triggers
- entity history
- deterministic twin reconciliation (`source_priority`, `freshest`)

## Current implementation baseline

### Backend
Current backend behavior is concentrated in:
- `pkg/digitaltwin/service.go`
- `pkg/digitaltwin/processor.go`
- `pkg/digitaltwin/inference.go`
- `pkg/metadatastore/sqlitestore.go`
- `pkg/api/digitaltwin_handler.go`
- `pkg/api/routes_digitaltwins.go`
- `pkg/api/doc/schemas.go`

Current state model:
- `digital_twins` stores twin config/current metadata
- `dt_entities` stores current materialized entities
- `dt_entity_revisions` stores entity snapshots per save
- relationships are embedded in entity JSON, not temporalized independently
- predictions, scenarios, runs, alerts, actions are stored separately

Current gaps relevant to version control:
- no `sync_run` model anchoring temporal writes
- no standalone relationship history
- no checkpoint/snapshot table
- no reconstruction service or API
- no diff API
- no UI support for history, run diffs, or time-travel inspection
- no explicit ontology-version stamp on temporal records

### Frontend
Current twin UI lives primarily in:
- `frontend/pages/DigitalTwinsPage.js`

Current UI capabilities:
- twin list
- twin detail workspace
- overview cards
- entity list/get/update
- scenarios
- actions
- runs
- alerts
- automations
- query panel
- process/sync buttons

Current frontend limitations:
- no history tab
- no run timeline
- no entity revision browser
- no compare-selected-runs workflow
- no point-in-time state reconstruction view
- no surfacing of reconciliation provenance/conflicts except raw entity payloads if manually inspected

## Architecture principles

### 1. Keep hot-path reads fast
Normal operator/API reads should continue to use materialized current-state tables.
Time-travel and diff queries should be explicit and separate.

### 2. Use semantic temporal records, not git-like text diffs
The system state is graph/entity based, not source-code text. We should store:
- entity revisions
- relationship revisions
- sync-run metadata
- checkpoints
- provenance
not line-oriented diffs.

### 3. Use append-only history plus periodic checkpoints
This gives:
- full auditability
- efficient storage growth
- bounded replay cost
- simpler operational semantics than full cloning

### 4. Reserve branch semantics for scenarios
Operational history should not be modeled as full twin clones.
Scenario branches can later reference base snapshots and overlays.

## Target backend design

## A. Sync-run anchors

### New model
Add to `pkg/models/digitaltwin.go`:
- `TwinSyncRun` or `TwinVersionRun`

Suggested fields:
- `ID string`
- `DigitalTwinID string`
- `TriggerType string` (`manual`, `pipeline_completed`, `webhook`, `scheduled`, `system`)
- `TriggeredBy string`
- `SourceIDs []string`
- `OntologyVersion string`
- `ReconciliationStrategy string`
- `StartedAt time.Time`
- `CompletedAt *time.Time`
- `Status string` (`running`, `completed`, `failed`)
- `BaseSnapshotID string`
- `EntityRevisionHighWatermark int64`
- `RelationshipRevisionHighWatermark int64`
- `Summary map[string]interface{}`

### Store changes
Add to `pkg/metadatastore/store.go`:
- `SaveTwinSyncRun(run *models.TwinSyncRun) error`
- `GetTwinSyncRun(id string) (*models.TwinSyncRun, error)`
- `ListTwinSyncRuns(twinID string, limit int) ([]*models.TwinSyncRun, error)`

Add SQLite table in `pkg/metadatastore/sqlitestore.go`:
- `dt_sync_runs`
with indices on:
- `twin_id`
- `started_at`
- `status`

### Service integration
In `pkg/digitaltwin/service.go` and/or `pkg/digitaltwin/processor.go`:
- every explicit sync/process operation creates a sync-run record first
- all temporal writes during the run reference `sync_run_id`
- successful completion updates high-watermarks and final summary

## B. Relationship temporalization

### New model
Add to `pkg/models/digitaltwin.go`:
- `RelationshipRevision`

Suggested fields:
- `ID string`
- `DigitalTwinID string`
- `SourceEntityID string`
- `TargetEntityID string`
- `RelationshipType string`
- `Revision int`
- `SyncRunID string`
- `ChangeType string` (`added`, `updated`, `removed`)
- `DeltaData map[string]interface{}`
- `FullState map[string]interface{}` optional
- `Provenance map[string]interface{}`
- `RecordedAt time.Time`
- `OntologyVersion string`

### Store changes
Add to `pkg/metadatastore/store.go`:
- `SaveRelationshipRevision(rev *models.RelationshipRevision) error`
- `ListRelationshipRevisions(twinID string, entityID string, limit int) ([]*models.RelationshipRevision, error)`

Add SQLite table in `pkg/metadatastore/sqlitestore.go`:
- `dt_relationship_revisions`
with indices on:
- `twin_id`
- `source_entity_id`
- `target_entity_id`
- `recorded_at`

### Service integration
In `pkg/digitaltwin/service.go` `wireRelationships(...)`:
- detect relationship add/update/remove events relative to currently materialized relationships
- write revision records before or alongside saving the updated entity graph
- avoid duplicate history writes for unchanged relationships

## C. Snapshot/checkpoint layer

### New model
Add to `pkg/models/digitaltwin.go`:
- `TwinSnapshot`

Suggested fields:
- `ID string`
- `DigitalTwinID string`
- `SyncRunID string`
- `SnapshotKind string` (`full`, `checkpoint`, `branch_base`)
- `EntityState []byte` or string blob
- `RelationshipState []byte` or string blob
- `CreatedAt time.Time`
- `EntityRevisionHighWatermark int64`
- `RelationshipRevisionHighWatermark int64`
- `Metadata map[string]interface{}`

### Store changes
Add to `pkg/metadatastore/store.go`:
- `SaveTwinSnapshot(snapshot *models.TwinSnapshot) error`
- `GetTwinSnapshot(id string) (*models.TwinSnapshot, error)`
- `ListTwinSnapshots(twinID string, limit int) ([]*models.TwinSnapshot, error)`
- `GetLatestTwinSnapshotBeforeRun(twinID, syncRunID string) (*models.TwinSnapshot, error)`
- `GetLatestTwinSnapshotBeforeTime(twinID string, at time.Time) (*models.TwinSnapshot, error)`

SQLite table in `pkg/metadatastore/sqlitestore.go`:
- `dt_snapshots`
with indices on:
- `twin_id`
- `created_at`
- `sync_run_id`

### Snapshot format
Initial implementation should use:
- compressed JSON blobs of entity and relationship state
- no premature normalization

Compression approach:
- gzip or zstd if available in stdlib-adjacent acceptable tooling
- start with gzip for simplicity if needed
- otherwise plain JSON first, compression second

### Snapshot cadence
Initial recommendation:
- one full checkpoint per successful sync run

Reason:
- simplest reconstruction semantics
- easiest debugging
- acceptable until run count or twin size becomes large

Later optimization:
- configurable cadence
- only checkpoint when changed-entity count exceeds threshold

## D. Entity history evolution

Current `dt_entity_revisions` stores full snapshots.
Do not replace it abruptly.

### Short-term evolution
Add columns/fields to support richer semantics while remaining backward-compatible:
- `sync_run_id`
- `change_type`
- `delta_data`
- `full_state`
- `provenance`
- `ontology_version`
- `parent_revision`

### Write behavior
For new writes:
- store `delta_data` for changed fields
- optionally continue storing `full_state` during transition
- attach `sync_run_id`, reconciliation metadata, and ontology version

### Migration strategy
Existing rows remain valid legacy full snapshots.
No immediate expensive backfill required.
A compatibility decoder can interpret:
- old rows as full snapshots
- new rows as delta/full hybrid entries

## E. Reconstruction service

### New service layer
Add to `pkg/digitaltwin/service.go` or a new file like:
- `pkg/digitaltwin/history.go`
- `pkg/digitaltwin/reconstruct.go`

New service methods:
- `GetTwinStateAtRun(twinID, runID string) (*models.TwinStateSnapshot, error)`
- `GetTwinStateAtTime(twinID string, at time.Time) (*models.TwinStateSnapshot, error)`
- `DiffTwinStateByRun(twinID, fromRunID, toRunID string) (*models.TwinStateDiff, error)`

### Reconstruction algorithm
Given `at_run`:
1. load latest snapshot at or before run
2. load entity revisions after snapshot watermark and up to run watermark
3. load relationship revisions after snapshot watermark and up to run watermark
4. replay into in-memory graph maps
5. emit reconstructed graph

### Data structures
Use in-memory maps keyed by:
- `entity_id`
- relationship composite key `(source_entity_id, relationship_type, target_entity_id)`

This avoids replay becoming O(n²).

## F. Provenance model

### What every temporal write should carry
For entities and relationships:
- `source_storage_ids`
- source CIR URI / record ID when available
- `sync_run_id`
- `trigger_type`
- `triggered_by`
- `reconciliation_strategy`
- winning source per changed attribute
- losing/conflicting candidates where applicable
- ontology version

### Practical shape
Initially provenance can remain embedded JSON in revision rows.
Defer normalization to a dedicated provenance table until the shape stabilizes and volume justifies it.

## API plan

## New backend routes
Add to `pkg/api/routes_digitaltwins.go` and `pkg/api/digitaltwin_handler.go`:

### Run history
- `GET /api/digital-twins/{id}/history/runs?limit=N`
- `GET /api/digital-twins/{id}/history/runs/{runId}`

### Snapshots
- `GET /api/digital-twins/{id}/history/snapshots?limit=N`
- `GET /api/digital-twins/{id}/history/snapshots/{snapshotId}`

### Relationship history
- `GET /api/digital-twins/{id}/relationships/history?entity_id=...&limit=N`

### Point-in-time reconstruction
- `GET /api/digital-twins/{id}/state?at_run=...`
- `GET /api/digital-twins/{id}/state?at_time=...`

### Diff API
- `GET /api/digital-twins/{id}/diff?from_run=...&to_run=...`
- optional later: `from_time` / `to_time`

## Existing route interactions
Current routes remain:
- `/sync`
- `/runs`
- `/entities`
- `/entities/{entityId}`
- `/entities/{entityId}/history`

The new APIs should build on current twin identity and run semantics, not replace them.

## Frontend plan

Primary file:
- `frontend/pages/DigitalTwinsPage.js`

### New tabs / sections
Add tabs such as:
- `Overview`
- `Entities`
- `History`
- `Runs`
- `Diff`
- existing `Scenarios`, `Actions`, etc.

### History tab behavior
Show:
- run timeline table
- latest snapshots
- selected run metadata
- reconstruction state retrieval controls
- entity history browser

### Diff workflow
Operator flow:
1. select twin
2. open `Diff`
3. choose `from run` and `to run`
4. frontend calls diff API
5. render sections:
   - entity created / updated / removed counts
   - changed attributes grouped by entity
   - relationship adds/removes
   - reconciliation conflicts introduced/resolved

### Entity history browser
On entity details panel:
- add “History” button or sub-tab
- fetch `/entities/{entityId}/history`
- show revision timeline with:
  - revision number
  - recorded time
  - changed attributes summary
  - source/provenance summary
- allow compare selected revisions

### Point-in-time state UX
On History tab:
- allow selecting a run or timestamp
- fetch reconstructed twin state
- render read-only entity/relationship table view
- never mutate live state from historical views

### Frontend API client impact
Likely files:
- `frontend/lib/api.js` unchanged structurally
- `frontend/pages/DigitalTwinsPage.js` new data fetches and UI flows
- possibly primitive table/modal components reused, not rewritten

## File-by-file implementation plan

### `pkg/models/digitaltwin.go`
Add:
- `TwinSyncRun`
- `TwinSnapshot`
- `RelationshipRevision`
- `TwinStateSnapshot`
- `TwinStateDiff`
- enriched `EntityRevision`

### `pkg/metadatastore/store.go`
Add store methods for:
- sync runs
- relationship revisions
- snapshots
- reconstruction lookups

### `pkg/metadatastore/sqlitestore.go`
Add:
- new schemas
- new read/write methods
- high-watermark lookup queries
- snapshot retrieval queries
- relationship revision retrieval

Critical requirement:
- use real upsert semantics, not `INSERT OR REPLACE`, to avoid FK cascade deletion of history

### `pkg/digitaltwin/service.go`
Add:
- sync-run lifecycle creation/update
- dual-write logic for current state + temporal records
- snapshot creation at sync completion
- reconstruction entrypoints
- diff generation

Refactor likely needed around:
- `SyncWithStorage`
- `syncFromStorage`
- `wireRelationships`
- reconciliation metadata write paths

### `pkg/digitaltwin/processor.go`
Add:
- run/twin processing linkage to sync-run IDs where relevant
- alert/action records carrying version references

### `pkg/api/digitaltwin_handler.go`
Add handlers for:
- run history
- snapshots
- relationship history
- reconstructed state
- diff

### `pkg/api/routes_digitaltwins.go`
Document all new endpoints.

### `pkg/api/doc/schemas.go`
Add schemas for:
- `TwinSyncRun`
- `TwinSnapshot`
- `RelationshipRevision`
- `TwinStateSnapshot`
- `TwinStateDiff`
- enriched `EntityRevision`

### `frontend/pages/DigitalTwinsPage.js`
Extend with:
- History tab
- Diff tab
- run selection state
- snapshot list
- reconstructed state viewer
- revision comparison UI

## Scalability considerations

### 1. Replay cost
Problem:
- replaying from genesis will become too expensive

Mitigation:
- require snapshots/checkpoints
- high-watermark based replay
- one snapshot per sync run initially

### 2. Storage growth
Problem:
- large twins with frequent syncs will create many revisions

Mitigation:
- move new writes toward delta-focused revisions
- only checkpoint per run, not full clone per change
- optionally compress snapshots
- later archive old runs/snapshots by retention policy

### 3. Query performance
Problem:
- diff and history queries can become slow

Mitigation:
- indices on `twin_id`, `entity_id`, `recorded_at`, `sync_run_id`
- paginate history APIs
- keep current-state reads separate from historical reads

### 4. Frontend payload size
Problem:
- reconstructed twin states may be large

Mitigation:
- server-side pagination/filtering for state views
- summary-first diff endpoints
- lazy entity detail loading instead of rendering full graph payloads immediately

## Security considerations

### 1. Historical data sensitivity
History can expose:
- superseded values
- conflict candidates
- source provenance
- potentially sensitive operational details

Required controls:
- apply the same authz checks as current twin reads
- do not expose hidden secrets/tokens in provenance
- redact sensitive fields if source payloads may contain secrets

### 2. Webhook-to-history linkage
Webhook-triggered runs should preserve provenance but not store raw secrets.
Only store:
- trigger type
- trigger source label
- source_event_id
- maybe hashed token identifier if ever needed, not raw token

### 3. Time-travel reads must be read-only
Historical reconstruction endpoints must never mutate current state.
Keep reconstruction entirely in-memory/read-path only.

### 4. Denial-of-service risk
Historical diff/replay can be expensive.
Mitigations:
- limit ranges
- require pagination/limits
- cap maximum reconstruction window if no checkpoint exists
- add request timeout safeguards on broad history queries

## Rollout sequence

### Slice 1
- `dt_sync_runs`
- sync-run creation/update
- API to list/get runs

### Slice 2
- `dt_relationship_revisions`
- temporal writes from `wireRelationships`
- relationship history API

### Slice 3
- `dt_snapshots`
- snapshot creation at successful sync completion
- snapshot list API

### Slice 4
- reconstruction service
- `state?at_run=` API
- frontend history tab initial integration

### Slice 5
- diff service/API
- frontend diff tab

### Slice 6
- richer entity delta revisions
- optional compatibility/backfill for old entity revision rows

## Testing plan

### Unit
- sync run persistence
- relationship revision persistence
- snapshot save/load
- reconstruction replay correctness
- diff generation correctness
- provenance payload correctness

### Integration
- sync same twin twice, reconstruct both versions, assert differences
- webhook-triggered sync produces history tied to trigger provenance
- reconciliation conflict appears in temporal records and diff output
- snapshot replay equals current materialized state for that run

### Frontend
Given current frontend architecture, initially rely on API contract tests and manual/browser validation, then add focused JS tests only if a test harness is introduced.

## Recommended next implementation step
Implement this next:
1. `dt_sync_runs`
2. backend handlers/routes for run history
3. tie `SyncWithStorage` / explicit processing runs to sync-run IDs

That is the cleanest first move into full version control because every later temporal record needs a stable run anchor.
