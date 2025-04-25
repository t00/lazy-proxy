#!/bin/bash

set -e  # Exit on error

IMAGE_NAME=lazy-proxy-builder
CONTAINER_NAME=lazy-proxy-temp
OUTPUT_PATH=./lazy-proxy

echo "🔨 Building Docker image..."
docker build -t $IMAGE_NAME .

echo "📦 Creating temporary container..."
docker create --name $CONTAINER_NAME $IMAGE_NAME

echo "📤 Copying binary to host ($OUTPUT_PATH)..."
docker cp $CONTAINER_NAME:/app/lazy-proxy $OUTPUT_PATH

echo "🧹 Cleaning up container and image..."
docker rm $CONTAINER_NAME > /dev/null
docker rmi $IMAGE_NAME > /dev/null

echo "✅ Done! Binary is at: $OUTPUT_PATH"
