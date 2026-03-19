# Feature: Add Authentication to Backend (JWT + gRPC Interceptors)

## Summary

Implement request authentication for all gRPC endpoints in this backend by validating either bearer JWT tokens or api-token metadata values, extracting trusted identity claims, and attaching a normalized principal to request context for downstream authorization.

## Why

The backend needs a consistent and centralized authentication layer before authorization can make reliable policy decisions.

Today, identity is expected from upstream systems, but the backend should still verify token integrity, issuer, and audience so service methods do not rely on untrusted metadata.

The service also needs a practical path for non-IDP clients via filesystem-managed api-tokens that resolve to trusted claims.

## Goals

- Authenticate every gRPC call before handler execution.
- Support anonymous access only for explicitly allowed read endpoints.
- Produce correct gRPC status codes and observable authn diagnostics.
- Authentication sources may be:
    - JWT provided by some IDP
    - custom token in gRPC metadata identified by "api-token" key
- Authentication needs the following attributes
    - id (the user id)
    - tenant
    - associated roles
    - scopes
        - simulationsIDs
        - seriesIDs

### JWT

- Validate bearer JWTs from incoming metadata.
- Verify token signature, issuer, audience, time-based claims, and subject.
- Normalize identity claims into a backend principal model in context.

### API-Token

- Validate against data read from filesystem
- use claims
- Normalize identity claims into a backend principal model in context.

## Non-Goals

- Building a login UI or user credential workflows.
- Managing users, groups, or identity lifecycle in this service.
- Replacing upstream identity provider responsibilities.

## Authentication Model

- Support two authentication sources:
    - JWT from Authorization metadata using Bearer scheme
    - api-token from gRPC metadata key api-token
- JWT flow:
    - identity provider remains external (CMS/gateway/IdP)
    - backend trusts only validated JWTs signed by configured keys
- api-token flow:
    - token is looked up in a filesystem-backed token store
    - token record includes claims used to build principal attributes
- Principal context should include at minimum:
    - id
    - tenant
    - roles
    - scopes
        - simulationsIDs
        - seriesIDs
- If both Authorization and api-token are present in the same request, fail authentication to avoid ambiguous identity.

## Implementation Plan

1. Add authentication interceptor(s)

- Add unary (and stream if needed) gRPC interceptor that runs before authorization.
- Extract Authorization metadata and api-token metadata.
- Select and validate exactly one auth source per request.
- Reject malformed or missing tokens for protected endpoints.

2. Add JWT validation layer

- Implement validator package to verify:
    - signature against configured JWKS/public keys
    - issuer and audience
    - exp, nbf, iat with configurable clock skew
    - required claims for principal mapping (id/sub, tenant, roles, scopes)
- Support key rotation via periodic JWKS refresh with safe cache TTL.

3. Add api-token validation layer

- Implement filesystem-backed token provider to load trusted token records and claims.
- Validate api-token presence and active status against configured token file(s).
- Support reload strategy (startup load and configurable refresh/reload mechanism).
- Enforce strict parsing and reject malformed or duplicated token entries.

4. Add principal mapping and context wiring

- Map JWT claims and api-token claims to a single typed principal struct.
- Attach principal to context for use by authz interceptor and handlers.
- Keep mapping strict and explicit to avoid ambiguous claim usage.

5. Add endpoint auth requirements mapping

- Define which RPCs allow anonymous access.
- Require valid principal for all other RPCs.
- Ensure interceptor chain order: authentication first, authorization second.

6. Wire into server startup and config

- Add config for issuer, audience, JWKS URL/public keys, cache settings, and api-token file path/reload settings.
- Register authn interceptor in server bootstrap.
- Fail startup on invalid authn configuration for non-dev environments.

7. Add observability and error handling

- Emit metrics/logs for authn outcomes (source, success/failure reason, latency).
- Return Unauthenticated for missing/invalid tokens.
- Avoid leaking sensitive token contents in logs/errors.

## Acceptance Criteria

- Every RPC passes through authentication before business handlers.
- Protected RPCs return Unauthenticated when credentials are missing or invalid.
- JWT tokens with invalid signature/issuer/audience/time claims are denied.
- api-token values not present/active in filesystem token store are denied.
- Valid JWT and valid api-token requests produce the same normalized principal shape in context.
- Requests carrying both Authorization and api-token are denied as invalid/ambiguous authentication input.
- Endpoints explicitly marked anonymous remain callable without a token.
- Authentication runs before authorization in interceptor chain.
- Authn diagnostics are visible in logs/metrics without exposing secrets.
- Unit and integration tests cover core positive/negative authn paths.

## Testing Requirements

- Unit tests:
    - auth source selection from metadata (Authorization vs api-token)
    - rejection when both auth sources are present
    - bearer token extraction from metadata
    - JWT validation cases (signature, issuer, audience, exp/nbf/iat)
    - api-token lookup/validation from filesystem data
    - principal claim mapping for both JWT and api-token
    - endpoint auth requirement mapping
- Integration tests:
    - interceptor chain behavior (authn then authz)
    - protected endpoint rejects unauthenticated requests
    - representative endpoints accept valid JWT-authenticated requests
    - representative endpoints accept valid api-token authenticated requests
    - anonymous endpoint behavior

## Suggested Task Breakdown

- [ ] Add authn package (source selector, validator, principal model)
- [ ] Add gRPC authentication interceptor and chain ordering
- [ ] Add JWT/JWKS validation and claim mapping
- [ ] Add filesystem api-token provider and validation
- [ ] Add configuration for JWT and api-token sources
- [ ] Add anonymous vs protected RPC mapping
- [ ] Add authn metrics/logging
- [ ] Add unit and integration tests
- [ ] Add authn docs to README-dev

## Open Questions

- Which exact issuer and audience values should be enforced per environment?
- What is the canonical schema for filesystem api-token entries?
- Which role/scope claim names should be canonical across JWT and api-token claims?
- Do we need service-to-service token support with separate audience rules?
- Should local development allow a mock verifier mode behind explicit config?
