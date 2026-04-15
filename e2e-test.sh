#!/bin/bash

echo "Building the application..."
go build -o llm-gateway

echo "Starting the server..."
./llm-gateway &
SERVER_PID=$!

echo "Waiting for server to start..."
sleep 3

echo "Running integration tests..."
go test -run TestIntegration -v

echo "Stopping server..."
kill $SERVER_PID

echo "Done!"
