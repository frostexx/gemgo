#!/bin/bash
set -e

echo "Building frontend..."
npm install
npm run build

echo "Building backend..."
go build -o gemgo .

echo "Build complete!"