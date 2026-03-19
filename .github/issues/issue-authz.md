# Feature: Add Authorization to Backend (OPA + gRPC Interceptors)

## Summary

Implement authorization for all gRPC endpoints in this backend using policy-based access control with Open Policy Agent (OPA), following the architecture and policy model defined in github.com/srlmgr/concept.

## Why

The backend currently needs consistent, centralized authorization to protect write operations and scope read/write access by role and resource ownership.

The concept repository defines:

- External identity management (CMS/gateway) and internal authorization decisions
- Role and scope based access rules
- gRPC interceptor-driven enforcement
- OPA/Rego policy model and RPC-to-capability mapping

## Goals

- Enforce authorization on every gRPC call (CommandService, ImportService, AdminService, QueryService)
- Keep identity external; consume trusted identity and claims from request context/metadata
- Authorize with OPA using policy keys in service/method format
- Enforce both role capabilities and scope constraints (series and simulation)
- Return correct gRPC status codes for auth failures

## Non-Goals

- Building user management or authentication provider flows
- Replacing upstream identity provider responsibilities

## Roles and Access Model

Support these roles:

- anonymous
- season_operator
- series_operator
- master_data_operator
- administrator

High-level rules:

- anonymous: read-only query access
- season_operator: season/event/import/admin writes within assigned series scope
- series_operator: season_operator permissions + series writes within assigned simulation scope
- master_data_operator: simulation and global track/car master data writes; no import finalization/admin write
- administrator: full access

## Implementation Plan

1. Add authorization interceptor(s)

- Add unary (and stream if needed) gRPC interceptor for authorization.
- Extract identity claims from incoming context/metadata.
- Build policy input payload including subject, capability, policyKey, resource scope, and request metadata.

2. Add resource scope resolver

- Resolve resource hierarchy required for scope checks.
- Example: eventId -> seasonId -> seriesId -> simulationId.
- Add resolver logic for list endpoints with optional filters.

3. Add OPA policy integration

- Add OPA client/evaluation layer.
- Load and evaluate Rego policy package.
- Keep policy files versioned with backend code.
- Add optional short TTL decision cache (safe defaults, easy disable).

4. Add RPC-to-capability mapping

- Add explicit map for all protected RPCs.
- Use policy key convention: Service/Method.
- Enforce method-capability compatibility.

5. Wire into server startup

- Register interceptor chain in server bootstrap.
- Ensure no bypass paths for protected handlers.

6. Add observability and error handling

- Auth decision logs/metrics (allow/deny, role, capability, latency).
- Return Unauthenticated for missing/invalid identity.
- Return PermissionDenied for authenticated but unauthorized requests.

## Acceptance Criteria

- Every RPC is evaluated for authorization before handler execution.
- Role and scope rules are enforced according to concept docs.
- Anonymous users can only access allowed read endpoints.
- Season and series operators are denied outside their configured scopes.
- Master data operator cannot perform import finalization/admin writes.
- Administrator can access all mapped endpoints.
- Missing/invalid identity returns Unauthenticated.
- Denied policy decisions return PermissionDenied.
- OPA policy is versioned in repo and loaded at runtime.
- Unit and integration tests cover positive and negative authorization paths.

## Testing Requirements

- Unit tests:
    - claim extraction
    - policy input assembly
    - RPC-to-capability mapping
    - scope resolver behavior
- Integration tests:
    - interceptor + OPA decision flow end-to-end on representative RPCs
    - role matrix verification
    - scope boundary checks
    - list endpoint filtering behavior when no explicit scope filter is provided

## Suggested Task Breakdown

- [ ] Add authz package (claims, mapping, input model, opa client)
- [ ] Add gRPC interceptor and hook into server startup
- [ ] Implement resource scope resolver
- [ ] Add Rego policy and policy loading
- [ ] Add endpoint mapping for all gRPC methods
- [ ] Add tests for role/capability/scope matrix
- [ ] Add authz docs to README-dev

## References (concept repo)

- docs/architecture/03-api-design.md (authorization model, mapping, policy key convention)
- docs/architecture/04-system-architecture.md (security flow, OPA role/scope enforcement)
- docs/architecture/06-authorization-policy.md (OPA input contract, Rego starter policy, test cases)
- docs/01-project-requirements.md (business role expectations)
