# domain_exporter

Push the expiration time of your domains as prometheus metrics to remote write.

My use case is to run it once a day and monitor it with Alertmanager or Grafana.

## Configuration

Not supported without basic auth.

```yaml
prometheus:
  url: ""
  user: ""
  pass: ""
domains:
  - google.com
```

And pass file path as argument to `domain_exporter`:

```bash
domain_exporter --config=config.yaml
```
