![Go Magic](https://github.com/MariaLetta/free-gophers-pack/blob/master/goroutines/png/14.png)

Image is from [MariaLetta/free-gophers-pack/goroutines](https://github.com/MariaLetta/free-gophers-pack/tree/master/goroutines)

# MRPS

`mrps` is a HTTP->HTTPS reverse proxy server with TCP proxy capabilities. Built with `Let's Encrypt` using [certmagic](https://github.com/caddyserver/certmagic).

## Features

- Dynamic routing for HTTP/HTTPS and TCP traffic
- Automatic HTTPS with Let's Encrypt
- TCP proxy with optional TLS termination
- Configurable routing rules
- Path rewrites (HTTP only)
- Global and domain-based rate limiting
- Load balancing algorithms
- Scrappable metrics
- Health checks

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) installed on your machine
- Port 80 and 443 available for HTTP/HTTPS traffic
- Additional ports as needed for TCP proxying

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

#### Miscellaneous

```yaml
misc:
  email: your@mail.com           # Used for certmagic (optional)
  allow_http: true               # Allow traffic on port 80
  secure: true                   # Enable HTTPS
  health_check_interval: 1000    # Health check interval in ms
  enable_metrics: true
  metrics_port: "7070"           # Default 7070
  enable_api: true
  api_port: "6060"               # Default 6060
  domain: your_domain.com        # Main domain for the service
  allowed_origins:               # Used for WS connections
  - https://your_domain.com
  - http://localhost:5050
```

#### Route Configuration

Routes define how incoming requests are routed to different services. MRPS now supports both HTTP and TCP protocols.

##### HTTP Routes

Basic HTTP route:

```yaml
domains:
  domain.com:
    enabled: true
    protocol: http
    routes:
      /api:
        dests:
        - url: http://localhost:3000
      /:
        dests:
        - url: http://localhost:9090
```

##### TCP Routes

Note: TCP currently uses the same configuration as HTTP, hence the `routes`, TCP will only forward request to `/`. Will fix it in the future.

TCP routes proxy raw TCP connections, with optional TLS termination:

```yaml
domains:
  tcp.domain.com:
    enabled: true
    protocol: tcp
    routes:
      /:
        dests:
        - url: localhost:8888          # Plain TCP backend
  secure-tcp.domain.com:
    enabled: true
    protocol: tcp
    routes:
      /:
        dests:
        - url: localhost:8890
          with_tls: true               # TLS-enabled backend
```

**Important Notes for TCP Routes:**
- TCP destinations should not include `http://` or `https://` prefixes
- Use `with_tls: true` when the backend service expects TLS connections
- Path rewrites are not supported for TCP routes
- Wildcard domains are supported (e.g., `'*.tcp.domain.com'`)

#### Rate Limiting Configuration

Rate limiting defines how many requests a client can make in a specified timeframe, applicable to both HTTP and TCP connections at domain and global scope.

##### Global

```yaml
rate_limit:
  burst: 50
  rate: 10
  cooldown: 60000
```

##### Domain-specific

```yaml
domains:
  domain.com:
    enabled: true
    protocol: http
    routes:
      /:
        dests:
        - url: http://localhost:6060
    rate_limit:
      burst: 50
      rate: 10
      cooldown: 60000
```

**Rate Limit Parameters:**
- `burst`: Maximum requests allowed in a short period
- `rate`: Requests per second at which tokens are replenished
- `cooldown`: Time (in milliseconds) a client must wait after exhausting the burst limit

#### Path Rewrites (HTTP Only)

Path rewrites are available for HTTP routes only. There are two types: `regex` and `prefix`.

```yaml
domains:                                       
  domain.com:
    enabled: true
    protocol: http                                                      
    routes:                                  
      /api/v1:                               
        dests:
        - url: http://localhost:3001
        rewrite:
          type: regex
          value: ^/api/v1/(.*)$          # Rewritten to /
          replace_val: /$1
      /metrics:
        dests:
        - url: http://localhost:8080  
  sub.domain.com:
    enabled: true
    protocol: http                                                      
    routes:                                  
      /api/v1:                               
        dests:
        - url: http://localhost:3001
        rewrite:
          type: prefix
          value: /api/v1                 # Rewritten to /new/path
          replace_val: /new/path
      /metrics:
        dests:
        - url: http://localhost:8080
```

### Load Balancing

Three load-balancing algorithms are available: `rr` (round robin), `wrr` (weighted round robin), and `iphash` (IP hash). These work for both HTTP and TCP routes.

```yaml
domains:                                       
  domain.com:
    enabled: true
    protocol: http                                                      
    routes:                                  
      /api/v1:                               
        dests: 
        - url: http://localhost:3001
        - url: http://172.44.38.89
        balancer: rr
      /metrics:
        dests:
        - url: http://localhost:8080
          weight: 3
        - url: http://172.44.38.89
          weight: 2
        balancer: wrr
  tcp.domain.com:
    enabled: true
    protocol: tcp                                                      
    routes:                                  
      /:                               
        dests:
        - url: localhost:3001
        - url: 172.44.38.89:3001
        balancer: iphash
```

### Protocol Configuration

Each domain must specify a protocol type:

- `protocol: http` - For HTTP/HTTPS traffic with full reverse proxy features
- `protocol: tcp` - For raw TCP proxying with optional TLS termination

### TLS Certificates

The server automatically manages TLS certificates through Let's Encrypt using [certmagic](https://github.com/caddyserver/certmagic). This applies to both HTTP and TCP protocols when TLS termination is required.

### Running the Server

There's a makefile provided which you can use to build the binary and run the server as a service.

- `make build` - Build the binary on the specified path
- `make copy_config` - Copy the ***mrps.yaml*** and ***mrps.service*** to the specified path
- `make reload` - Reload ***systemd***
- `make restart` - Restart the ***mrps.service***
- `make start` - Start the ***mrps.service***
- `make install` - Do everything

You can configure the `mrps.service` as needed, but you need to keep the:
```
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE
```
as the reverse proxy server needs to bind on privileged ports, `80` and `443`.

### Metrics

The reverse proxy server exposes Prometheus-compatible metrics at the `metrics_port/metrics` endpoint.

#### Custom Metrics

1. `http_requests_total`
   - Type: Counter
   - Description: Total number of HTTP requests processed by the server
   - Labels:
     - method: HTTP method (GET, POST, etc.)
     - endpoint: The URL path of the request

2. `http_request_duration_seconds`
   - Type: Histogram
   - Description: Measures the duration of HTTP requests in seconds
   - Labels:
     - method: HTTP method (GET, POST, etc.)
     - endpoint: The URL path of the request

3. `http_active_requests`
   - Type: Gauge
   - Description: Number of currently active HTTP requests being processed by the server

#### Scraping Metrics

Prometheus can scrape these metrics by configuring the `metrics_port/metrics` endpoint as a target. Example scrape configuration in Prometheus:

```yaml
scrape_configs:
  - job_name: 'mrps'
    scheme: 'http'
    static_configs:
      - targets: ['localhost:7070']
```

To change the port, define it in `config.yaml`:

```yaml
misc:
  metrics_port: "8080"
```

## Example Configuration

Here's a complete example showing both HTTP and TCP configurations:

```yaml
domains:
  # HTTP service
  api.example.com:
    enabled: true
    protocol: http
    routes:
      /v1:
        dests:
        - url: http://localhost:3000
        rewrite:
          type: prefix
          value: /v1
          replace_val: /
      /:
        dests:
        - url: http://localhost:8080
    rate_limit:
      burst: 50
      rate: 10
      cooldown: 60000

  # TCP service with TLS backend
  secure-tcp.example.com:
    enabled: true
    protocol: tcp
    routes:
      /:
        dests:
        - url: localhost:5432
          with_tls: true
    rate_limit:
      burst: 20
      rate: 5
      cooldown: 60000

  # Plain TCP service
  tcp.example.com:
    enabled: true
    protocol: tcp
    routes:
      /:
        dests:
        - url: localhost:6379
    rate_limit:
      burst: 100
      rate: 50
      cooldown: 30000

misc:
  email: admin@example.com
  secure: true
  allow_http: false
  health_check_interval: 1000
  enable_metrics: true
  metrics_port: "7070"
  enable_api: true
  api_port: "6060"
  domain: mrps.example.com
  allowed_origins:
  - https://mrps.example.com

rate_limit:
  burst: 100
  rate: 50
  cooldown: 60000
```
