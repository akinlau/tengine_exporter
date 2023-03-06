# Tengine Exporter for Prometheus

This is a simple server that periodically scrapes [tengine stats](http://tengine.taobao.org/document/http_upstream_check.html) and exports them via HTTP for Prometheus
consumption(defult use csv data format, -nginx.scrape_uri http://127.0.0.1/nginx_status?format=csv).

To run it:

```bash
./nginx_exporter [flags]
```

Help on flags:
```bash
./nginx_exporter --help
```
