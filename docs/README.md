# Lab Documentation

Welcome to the Lab project documentation. This index organizes all documentation by category for easy navigation.

---

## 📖 Quick Start

**New to Lab?** Start here:
1. **[README.md](../README.md)** — Project overview and quick start
2. **[Development Guide](development/DEVELOPMENT.md)** — Set up your development environment
3. **[Deployment Guide](deployment/DEPLOYMENT.md)** — Deploy to production

---

## 📂 Documentation Categories

### Development

Guides for setting up and working with the codebase.

| Document | Description |
|----------|-------------|
| **[DEVELOPMENT.md](development/DEVELOPMENT.md)** | Complete development setup guide (Direct, Docker, Vagrant workflows) |
| **[DOCKER_CI.md](development/DOCKER_CI.md)** | Docker-based CI builds and testing |
| **[VAGRANT.md](development/VAGRANT.md)** | Vagrant x86_64 emulation for ARM Mac |

### Deployment

Production deployment guides and architecture documentation.

| Document | Description |
|----------|-------------|
| **[DEPLOYMENT.md](deployment/DEPLOYMENT.md)** | Production deployment guide (systemd, Docker, manual) |
| **[DEPLOYMENT_ARCHITECTURE.md](deployment/DEPLOYMENT_ARCHITECTURE.md)** | System architecture and design decisions |
| **[DEPLOYMENT_SYSTEMD.md](deployment/DEPLOYMENT_SYSTEMD.md)** | Systemd service configuration and management |

### API

Backend API documentation and guidelines.

| Document | Description |
|----------|-------------|
| **[STYLE_GUIDE.md](api/STYLE_GUIDE.md)** | Go API coding standards and conventions |
| **[AUTH.md](api/AUTH.md)** | Authentication system documentation |
| **[API_VERSIONING.md](api/API_VERSIONING.md)** | API versioning strategy |
| **[ERROR_HANDLING.md](api/ERROR_HANDLING.md)** | Error handling patterns and typed errors |
| **[STATUS_API.md](api/STATUS_API.md)** | Status API reference for dashboards and monitoring |
| **[HOMEPAGE_EXAMPLES.md](api/HOMEPAGE_EXAMPLES.md)** | Homepage dashboard integration examples |

### Security

Security documentation and audit reports.

| Document | Description |
|----------|-------------|
| **[SECURITY_AUDIT.md](security/SECURITY_AUDIT.md)** | Comprehensive security audit report (SQL injection, XSS, timeouts, audit logging) |

### Project

Project management and planning documentation.

| Document | Description |
|----------|-------------|
| **[AGENTS.md](../AGENTS.md)** | Guide for AI agents working on the project (ROOT LEVEL) |
| **[PLAN.md](project/PLAN.md)** | Project roadmap and feature status |
| **[RELEASE.md](project/RELEASE.md)** | Release process and versioning |
| **[GITOPS_SPEC.md](project/GITOPS_SPEC.md)** | GitOps specification and implementation plan |
| **[IDEAS.md](project/IDEAS.md)** | Brainstorming and future feature ideas |

### Web

Frontend web application documentation.

| Document | Description |
|----------|-------------|
| **[VM_DIAGNOSTICS.md](web/VM_DIAGNOSTICS.md)** | VM diagnostics and troubleshooting guide |

---

## 🔗 External Resources

- **[GitHub Repository](https://github.com/doomedramen/lab)** — Source code and issues
- **[Contributing Guidelines](../CONTRIBUTING.md)** — How to contribute to the project

---

## 📝 Documentation Maintenance

### Adding New Documentation

1. Place new docs in the appropriate category folder
2. Add entry to this index
3. Update cross-references in related docs
4. Run link checker to verify all links work

### Updating Documentation

When updating code, check if related documentation needs updates:
- **API changes** → Update `docs/api/` docs
- **Deployment changes** → Update `docs/deployment/` docs
- **Security changes** → Update `docs/security/SECURITY_AUDIT.md`
- **New features** → Update `docs/project/PLAN.md`

### Documentation Standards

- Use Markdown format
- Include table of contents for long documents
- Use relative links for internal docs (e.g., `../api/STYLE_GUIDE.md`)
- Use absolute URLs for external links
- Keep code examples up-to-date and tested
- Include "Last updated" date for time-sensitive content

---

## 📊 Documentation Status

| Category | Count | Status |
|----------|-------|--------|
| Development | 3 | ✅ Complete |
| Deployment | 3 | ✅ Complete |
| API | 6 | ✅ Complete |
| Security | 1 | ✅ Complete |
| Project | 4 | ✅ Complete |
| Web | 1 | ✅ Complete |
| **Root Level** | **1** | **✅ Complete** |
| **Total** | **19** | **✅ Current** |

**Last updated:** March 9, 2026
