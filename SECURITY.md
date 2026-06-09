# Security Policy

## Supported Release Line

Security fixes are currently targeted at the latest public release line only.

At the moment that means:

- `0.1.x`: supported

## Reporting A Vulnerability

Please do not open public issues for suspected vulnerabilities.

Report privately to the maintainers with:

- a clear description of the issue
- affected version or commit
- reproduction steps if available
- impact assessment
- any proposed mitigation

Until a dedicated security contact channel is published, use a private maintainer contact path rather than public issue trackers.

## Security Expectations For 0.1.0

ALMS `0.1.0` uses a minimal shared-token model and is intended for trusted or protected environments. If you deploy it on an untrusted network, place it behind a reverse proxy, terminate TLS, and restrict access at the network layer.
