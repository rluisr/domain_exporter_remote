# domain_exporter

Push the expiration time of your domains as prometheus metrics to remote write.

My use case is to run it once a day and monitor it with Alertmanager or Grafana.

## Configuration

Not supported without basic auth.

```yaml
prometheus:
  url: "http://prometheus:9090/api/v1/write" # Prometheus remote write endpoint URL
  user: "username" # Basic auth username
  pass: "password" # Basic auth password
  # Note: URL must be a valid Prometheus remote write endpoint
  # Basic auth credentials must be provided
domains:
  - google.com
```

And pass file path as argument to `domain_exporter`:

```bash
domain_exporter --config=config.yaml
```
