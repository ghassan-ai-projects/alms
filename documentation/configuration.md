# Configuration

## Sources

ALMS loads configuration in this order:

1. defaults compiled into the binary
2. explicit `-config` path if provided
3. first existing file in:
   - `~/.alms/alms.yaml`
   - `/etc/alms/alms.yaml`
   - `/opt/alms/alms.yaml`
4. environment variable overrides

## YAML Shape

Example:

```yaml
server:
  host: "0.0.0.0"
  port: 8001

database:
  dsn: "postgres://alms:alms@localhost:5432/alms_db?sslmode=disable"

auth:
  token: "change-me"
```

## Environment Variables

- `ALMS_PG_DSN`: overrides `database.dsn`
- `ALMS_AUTH_TOKEN`: overrides `auth.token`

## Defaults

- host: `127.0.0.1`
- port: `8001`
- DSN: `postgres://alms:alms@localhost:5432/alms_db?sslmode=disable`
- auth token: empty

## CLI Flags

- `-config`: load a specific config file
- `-migrate`: print migration guidance and exit
- `-version`: print version and commit and exit

## Production Guidance

- keep credentials in environment variables or a secret store
- bind ALMS behind a reverse proxy if exposing it beyond localhost
- terminate TLS at the proxy or ingress layer
- do not rely on the empty-token default outside local development
