#!/bin/bash

echo "Running unit tests..."
go test ./... -v

echo ""
echo "Running integration tests..."
go test -run TestIntegration -v
