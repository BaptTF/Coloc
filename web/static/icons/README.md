# PWA Icons

This directory contains Progressive Web App icons for different platforms and sizes.

## Required Icons

For a complete PWA implementation, you'll need the following icons:

### App Icons
- `icon-72x72.png` - 72×72px
- `icon-96x96.png` - 96×96px  
- `icon-128x128.png` - 128×128px
- `icon-144x144.png` - 144×144px
- `icon-152x152.png` - 152×152px
- `icon-192x192.png` - 192×192px
- `icon-384x384.png` - 384×384px
- `icon-512x512.png` - 512×512px

### Platform-Specific Icons
- `apple-touch-icon.png` - 180×180px (iOS)
- `favicon-16x16.png` - 16×16px (Browser)
- `favicon-32x32.png` - 32×32px (Browser)
- `safari-pinned-tab.svg` - SVG (Safari pinned tab)

### Shortcut Icons
- `youtube-96x96.png` - 96×96px (YouTube shortcut)
- `twitch-96x96.png` - 96×96px (Twitch shortcut)

## Design Guidelines

### Color Scheme
- **Primary:** #f97316 (Orange)
- **Secondary:** #0f172a (Dark blue)
- **Background:** White or transparent

### Icon Design
- **Simple, recognizable** video/download theme
- **High contrast** for visibility
- **No text** - icon should be self-explanatory
- **Consistent style** across all sizes

### Tools for Creation
- **Figma** - Professional design tool
- **Canva** - Easy online tool
- **GIMP** - Free Photoshop alternative
- **Online generators** - PWA icon generators

## Temporary Solution

For development and testing, you can:
1. Use online PWA icon generators
2. Create simple geometric shapes
3. Use emoji-based designs (temporary)
4. Download free icon sets

## Implementation

Once icons are created, ensure they're referenced in:
- `manifest.json` - PWA manifest
- `index.html` - Meta tags and links
- Service worker - Cache the icons

## Testing

Test icons on:
- **Android Chrome** - Install banner and home screen
- **iOS Safari** - Add to Home Screen
- **Desktop browsers** - Favicon and PWA install
- **Various screen densities** - 1x, 2x, 3x displays

---

**Note:** Replace this README with actual icon files for production deployment.