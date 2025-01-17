@echo off
echo Starting service...
start go run cmd/services/service/main.go
echo Service is running.

echo Starting service 1...
start go run cmd/services/service_1/main.go
echo Service 1 is running.

echo Both services are running.
