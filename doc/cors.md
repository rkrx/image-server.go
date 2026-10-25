# Cross-Origin Resource Sharing (CORS)

CORS is disabled by default. When enabled, preflight (`OPTIONS`) requests are
answered and matching responses receive the configured `Access-Control-*`
headers. The server rejects preflight requests from origins that are not on the
allowlist with `403`.

```toml
[cors]
enabled = true
allowedOrigins = ["https://app.example.com"]
allowedMethods = ["GET", "HEAD"]
allowedHeaders = ["Accept"]
exposedHeaders = ["Content-Length", "Content-Type"]
allowCredentials = false
maxAgeInSeconds = 600
```

| Setting            | Purpose                                                                                                                                              |
|--------------------|------------------------------------------------------------------------------------------------------------------------------------------------------|
| `enabled`          | Master switch. When `false`, no CORS headers are added and `OPTIONS` returns `405`.                                                                  |
| `allowedOrigins`   | Explicit list of origins, or `["*"]` for a fully public API. `*` cannot be combined with `allowCredentials`. Required when `enabled` is `true`.      |
| `allowedMethods`   | Methods advertised in preflight responses. Defaults to `GET`, `HEAD`.                                                                                |
| `allowedHeaders`   | Request headers advertised in preflight responses. When empty, the server reflects the client's `Access-Control-Request-Headers`.                    |
| `exposedHeaders`   | Response headers that the browser is allowed to read.                                                                                                |
| `allowCredentials` | When `true`, the response includes `Access-Control-Allow-Credentials: true`. Image responses are usually credential-less; keep this `false`.         |
| `maxAgeInSeconds`  | Browser cache lifetime for preflight responses. Must be between `0` and `86400`.                                                                     |

Specific origins are reflected into `Access-Control-Allow-Origin` and the server
adds `Vary: Origin` so caches keep per-origin entries separate. The wildcard
origin (`*`) is sent verbatim and does not add a `Vary: Origin` header.
