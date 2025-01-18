# Reverse Proxy Server

This project implements a reverse proxy server that routes requests to different services based on the URL prefix. The server listens for requests and proxies them to the appropriate target service defined in the configuration. It includes automatic TLS certificate generation and renewal through Let's Encrypt for secure HTTPS connections.

## Overview

The reverse proxy server operates using two main configuration components:
1. **Domains Configuration**: Defines the domain names that the proxy will handle
2. **Routes Configuration**: Specifies how incoming requests should be directed to backend services

## Features

- Dynamic request routing based on URL prefixes
- Automatic SSL/TLS certificate management via Let's Encrypt
- Support for multiple domains
- Configurable routing rules
- Zero-downtime certificate renewal

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) installed on your machine
- Domain names pointed to your server's IP address (for SSL certificates)
- Port 80 and 443 available for HTTP/HTTPS traffic

### Configuration

Configurations are stored on the root's `config.yaml`.

#### Domain Configuration

The domains configuration specifies which domains the proxy will handle. Example:

```yaml
domains:
  - domain.com
  - api.domain.com
  - sub.domain.com
```

#### Routes Configuration

Routes define how incoming requests are forwarded to backend services. Example:

```yaml
routes:
  "domain.com": "http://localhost:4000"
  "domain.com/api": "http://localhost:4001"
```

### SSL Certificates

The server automatically manages SSL certificates through Let's Encrypt:
- Certificates are obtained when the server starts
- Automatic renewal before expiration
- Certificates are cached locally for reuse

### Running the Server

1. Set up your configuration file
2. You have two options to run the server:

#### Option 1: Direct Run
```bash
go run main.go
```

#### Option 2: SystemD Deployment
The project includes a build-deploy script that automates the deployment process and sets up the server as a SystemD service:

1. Run the deployment script:
```bash
./build-deploy.sh
```

This script will:
- Build the Go binary
- Create a SystemD service file
- Install and enable the service
- Start the server

To check the service status:
```bash
systemctl status reverse-proxy
```

To view logs:
```bash
journalctl -u reverse-proxy -f
```
