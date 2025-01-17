#!/bin/bash

echo "Starting service..."
go run cmd/services/service/main.go &
echo "Service 0 is running."

echo "Starting service 1..."
go cmd/services/service_1/main.go &
echo "Service 1 is running."

echo "Both services are running."
wait
