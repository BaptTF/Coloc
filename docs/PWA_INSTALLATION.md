# PWA Installation Guide

> Transform your web experience into a native mobile app with Progressive Web App (PWA) technology.

## 📱 What is a PWA?

A Progressive Web App (PWA) is a web application that can be installed on your device just like a native app. It offers:

- **Offline functionality** - Works without internet connection
- **App-like experience** - Full screen, no browser UI
- **Push notifications** - Get alerts for download completion
- **Fast loading** - Cached resources for instant startup
- **Share integration** - Share videos directly from YouTube/Twitch
- **Auto-updates** - Always gets the latest version

## 🚀 Installation Instructions

### Android (Chrome/Edge/Firefox)

#### Method 1: Install Banner (Easiest)
1. **Open the app** in Chrome or Edge browser
2. **Wait for the install banner** to appear at the bottom
3. **Tap "Install"** to add to your home screen
4. **Confirm installation** when prompted

#### Method 2: Manual Installation
1. **Open the app** in your browser
2. **Tap the menu button** (⋮) in the top-right corner
3. **Select "Install app"** or "Add to Home screen"
4. **Confirm installation** by tapping "Install"

#### Method 3: From Share Menu
1. **In any app** (YouTube, Twitch, browser)
2. **Tap the Share button** 📤
3. **Select "Coloc"** from the share sheet
4. **Tap "Install"** when prompted

### iOS (iPhone/iPad)

#### Installation Steps
1. **Open the app** in Safari browser
2. **Tap the Share button** 📤 at the bottom of the screen
3. **Scroll down** and tap **"Add to Home Screen"** 🏠
4. **Edit the name** if desired (default: "Coloc")
5. **Tap "Add"** in the top-right corner

#### Visual Guide
```
Safari Interface:
┌─────────────────────────────┐
│ ← Coloc Video Downloader ⋮ │
├─────────────────────────────┤
│                             │
│        [App Content]        │
│                             │
├─────────────────────────────┤
│ [🔍] [📤] [📑] [...]        │
└─────────────────────────────┘

Share Sheet:
┌─────────────────────────────┐
│        Share Menu           │
├─────────────────────────────┤
│ Messages                    │
│ Mail                        │
│ Notes                       │
│ ────────────────────────── │
│ 🏠 Add to Home Screen       │ ← Tap this!
│ Find on Page                │
│ ...                         │
└─────────────────────────────┘
```

### Desktop (Chrome/Edge)

#### Installation Steps
1. **Open the app** in Chrome or Edge browser
2. **Look for the install icon** (⬇) in the address bar
3. **Click the install icon**
4. **Click "Install"** in the confirmation dialog
5. **Launch from desktop** or Start Menu

## 🎯 Web Share Target Feature

### What is Share Target?

The Share Target feature allows you to share video URLs directly from other apps to Coloc:

```
YouTube App → Share → Coloc → Auto-fill URL → Ready to Download
```

### How to Use

#### From YouTube App
1. **Open YouTube** and find a video
2. **Tap the Share button** 📤 below the video
3. **Select "Coloc"** from the share menu
4. **Coloc opens automatically** with the URL pre-filled
5. **Choose download mode** and tap download

#### From Twitch App
1. **Open Twitch** and find a video
2. **Tap the Share button** 
3. **Select "Coloc"** from the share menu
4. **URL is automatically detected** as Twitch content
5. **Start streaming** with one tap

#### From Any Browser
1. **Navigate to any video URL**
2. **Tap the browser's Share button** 📤
3. **Select "Coloc"** from the share menu
4. **URL is automatically filled** in the download form

### Supported Platforms

| Platform | Share Support | Notes |
|----------|---------------|-------|
| YouTube | ✅ Full | Auto-detects YouTube URLs |
| Twitch | ✅ Full | Auto-detects Twitch URLs |
| Direct URLs | ✅ Full | Works with any video link |
| iOS Safari | 🔄 Limited | Use copy-paste method |
| Android | ✅ Full | Native share integration |

## 🔧 PWA Features

### Offline Functionality
- **Browse downloaded videos** without internet
- **View download history** and queue status
- **Access settings** and configuration
- **Queue downloads** (processed when online)

### Push Notifications
- **Download completion alerts**
- **Error notifications**
- **VLC connection status updates**
- **Queue status changes**

### Background Sync
- **Retry failed downloads** automatically
- **Sync queue status** across devices
- **Update video list** when online

### App Shortcuts
- **Quick access** to YouTube downloads
- **Direct Twitch streaming** shortcut
- **Recent videos** quick access

## 🛠️ Troubleshooting

### Install Issues

#### "Install" button not showing
- **Ensure you're using a supported browser** (Chrome, Edge, Firefox)
- **Check internet connection**
- **Clear browser cache** and reload
- **Try manual installation** method

#### iOS installation not working
- **Use Safari browser** (other browsers don't support PWA installation)
- **Ensure iOS 13.4+** for full PWA support
- **Check storage space** on device
- **Restart Safari** and try again

#### App not appearing on home screen
- **Check all home screen pages**
- **Look in App Library** (iOS 14+)
- **Search for "Coloc"** on device
- **Restart device** if needed

### Share Target Issues

#### Coloc not appearing in share menu
- **Install the PWA first** (required for share target)
- **Check app permissions** in device settings
- **Restart the source app** (YouTube, Twitch)
- **Clear app cache** if needed

#### Shared URL not working
- **Ensure URL is valid** and accessible
- **Check internet connection**
- **Try copy-paste method** as fallback
- **Report the issue** with URL details

### Performance Issues

#### App loading slowly
- **Check internet connection** for first load
- **Clear browser cache** if corrupted
- **Close other apps** to free memory
- **Restart device** if needed

#### Downloads not working
- **Check server connection** status
- **Verify VLC configuration**
- **Ensure sufficient storage space**
- **Check internet connectivity**

## 📋 Device Compatibility

### Mobile Devices
| Device | OS Version | Browser | Install | Share Target |
|--------|------------|---------|---------|--------------|
| Android | 8.0+ | Chrome/Edge | ✅ | ✅ |
| Android | 8.0+ | Firefox | ✅ | 🔄 |
| iPhone | 13.4+ | Safari | ✅ | 🔄 |
| iPad | 13.4+ | Safari | ✅ | 🔄 |

### Desktop
| Platform | Browser | Install | Notes |
|----------|---------|---------|-------|
| Windows | Chrome | ✅ | Full support |
| Windows | Edge | ✅ | Full support |
| macOS | Chrome | ✅ | Full support |
| macOS | Safari | 🔄 | Limited support |
| Linux | Chrome | ✅ | Full support |

## 🔄 Updates

### Automatic Updates
- **PWA updates automatically** in the background
- **New version installs** on app restart
- **No manual intervention** required

### Manual Updates
1. **Open the app** with internet connection
2. **Wait for update** to download (automatic)
3. **Close and reopen** the app
4. **New version** will be active

### Update Notifications
- **Toast notifications** for available updates
- **Changelog** shown on major updates
- **Option to restart** immediately

## 🎨 Customization

### App Settings
- **Theme selection** (light/dark)
- **Download preferences**
- **VLC configuration**
- **Notification settings**

### Home Screen
- **Custom app name** (edit during install)
- **App icon** (automatically optimized)
- **Position on home screen** (user controlled)

## 📞 Support

### Getting Help
- **Check this guide** for common issues
- **Visit the GitHub repository** for known issues
- **Report bugs** with device and browser details
- **Request features** via GitHub issues

### Bug Reports
When reporting issues, include:
- **Device model** and OS version
- **Browser version**
- **Steps to reproduce**
- **Expected vs actual behavior**
- **Screenshots** if applicable

---

**Enjoy your native Coloc experience!** 🎬📱

*Last updated: November 2025*