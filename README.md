# Reverse Proxy Server

This project implements a reverse proxy server that routes requests to different services based on the URL prefix. The server listens for requests and proxies them to the appropriate target service defined in the configuration.

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) installed on your machine.

### Running the Services

To run the reverse proxy server and services locally, you can use the provided batch script (`run.bat`) for Windows or shell script (`run.sh`) for Unix-like systems.

#### Windows (`run.bat`)

1. Open a command prompt (`cmd`) and navigate to the project directory.
2. Run the batch script:

```bash
./run.bat
```

This will start two services and the reverse proxy server.

#### Unix-like systems (`run.sh`)

1. Open a terminal and navigate to the project directory.
2. Run the shell script:

```bash
chmod +x ./run.sh
./run.sh
```

This will start two services and the reverse proxy server.

### Services

1. Service: This service listens on port 3001 and responds with "Hello from service".
2. Service 1: This service listens on port 3002 and responds with "Hello from service-1".

These services are mainly for testing.

The reverse proxy server will route requests based on the URL prefix:
* `/service/api` will be routed to `service`
* `/service-1/api` will be routed to `service-1`

Check the `internal/config/config.go`.

## Testing the Reverse Proxy

You can run the tests included with the project to ensure everything is functioning as expected.

- Run the tests using Go's testing framework. Navigate to the root of the project and execute the following:

```bash
go test -v ./...
```

This command will execute the tests and provide a detailed log of the results. It will verify the correct routing of requests to the appropriate services and check that the proxy server handles requests properly.

- Alternatively, you can also test the reverse proxy manually by using a tool like `curl` or a browser. Below are the endpoints you can test:

   * **Service**:
     ```bash
     curl http://localhost:3000/service/api
     ```
     This should return:
     ```
     Hello from service
     ```

   * **Service 1**:
     ```bash
     curl http://localhost:3000/service-1/api
     ```
     This should return:
     ```
     Hello from service-1
     ```

   * **Unknown path**:
     ```bash
     curl http://localhost:3000/service/unknown
     ```
     This should return:
     ```
     404 not found
     ```

   * **Root path**:
     ```bash
     curl http://localhost:3000
     ``` 

     This should return:
     ```
     Hello, from reverse proxy server
     ```
