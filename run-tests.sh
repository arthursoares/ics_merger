#!/bin/bash
# Script to run all tests including Ruby validation in Docker

# Ensure script exits if any command fails
set -e

echo "Building test container..."
docker build -t ical_merger_test -f Dockerfile.test .

echo "Running tests..."
docker run --rm ical_merger_test

echo "Tests completed successfully!"