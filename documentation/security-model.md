# Security Model

## Current Security Posture

ALMS `0.1.0` uses a simple shared token model via `X-ALMS-TOKEN`. This is intentionally minimal and should be treated as suitable for trusted environments, private networks, local development, or deployments protected by a stronger outer layer.

## What This Means In Practice

Recommended baseline for production:

- run ALMS on a private network or behind a reverse proxy
- terminate TLS at the proxy or ingress
- require `X-ALMS-TOKEN`
- rotate the token through your secret-management process
- restrict who can reach the MCP endpoint

## Not Yet A Full Security Story

ALMS does not claim, in `0.1.0`, to provide:

- per-agent credentials
- RBAC
- multi-tenant isolation
- audit-grade authorization controls

That is acceptable for an early infrastructure project, but it should be explicit so adopters know the boundary.

## Threat Model Assumptions

`0.1.0` assumes:

- a trusted operator controls deployment
- a small set of trusted agents uses the server
- the network path is already protected or local

## Reporting Issues

See [../SECURITY.md](../SECURITY.md) for vulnerability reporting instructions.
