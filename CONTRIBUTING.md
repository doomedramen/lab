# Contributing to Lab

Thank you for your interest in contributing to Lab! We welcome all types of contributions, including bug reports, feature requests, documentation improvements, and code changes.

## How to Contribute

### 1. Report Bugs or Request Features
- Check the [Issues](https://github.com/doomedramen/lab/issues) page to see if your topic is already being discussed.
- If not, open a new issue with a clear description of the problem or suggestion.

### 2. Code Contributions
1. **Fork** the repository on GitHub.
2. **Clone** your fork locally.
3. **Create a branch** for your feature or fix: `git checkout -b feature/your-feature-name`.
4. **Implement your changes**. Ensure you follow the project's coding style (see [STYLE_GUIDE.md](./apps/api/STYLE_GUIDE.md) for Go).
5. **Run tests**:
   - Web: `pnpm --filter web test:e2e`
   - API: `pnpm --filter api test`
6. **Commit your changes**: `git commit -m 'Description of your change'`.
7. **Push to your fork**: `git push origin feature/your-feature-name`.
8. **Open a Pull Request** against the `main` branch of the original repository.

### 3. Documentation
Improvements to the README, Deployment Guide, or API documentation are always appreciated.

## Development Setup

See the [README.md](./README.md) and [DEPLOYMENT.md](./DEPLOYMENT.md) for detailed instructions on setting up your local development environment.

## Coding Standards
- **Frontend**: TypeScript, React, Next.js. Use `pnpm format` to ensure consistent formatting.
- **Backend**: Go. Follow standard Go idioms and the [STYLE_GUIDE.md](./apps/api/STYLE_GUIDE.md).
- **API**: Protocol Buffers (Buf). Define new services in `packages/proto`.

## License
By contributing, you agree that your contributions will be licensed under the project's [MIT License](./LICENSE).
