# Reverse Proxy Server

This project implements an HTTP/HTTPS reverse proxy server that routes requests to different services based on the URL. The server listens for requests and proxies them to the appropriate target service defined in the configuration. It includes automatic TLS certificate generation and renewal through Let's Encrypt for secure HTTPS connections.

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
- Domain names pointed to your server's IP address (for SSL certificates)
- Port 80 and 443 available for HTTP/HTTPS traffic

### Configuration

Configurations are stored on the root's `config.yaml`.

#### Email

Used for Let's Encrypt configuration 

```yaml
email: "your@email.com"
```

#### Routes Configuration

Routes define how incoming requests are routed to different services. Example:

```yaml
domains:
  "domain.com":
    routes:
      "/api":
        dest: "http://localhost:3000"
      "/":
        dest: "http://localhost:9090"
    rate_limit:
      burst: 100
      rate: 50
      cooldown: 60000
  "sub.domain.com":
    routes:
      "/api":
        dest: "http://localhost:3000"
      "/":
        dest: "http://localhost:9090"    
    rate_limit:
      burst: 10
      rate: 5
      cooldown: 60000
```

#### Rate Limiting Configuration

Rate limiting defines how many request a client can do in a specified timeframe, applicable to domain and global scope. Example:

```yaml
rate_limit:
  burst: 50
  rate: 10
  cooldown: 60000
```

`burst` Maximum requests allowed in a short period. This enables handling sudden traffic spikes.

`rate` Requests per second at which tokens are replenished.

`cooldown` Time (in milliseconds) a client must wait after exhausting the burst limit.

#### How it Works

- `burst`: Client can make up to burst requests (50 in the example) without hitting the limit.
- `rate`: After the burst, requests are limited to rate tokens per second (10 in this example).
- `cooldown`: After exceeding the burst, the client must wait for the cooldown period (60 seconds) before making more requests.

### Routing Rules

The routing rules are simple and configurable. Example:

```yaml
domains:                                       # <- All domains are configured here
  "domain.com":                                # <- Domain name
      routes:                                 
        "/api":
            dest: "http://localhost:3000"      # <- path : dest
        "/metrics":
            dest: "http://localhost:9090"      # <- path : dest
  "sub.domain.com":                                
      routes:                                 
        "/api":
            dest: "http://localhost:3001"      # <- path : dest
        "/metrics":
            dest: "http://localhost:8080"   
```

You can also use wild cards:

```yaml
domains:                                       
  "*.domain.com":                              # <- Any sub-domain will be routed here                              
      routes:                                  # will ignore the base domain
        "/api":                                # unless configured below    
            dest: "http://localhost:3001"      # <- path : dest
        "/metrics":
            dest: "http://localhost:8080"  
  "domain.com":                                # <- Base domain     
      routes:                                 
        "/api":                                # unless configured below    
            dest: "http://localhost:3001"      # <- path : dest
        "/metrics":
            dest: "http://localhost:8080" 
```

#### Path Rewrites

There's two types of rewrites available, regex and prefix. Example

```yaml
domains:                                       
  "domain.com":                                                       
      routes:                                  
        "/api/v1":                               
            dest: "http://localhost:3001"
            rewrite:
              type: "regex"
              value: "^/api/v1/(.*)$"          # <- will be rewritten to /
              replace_val: "/$1"
        "/metrics":
            dest: "http://localhost:8080"  
  "sub.domain.com":                                                       
      routes:                                  
        "/api/v1":                               
            dest: "http://localhost:3001"
            rewrite:
              type: "regex"
              value: "^/api/v1/(.*)$"          # <- will be rewritten to /api/v2
              replace_val: "/api/v2/$1"
        "/metrics":
            dest: "http://localhost:8080"  
  "sub.domain.com":                                                       
      routes:                                  
        "/api/v1":                               
            dest: "http://localhost:3001"
            rewrite:
              type: "prefix"
              value: "/api/v1"                 # <- will be rewritten to /
              replace_val: ""
        "/metrics":
            dest: "http://localhost:8080"  
  "sub.domain.com":                                                       
      routes:                                  
        "/api/v1":                               
            dest: "http://localhost:3001"
            rewrite:
              type: "prefix"
              value: "/api/v1"                 # <- will be rewritten to /new/path
              replace_val: "/new/path"
        "/metrics":
            dest: "http://localhost:8080"  
```

### TLS Certificates

The server automatically manages TLS certificates through Let's Encrypt using [certmagic](https://github.com/caddyserver/certmagic):
- Certificates are obtained when the server starts
- Automatic renewal before expiration
- Certificates are cached locally for reuse

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

The reverse proxy server exposes Prometheus-compatible metrics at the /metrics endpoint to help monitor the server's performance.

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
Prometheus can scrape these metrics by configuring the server's /metrics endpoint as a target. Example scrape configuration in Prometheus:
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
