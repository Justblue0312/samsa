#!/bin/bash

# generate_mocks.sh - Generate all mocks for the Samsa project
# 
# Usage: ./scripts/generate_mocks.sh
#
# Prerequisites:
#   go install github.com/golang/mock/mockgen@latest

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="$(dirname "$SCRIPT_DIR")"

echo "Generating mocks for Samsa project..."
echo "Server directory: $SERVER_DIR"
echo ""

# Check if mockgen is installed
if ! command -v mockgen &> /dev/null; then
    echo "Error: mockgen is not installed"
    echo "Run: go install github.com/golang/mock/mockgen@latest"
    exit 1
fi

cd "$SERVER_DIR"

# Comment mocks
echo "Generating comment mocks..."
mockgen -source=internal/feature/comment/repository.go \
    -destination=internal/feature/comment/mocks/mock_repository.go \
    -package=mocks

mockgen -source=internal/feature/comment/usecase.go \
    -destination=internal/feature/comment/mocks/mock_usecase.go \
    -package=mocks

mockgen -source=internal/feature/comment/http_handler.go \
    -destination=internal/feature/comment/mocks/mock_http_handler.go \
    -package=mocks

# File mocks
echo "Generating file mocks..."
mockgen -source=internal/feature/file/repository.go \
    -destination=internal/feature/file/mocks/repository_mock.go \
    -package=mocks

mockgen -source=internal/feature/file/usecase.go \
    -destination=internal/feature/file/mocks/usecase_mock.go \
    -package=mocks

# Submission mocks
echo "Generating submission mocks..."
mockgen -source=internal/feature/submission/repository.go \
    -destination=internal/feature/submission/mocks/mock_repository.go \
    -package=mocks

mockgen -source=internal/feature/submission/usecase.go \
    -destination=internal/feature/submission/mocks/mock_usecase.go \
    -package=mocks

# Story post mocks
echo "Generating story post mocks..."
mockgen -source=internal/feature/story_post/repository.go \
    -destination=internal/feature/story_post/mocks/mock_repository.go \
    -package=mocks

mockgen -source=internal/feature/story_post/usecase.go \
    -destination=internal/feature/story_post/mocks/mock_usecase.go \
    -package=mocks

mockgen -source=internal/feature/story_post/http_handler.go \
    -destination=internal/feature/story_post/mocks/mock_http_handler.go \
    -package=mocks

# Story vote mocks (if not already done)
echo "Generating story vote mocks..."
mockgen -source=internal/feature/story_vote/repository.go \
    -destination=internal/feature/story_vote/mocks/mock_repository.go \
    -package=mocks

mockgen -source=internal/feature/story_vote/usecase.go \
    -destination=internal/feature/story_vote/mocks/mock_usecase.go \
    -package=mocks

mockgen -source=internal/feature/story_vote/http_handler.go \
    -destination=internal/feature/story_vote/mocks/mock_http_handler.go \
    -package=mocks

# Author mocks
echo "Generating author mocks..."
mockgen -source=internal/feature/author/repository.go \
    -destination=internal/feature/author/mocks/mock_repository.go \
    -package=mocks

mockgen -source=internal/feature/author/usecase.go \
    -destination=internal/feature/author/mocks/mock_usecase.go \
    -package=mocks

# User mocks
echo "Generating user mocks..."
mockgen -source=internal/feature/user/repository.go \
    -destination=internal/feature/user/mocks/mock_repository.go \
    -package=mocks

mockgen -source=internal/feature/user/usecase.go \
    -destination=internal/feature/user/mocks/mock_usecase.go \
    -package=mocks

mockgen -source=internal/feature/user/http_handler.go \
    -destination=internal/feature/user/mocks/mock_http_handler.go \
    -package=mocks

# Auth mocks
echo "Generating auth mocks..."
mockgen -source=internal/feature/auth/usecase.go \
    -destination=internal/feature/auth/mocks/mock_usecase.go \
    -package=mocks

echo ""
echo "✓ All mocks generated successfully!"
echo ""
echo "Generated files:"
find internal/feature -name "mock_*.go" -o -name "*_mock.go" | sort
