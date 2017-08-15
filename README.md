# Tengine Exporter for Prometheus

This is a simple server that periodically scrapes [tengine stats](http://tengine.taobao.org/document/http_upstream_check.html) and exports them via HTTP for Prometheus
consumption.

To run it:

```bash
./nginx_exporter [flags]
```

Help on flags:
```bash
./nginx_exporter --help
```
