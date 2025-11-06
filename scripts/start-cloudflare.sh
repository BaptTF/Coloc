#!/bin/bash
# Coloc + Cloudflare Tunnel Startup Script

echo "🚀 Starting Coloc Video Downloader with Cloudflare Tunnel..."

# Start Coloc server
echo "📦 Starting Coloc server..."
docker compose up -d

# Wait for server to be ready
echo "⏳ Waiting for server to start..."
sleep 5

# Check if server is running
if curl -s http://localhost:8080 > /dev/null; then
    echo "✅ Coloc server is running on port 8080"
else
    echo "❌ Coloc server failed to start"
    exit 1
fi

# Start Cloudflare Tunnel
echo "🌐 Starting Cloudflare Tunnel..."
echo "📱 Your PWA will be available at the HTTPS URL shown below"
echo "🔗 Use this URL for mobile PWA installation"
echo ""

cloudflared tunnel --url http://localhost:8080