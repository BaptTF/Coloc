# Cloudflare Tunnel Setup for Coloc

Cloudflare Tunnel provides free, unlimited HTTPS tunnels without requiring authentication or registration.

## Quick Start

### 1. Install cloudflared

**NixOS:**
```bash
nix-shell -p cloudflared --run "cloudflared tunnel --url http://localhost:8080"
```

**Or install permanently:**
```bash
# Add to configuration.nix
environment.systemPackages = [ pkgs.cloudflared ];
```

**Other systems:**
```bash
# macOS
brew install cloudflared

# Linux
wget -q https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared-linux-amd64.deb
```

### 2. Start the Tunnel

```bash
# Start Coloc server
docker compose up -d

# Start Cloudflare Tunnel
cloudflared tunnel --url http://localhost:8080
```

### 3. Access Your App

Cloudflare will provide a HTTPS URL like:
```
https://random-words-trycloudflare.com
```

This URL works perfectly for PWA installation with trusted certificates.

## Configuration Options

### Basic Tunnel
```bash
cloudflared tunnel --url http://localhost:8080
```

### Custom Subdomain (requires Cloudflare account)
```bash
# Login to Cloudflare
cloudflared tunnel login

# Create tunnel
cloudflared tunnel create my-coloc-app

# Configure with custom domain
cloudflared tunnel route dns my-coloc-app myapp.example.com
```

### Advanced Configuration
Create `~/.cloudflared/config.yml`:
```yaml
tunnel: my-coloc-app
credentials-file: ~/.cloudflared/my-coloc-app.json

ingress:
  - hostname: myapp.example.com
    service: http://localhost:8080
  - service: http_status:404
```

## PWA Installation

1. **Start services:**
   ```bash
   docker compose up -d
   cloudflared tunnel --url http://localhost:8080
   ```

2. **Open on mobile device:**
   - Navigate to the `trycloudflare.com` URL
   - The PWA install prompt should appear automatically
   - Or use browser menu: "Add to Home Screen"

3. **Verify installation:**
   - Check app appears on home screen
   - Test offline functionality
   - Confirm video downloads work

## Troubleshooting

### Tunnel Not Starting
```bash
# Check if port 8080 is available
netstat -tlnp | grep :8080

# Verify Coloc server is running
curl http://localhost:8080
```

### PWA Not Installing
- Ensure HTTPS URL (Cloudflare provides this automatically)
- Check Service Worker is registered: DevTools > Application > Service Workers
- Verify manifest.json is accessible
- Try refreshing the page and waiting 30 seconds

### Connection Issues
```bash
# Check tunnel status
cloudflared tunnel info my-coloc-app

# View logs
cloudflared tunnel --url http://localhost:8080 --loglevel debug
```

## Comparison: Cloudflare vs ngrok

| Feature | Cloudflare Tunnel | ngrok |
|---------|-------------------|-------|
| Cost | Free unlimited | Free tier limited |
| Registration | Not required | Required |
| URL | `trycloudflare.com` | `ngrok.io`/`ngrok-free.app` |
| Certificate | Cloudflare trusted | ngrok trusted |
| Speed | Very fast | Fast |
| Custom domains | Paid account | Paid plan |
| Advanced features | Limited | Rich feature set |

## Automation

### Script for Easy Startup
```bash
#!/bin/bash
# start-cloudflare.sh

echo "Starting Coloc server..."
docker compose up -d

echo "Starting Cloudflare Tunnel..."
cloudflared tunnel --url http://localhost:8080
```

### Systemd Service (optional)
Create `/etc/systemd/system/coloc-tunnel.service`:
```ini
[Unit]
Description=Coloc Cloudflare Tunnel
After=docker.service
Requires=docker.service

[Service]
Type=simple
User=your-username
ExecStart=/usr/bin/cloudflared tunnel --url http://localhost:8080
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now coloc-tunnel.service
```

## Security Notes

- Cloudflare tunnels are secure by default
- Traffic is encrypted between device and Cloudflare
- No open ports on your local machine
- Perfect for development and testing