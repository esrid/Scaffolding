# Modern Web Application Scaffolding Tool
Scattold is a powerful CLI tool that generates a modern, production-ready web application template with best practices and essential features built-in.

## 🚀 Features

### Core Features
- **Modern Architecture**: Clean, modular structure following best practices
- **Authentication System**:
  - Google OAuth integration
  - Traditional email/password authentication
  - Admin panel with OTP verification
- **Database Integration**:
  - PostgreSQL support
  - Automatic migrations
  - Seeding capabilities
- **Frontend Development**:
  - TypeScript support
  - Tailwind CSS integration
  - ESBuild for fast bundling
  - Hot reloading
- **Security**:
  - Environment-based configuration
  - Secure password handling
  - OTP verification for admin access
- **Development Tools**:
  - Docker support
  - Makefile for common tasks
  - Structured logging
  - Graceful server shutdown

## 📁 Project Structure

```
template/
├── web/           # Frontend assets and templates
├── service/       # Business logic layer
├── handler/       # HTTP request handlers
├── db/           # Database interactions
├── utils/        # Utility functions
├── config/       # Configuration management
└── docker-compose.yaml
```

## 🛠️ Getting Started

1. **Clone the Repo**
   ```bash
   git clone github.com/esrid/Scaffolding
   ```

1. **EDIT THE CODE OR BUILD IT**

2. **Create a New Project**
   ```bash
   scattold --name myproject
   ```

3. **Start Development**
   ```bash
   cd myproject
   make esbuild (esbuild cli)   # Start TypeScript bundling
   make tailwind  (tailwind cli) # Start Tailwind CSS processing 
   go run . or air  # Start the server
   ```

## 🔧 Configuration

The tool generates a `.env` file with the following configurations:
- Application environment settings
- Database configuration
- Google OAuth credentials
- Admin panel settings


## 🙏 Acknowledgments

- Go standard library
- PostgreSQL
- Tailwind CSS
- ESBuild
- Deno 
# Scaffolding
