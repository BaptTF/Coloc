# Documentation Index

Welcome to the Coloc Video Downloader documentation!

## 📚 Documentation Structure

### [ARCHITECTURE.md](ARCHITECTURE.md) - System Design & Technical Details
**1,085 lines** | Comprehensive technical documentation

**What's inside:**
- System overview and high-level architecture
- Detailed backend architecture (Go packages, data flow)
- Frontend architecture (JavaScript modules, state management)
- VLC integration protocol and authentication flow
- Complete data flow diagrams
- Technology stack and performance considerations

**Read this if you want to:**
- Understand how the system works internally
- Learn about the backend Go packages and their responsibilities
- Understand the frontend JavaScript module structure
- Learn how VLC authentication and playback control works
- See the complete data flow between components

---

### [API.md](API.md) - Complete API Reference
**840 lines** | HTTP and WebSocket API documentation

**What's inside:**
- All HTTP endpoints with request/response examples
- WebSocket protocol and message types
- Data models and type definitions
- Error handling and status codes
- Rate limiting and CORS considerations

**Read this if you want to:**
- Integrate with the API programmatically
- Understand the HTTP endpoints and their parameters
- Learn the WebSocket message protocol
- See all data structures and their fields
- Handle errors properly in your client code

---

### [DEVELOPMENT.md](DEVELOPMENT.md) - Setup & Contributing
**785 lines** | Development setup and contribution guidelines

**What's inside:**
- Local development setup instructions
- Project structure explanation
- Building and testing procedures
- Code style guidelines (Go, JavaScript, CSS)
- Contributing workflow and PR guidelines
- Troubleshooting common issues

**Read this if you want to:**
- Set up a local development environment
- Contribute code to the project
- Run tests and understand the test structure
- Follow code style conventions
- Debug common development issues

---

## 🎯 Quick Navigation

### For Users
Start with the main [README.md](../README.md) in the root directory for:
- Project goals and features
- Quick start guide
- VLC setup instructions
- Basic usage

### For Developers
1. **First time?** Read [DEVELOPMENT.md](DEVELOPMENT.md) for setup
2. **Understanding the code?** Read [ARCHITECTURE.md](ARCHITECTURE.md)
3. **Building an integration?** Read [API.md](API.md)

### For Contributors
1. [DEVELOPMENT.md](DEVELOPMENT.md) - Setup and guidelines
2. [ARCHITECTURE.md](ARCHITECTURE.md) - Understand the codebase
3. Submit your PR following the contribution guidelines

---

## 📖 Documentation Statistics

| Document | Lines | Purpose |
|----------|-------|---------|
| [README.md](../README.md) | 187 | Project overview and quick start |
| [ARCHITECTURE.md](ARCHITECTURE.md) | 1,085 | Technical architecture and design |
| [API.md](API.md) | 840 | API reference and protocols |
| [DEVELOPMENT.md](DEVELOPMENT.md) | 785 | Development and contribution guide |
| **Total** | **2,897** | Complete documentation |

---

## 🔍 Find What You Need

### Backend Topics

| Topic | Document | Section |
|-------|----------|---------|
| Go package structure | ARCHITECTURE.md | Backend Architecture |
| HTTP handlers | ARCHITECTURE.md | internal/handlers |
| Download processing | ARCHITECTURE.md | internal/download |
| WebSocket implementation | ARCHITECTURE.md | internal/websocket |
| VLC integration code | ARCHITECTURE.md | internal/vlc |
| Global state management | ARCHITECTURE.md | pkg/config |

### Frontend Topics

| Topic | Document | Section |
|-------|----------|---------|
| JavaScript modules | ARCHITECTURE.md | Frontend Architecture |
| State management | ARCHITECTURE.md | state.js |
| WebSocket client | ARCHITECTURE.md | websocket.js |
| API client | ARCHITECTURE.md | api.js |
| VLC UI integration | ARCHITECTURE.md | vlc.js |
| Download queue UI | ARCHITECTURE.md | download.js |

### API Topics

| Topic | Document | Section |
|-------|----------|---------|
| Download endpoints | API.md | Download Endpoints |
| Queue management | API.md | Queue Management |
| VLC authentication | API.md | VLC Integration |
| WebSocket protocol | API.md | WebSocket API |
| Data models | API.md | Data Models |
| Error handling | API.md | Error Handling |

### Development Topics

| Topic | Document | Section |
|-------|----------|---------|
| Local setup | DEVELOPMENT.md | Development Setup |
| Running tests | DEVELOPMENT.md | Testing |
| Code style | DEVELOPMENT.md | Code Style |
| Contributing | DEVELOPMENT.md | Contributing |
| Troubleshooting | DEVELOPMENT.md | Troubleshooting |
| Building | DEVELOPMENT.md | Building |

---

## 🎓 Learning Path

### Beginner Path
1. Read [README.md](../README.md) - Understand what the project does
2. Follow Quick Start - Get it running
3. Read [DEVELOPMENT.md](DEVELOPMENT.md) - Set up development environment
4. Explore the code with [ARCHITECTURE.md](ARCHITECTURE.md) as reference

### Advanced Path
1. Deep dive into [ARCHITECTURE.md](ARCHITECTURE.md) - Understand all components
2. Study [API.md](API.md) - Learn the complete API
3. Read [DEVELOPMENT.md](DEVELOPMENT.md) - Master the development workflow
4. Start contributing!

---

## 💡 Key Concepts

### The Three Pillars

1. **Backend (Go)**
   - HTTP server handling requests
   - Download queue with yt-dlp integration
   - WebSocket for real-time updates
   - VLC integration for playback control

2. **Frontend (JavaScript)**
   - ES6 modules for clean code organization
   - WebSocket client for live updates
   - VLC authentication UI
   - Download queue visualization

3. **VLC Integration**
   - Challenge-response authentication
   - Server-side cookie management
   - Playback control via HTTP API
   - Automatic video playback

---

## 🔗 External Resources

- [Go Documentation](https://golang.org/doc/)
- [yt-dlp Documentation](https://github.com/yt-dlp/yt-dlp)
- [VLC HTTP Interface](https://wiki.videolan.org/VLC_HTTP_requests/)
- [WebSocket Protocol](https://tools.ietf.org/html/rfc6455)
- [Docker Documentation](https://docs.docker.com/)

---

## 📝 Documentation Maintenance

This documentation is maintained alongside the code. When making changes:

1. **Code changes** → Update ARCHITECTURE.md if architecture changes
2. **API changes** → Update API.md with new endpoints/messages
3. **Setup changes** → Update DEVELOPMENT.md with new requirements
4. **Feature changes** → Update README.md with new features

---

## ❓ Still Have Questions?

- Check the [Troubleshooting](DEVELOPMENT.md#troubleshooting) section
- Review the [FAQ](#) (coming soon)
- Open an issue on GitHub
- Check existing issues for similar questions

---

**Last Updated:** October 26, 2025

**Documentation Version:** 1.0.0
