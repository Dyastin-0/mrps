# MRPS

`mrps` is a simple HTTP->HTTPS reverse proxy server.

## Features

- Dynamic routing
- Automatic HTTPS
- Configurable routing rules
- Path rewrites
- Global and domain-based rate limiting
- Scrappable metrics

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) installed on your machine
- Port 80 and 443 available for HTTP/HTTPS traffic

#### Environment

`.env` is used when mrps API is enabled.

```
AUTH_EMAIL=your@mail.com
AUTH_PASSWORD=12345
REFRESH_TOKEN_KEY=12321
ACCESS_TOKEN_KEY=32123
```

### Configuration

Configurations are stored on the root's `config.yaml`.

#### miscellaneous

```yaml
Misc:
  email: your@mail.com       # Used for certmagic (optional)
  enable_metrics: true
  metrics_port: 5000         # Default 7070
  enable_api: true
  api_port: 3000             # Default 6060
  allowed_origins:           # Used for WS connections
  - https://your_domain.com
  - http://localhost:5050
```

#### Route Configuration

Routes define how incoming requests are routed to different services.

Basic route:

```yaml
domains:
  domain.com:
    routes:
      /api:
        dests:
        - http://localhost:3000
      /:
        dests:
        - http://localhost:9090
```

#### Rate Limiting Configuration

Rate limiting defines how many request a client can do in a specified timeframe, applicable to domain and global scope.

##### Global

```yaml
rate_limit:
  burst: 50
  rate: 10
  cooldown: 60000
```

##### Domain

```yaml
domains:
  domain.com:
    routes:
      /:
        dests:
        - url: http://localhost6060
    rate_limit:
      burst: 50
      rate: 10
      cooldown: 60000
```

`burst` Maximum requests allowed in a short period.

`rate` Requests per second at which tokens are replenished.

`cooldown` Time (in milliseconds) a client must wait after exhausting the burst limit.

#### Path Rewrites

There's two types of rewrites available, `regex` and `prefix`.

```yaml
domains:                                       
  domain.com:                                                       
      routes:                                  
        /api/v1:                               
            dests:
            - http://localhost:3001
            rewrite:
              type: regex
              value: ^/api/v1/(.*)$          # <- will be rewritten to /
              replace_val: /$1
        /metrics:
            dests:
            - http://localhost:8080  
  sub.domain.com:                                                       
      routes:                                  
        /api/v1:                               
            dests:
            - http://localhost:3001
            rewrite:
              type: prefix
              value: /api/v1                 # <- will be rewritten to /new/path
              replace_val: /new/path
        /metrics:
            dests:
            - http://localhost:8080" 
```

### Load-balancing

There three (3) load-balancing algorithm available: `rr`, `wrr`, and `iphash`; respectively, round robin, weighted round robin, and IP hash.

```yaml
domains:                                       
  domain.com:                                                       
      routes:                                  
        /api/v1:                               
            dests: 
              - http://localhost:3001
              - http://172.44.38.89
            balancer: rr
        /metrics:
            dests:
              - http://localhost:8080
                weight: 3
              - http://172.44.38.89
                weight: 2
            balancer: wrr
  sub.domain.com:                                                       
      routes:                                  
        /api/v1:                               
            dests:
              - http://localhost:3001
              - http://172.44.38.89
            balancer: iphash
```

### TLS Certificates

The server automatically manages TLS certificates through Let's Encrypt using [certmagic](https://github.com/caddyserver/certmagic)

### Running the Server

There's a makefile provided which you can use to build the binary and run the server as a service.

`make build` will build the binary on the specified path.

`make copy_config` will copy the ***mrps.yaml*** and ***mrps.service*** to the specified path.

`make reload` will reload ***systemd***.

`make restart` will restart the ***mrps.service***.

`make start` will start the ***mrps.service***.

use `make install` to do everything.

You can configure the `mrps.service` as needed, but you need to keep the:
```
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE
```
as the reverse proxy server needs to bind on privileged ports, `80` and `443`.

### Metrics

The reverse proxy server exposes Prometheus-compatible metrics at the `metrics_port/metrics`.

#### Custom Metrics

1. `http_requests_total`
  - Type: Counter
  - Description: Total number of HTTP requests processed by the server.
  - Labels:
    - method: HTTP method (GET, POST, etc.)
    - endpoint: The URL path of the request.
2. `http_request_duration_seconds`
  - Type: Histogram
  - Description: Measures the duration of HTTP requests in seconds.
  - Labels:
    - method: HTTP method (GET, POST, etc.)
    - endpoint: The URL path of the request.
3. `http_active_requests`
  - Type: Gauge
  - Description: Number of currently active HTTP requests being processed by the server.

#### Scraping Metrics
Prometheus can scrape these metrics by configuring the  `metrics_port/metrics` endpoint as a target. Example scrape configuration in Prometheus:

```yaml
scrape_configs:
  - job_name: 'reverse_proxy'
    scheme: 'http'
    static_configs:
      - targets: ['localhost:7070']
```

to change the port, define it on `mrps.yaml`:

```yaml
misc:
  metrics_port:8080
```
