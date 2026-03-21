# Feature: Add Race Query API Contract (Proto + Generated Bindings)

## Summary

Add Race read endpoints to the Query API contract and regenerate consumed Go bindings:

- `ListRaces`
- `GetRace`

This issue defines the API-level prerequisite required before implementing `services/query/races.go` in this repository.

## Why

The backend query layer cannot implement race handlers without generated request/response types and query service methods. Current generated Query bindings in this repository do not include race query RPCs.

## Goals

- Extend Query proto contract with race read RPCs.
- Add request/response messages for list/get race operations.
- Ensure response payloads use `common.v1.Race`.
- Regenerate and publish Buf modules consumed by this backend.
- Update backend dependency versions to the generated module commit that includes the new RPCs.

## Non-Goals

- Implementing backend repository/query handlers (`issue-query-races.md`).
- Implementing race command handlers (`issue-command-races.md`).
- Changing race write API shape (`CreateRace`, `UpdateRace`, `DeleteRace`).

## API Contract Requirements

1. **Query Service RPCs**
    - Add to `backend.query.v1.QueryService`:
        - `rpc ListRaces(ListRacesRequest) returns (ListRacesResponse);`
        - `rpc GetRace(GetRaceRequest) returns (GetRaceResponse);`

2. **Messages**
    - `ListRacesRequest`
        - `uint32 event_id` (optional filter; `0` means no filter)
    - `ListRacesResponse`
        - `repeated backend.common.v1.Race items`
    - `GetRaceRequest`
        - `uint32 id`
    - `GetRaceResponse`
        - `backend.common.v1.Race race`

3. **Common Model Dependency**
    - Ensure `backend.common.v1.Race` exists with fields expected by backend conversion:
        - `id`
        - `event_id`
        - `name`
        - `session_type`
        - `sequence_no`

## Implementation Plan

1. **Update API repository (srlmgr/api)**
    - Add RPC declarations to Query service proto.
    - Add new request/response messages in query v1 proto.
    - Reuse `common.v1.Race` as response payload type.

2. **Regenerate API artifacts**
    - Regenerate protobuf and Connect code in API pipeline.
    - Publish updated Buf generated modules.

3. **Update backend module dependencies**
    - In this repository, update:
        - `buf.build/gen/go/srlmgr/api/protocolbuffers/go`
        - `buf.build/gen/go/srlmgr/api/connectrpc/go`
    - Use:
        - `make update-bufbuild id=<api-commit-or-buf-ref>`
        - `go mod tidy`

4. **Wire authorization mapping for new query RPCs**
    - Add `ListRaces` and `GetRace` to query policy mapping in `authz/mapping.go` with:
        - `Capability: "query.read"`
        - `AllowAnonymous: true`

## Acceptance Criteria

- Generated package `backend/query/v1` exposes:
    - `ListRacesRequest`
    - `ListRacesResponse`
    - `GetRaceRequest`
    - `GetRaceResponse`
- Generated Connect Query handler interface includes:
    - `ListRaces(...)`
    - `GetRace(...)`
- Backend compiles after dependency update.
- Authorization mapping includes both new query methods as anonymous read endpoints.

## Verification

- `go doc buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1.ListRacesRequest`
- `go doc buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1.GetRaceRequest`
- `go doc buf.build/gen/go/srlmgr/api/connectrpc/go/backend/query/v1/queryv1connect.QueryServiceHandler`

## Follow-up

- After this issue is complete, execute `issue-query-races.md` to implement repository and service handlers.
