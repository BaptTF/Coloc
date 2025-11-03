# iOS PWA Installation Guide

> Install Coloc on your iPhone or iPad for a native app experience.

## 📱 System Requirements

- **iOS 13.4 or higher** (recommended: iOS 14+)
- **Safari browser** (required for PWA installation)
- **At least 50MB free storage space**
- **Internet connection** for initial installation

## 🚀 Installation Steps

### Step-by-Step Installation

1. **Open Safari browser** on your iPhone/iPad
2. **Navigate to** your Coloc server URL
3. **Tap the Share button** 📤 at the bottom of the screen
4. **Scroll down** in the share menu
5. **Tap "Add to Home Screen"** 🏠
6. **Edit the name** if desired (default: "Coloc")
7. **Tap "Add"** in the top-right corner

### Visual Guide

```
Step 1: Open in Safari
┌─────────────────────────────┐
│  ← Coloc Video Downloader   │
├─────────────────────────────┤
│                             │
│      [App Content]          │
│                             │
│                             │
├─────────────────────────────┤
│ [🔍] [📤] [📑] [...]        │ ← Tap Share button
└─────────────────────────────┘

Step 2: Share Menu
┌─────────────────────────────┐
│        Share Menu           │
├─────────────────────────────┤
│ Messages                    │
│ Mail                        │
│ Notes                       │
│ Reminders                   │
│ ────────────────────────── │
│ 🏠 Add to Home Screen       │ ← Scroll and tap this
│ Find on Page                │
│ Bookmark                    │
│ ...                         │
└─────────────────────────────┘

Step 3: Add to Home Screen
┌─────────────────────────────┐
│                             │
│    [📱 Coloc Icon]          │
│                             │
│         Coloc               │
│    Video Downloader         │
│                             │
│                         [Add] │ ← Tap to install
└─────────────────────────────┘
```

## 🎯 iOS Limitations and Workarounds

### Share Target Limitations
**iOS Safari has limited Web Share Target support.** Here are the workarounds:

#### Method 1: Copy-Paste (Recommended)
1. **In YouTube app:** Tap Share → Copy Link
2. **Open Coloc PWA** from home screen
3. **Tap and hold** the URL input field
4. **Select "Paste"**
5. **URL is auto-filled** and ready to download

#### Method 2: Safari Share
1. **Open video in Safari** (not YouTube app)
2. **Tap Share button** 📤
3. **Find "Coloc"** in share menu (if available)
4. **Tap to open** with URL pre-filled

#### Method 3: Shortcuts App (Advanced)
Create a Siri Shortcut to share URLs to Coloc:
1. **Open Shortcuts app**
2. **Create new shortcut**
3. **Add action:** "Get URL from Input"
4. **Add action:** "Open URL" with your Coloc URL + "?url=[URL]"
5. **Add to Share Sheet**

## ⚙️ iOS-Specific Features

### Native App Integration
- **Home screen icon** with proper badge
- **App Library** integration (iOS 14+)
- **Spotlight search** integration
- **Siri suggestions** (based on usage)
- **Widget support** (future update)

### iOS Optimizations
- **Safe area handling** for notched devices
- **Gesture navigation** support
- **Dark mode** integration
- **Dynamic Type** support
- **Haptic feedback** on interactions

### Performance Features
- **Fast app switching**
- **Background refresh** limited by iOS
- **Memory management** optimized
- **Battery usage** optimized

## 🔧 Configuration

### Safari Settings
1. **Open Settings** → Safari
2. **Enable "Block Pop-ups"** (recommended)
3. **Disable "Prevent Cross-Site Tracking"** if needed
4. **Clear History** if experiencing issues

### App Permissions
iOS PWAs have limited permissions:
- **Notifications:** Available with iOS 16.4+
- **Camera/Microphone:** Available when requested
- **Location:** Available when requested
- **Background sync:** Limited by iOS

### Home Screen Organization
- **Move icon:** Long-press → "Edit Home Screen"
- **Create folder:** Drag icon onto another app
- **Remove from home:** Long-press → "Remove App" → "Remove from Home Screen"
- **App Library access:** Swipe to last page

## 🛠️ Troubleshooting

### Installation Issues

#### "Add to Home Screen" not appearing
**Solutions:**
- **Use Safari only:** Other browsers don't support PWA installation
- **Check iOS version:** Must be iOS 13.4+
- **Update Safari:** Ensure latest iOS version
- **Reload page:** Pull to refresh and try again
- **Clear Safari data:** Settings → Safari → Clear History

#### Installation fails
**Solutions:**
- **Free up storage:** Need at least 50MB available
- **Restart device:** Clear temporary issues
- **Check internet connection:** Required for installation
- **Try different URL:** Ensure server is accessible

#### App icon not appearing
**Solutions:**
- **Check all home screens:** Swipe through pages
- **Check App Library:** Swipe to last page (iOS 14+)
- **Search for "Coloc":** Swipe down on home screen
- **Restart device:** Home screen may need refresh

### Share Issues

#### Copy-paste not working
**Solutions:**
- **Use YouTube app's copy feature**
- **Try Safari instead of YouTube app**
- **Check clipboard:** Open Notes app to verify
- **Restart both apps:** Force close and reopen

#### URL not auto-detecting
**Solutions:**
- **Ensure complete URL** copied
- **Check for extra characters** in clipboard
- **Try manual typing** as fallback
- **Report specific URL** if consistently failing

### Performance Issues

#### App loading slowly
**Solutions:**
- **Check internet connection:** First load needs connectivity
- **Clear Safari cache:** Settings → Safari → Clear History
- **Close background apps:** Double-click home button
- **Restart device:** Clear system memory

#### Downloads not working
**Solutions:**
- **Check server status:** Ensure backend is running
- **Verify Wi-Fi connection:** Downloads need stable connection
- **Try different video:** Test with known working URL
- **Check VLC settings:** Ensure VLC is accessible

## 📋 Device-Specific Tips

### iPhone Models

#### iPhone with Face ID
- **Swipe up from bottom** to go home
- **Swipe down from top-right** for Control Center
- **Use gesture navigation** for best experience

#### iPhone with Home Button
- **Press Home button** to go home
- **Swipe up from bottom** for Control Center
- **Double-click Home button** for app switcher

#### iPhone SE (3rd gen)
- **Touch ID support** for secure authentication
- **Compact size** optimized for one-handed use
- **Portrait orientation** recommended

### iPad Models

#### iPad Pro/ Air
- **Split-screen multitasking** supported
- **Slide Over** for quick access
- **Landscape orientation** optimized
- **Apple Pencil** compatible (for future features)

#### iPad mini
- **Portable size** for mobile use
- **Touch ID** for quick authentication
- **Portrait/landscape** both supported
- **Compact mode** available

#### iPad (9th gen)
- **A13 Bionic chip** for smooth performance
- **Center Stage** compatible (future feature)
- **Smart Keyboard** support
- **Apple Pencil** (1st gen) support

## 🔄 Updates and Maintenance

### Automatic Updates
- **Updates check** when app is opened
- **Background updates** limited by iOS
- **Manual refresh** sometimes needed

### Manual Update Process
1. **Open Coloc app**
2. **Connect to internet**
3. **Pull to refresh** if needed
4. **Close and reopen** app
5. **Update applies** automatically

### Storage Management
1. **Open Settings** → General → iPad/iPhone Storage
2. **Find Coloc** in app list
3. **Check storage usage**
4. **Offload app** if needed (preserves data)

## 🎨 Customization

### Home Screen Layout
- **Custom widgets** (iOS 14+)
- **App folders** for organization
- **Custom app icons** (iOS 14.3+)
- **Focus modes** integration

### Display Settings
- **Display & Brightness** → Light/Dark Mode
- **Display Zoom** for larger text
- **Bold Text** for better readability
- **Reduce Motion** if sensitive to animations

### Accessibility
- **VoiceOver** support
- **Zoom** magnification
- **Larger Text** dynamic type
- **Switch Control** compatibility

## 📊 iOS Limitations

### Known Limitations
- **No background sync** (iOS restriction)
- **Limited notifications** (iOS 16.4+ for full support)
- **No share target** in most apps
- **Storage limits** based on device space
- **Memory management** aggressive on older devices

### Workarounds
- **Manual refresh** for updates
- **Copy-paste method** for sharing
- **Frequent app opening** for background tasks
- **Wi-Fi only** for large downloads

## 🔒 Privacy and Security

### Safari Privacy
- **Intelligent Tracking Prevention** enabled
- **Privacy Report** available
- **Hide IP Address** option
- **Private Browsing** mode

### App Data
- **Stored locally** on device
- **Synced with iCloud** (if enabled)
- **Encrypted** with device passcode
- **Sandboxed** from other apps

---

**Need more help?** Check the main [PWA Installation Guide](PWA_INSTALLATION.md) or [Android Guide](PWA_ANDROID_GUIDE.md).

*Compatible with iOS 13.4+ • Safari required • Last updated: November 2025*