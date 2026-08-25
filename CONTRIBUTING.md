# Contributing to Retail POS System

Thank you for your interest in contributing to the Retail POS System! This document provides guidelines and information for contributors.

## Development Setup

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/your-username/retail-pos-system.git
   cd retail-pos-system
   ```

3. **Set up development environment** (see README.md for detailed instructions)

4. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Guidelines

### Code Style
- Follow Go naming conventions
- Use `gofmt` for formatting
- Add documentation for exported functions
- Write clear, concise commit messages

### Testing
- Write tests for new features
- Ensure all tests pass before submitting PR
- Run backend tests with required env vars:
  ```bash
  TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos TEST_DB_PASSWORD=admin123 \
  DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only \
  go test -p 1 -count=1 ./...
  ```
  Or use `make test` (reads from `.env.test` if present).
- Run frontend unit tests: `cd web && npm run test:run`
- Run E2E tests: `npx playwright test --reporter=list` (run from repo root; requires both backend and frontend servers running)
- Use targeted testing: `go test -p 1 -count=1 ./internal/<package>/...` for backend, `cd web && npx vitest run <path>` for frontend
- Maintain test coverage

### Pull Request Process
1. Update the README.md if needed
2. Ensure all tests pass
3. Update documentation for any API changes
4. Request review from maintainers

### Git Policy
- Never auto-commit on each change. Commits must be made manually when ready.
- Write clear, concise commit messages that match the repo style.

## Code of Conduct

This project follows the Contributor Covenant Code of Conduct. By participating, you are expected to uphold this code.

## License

By contributing to this project, you agree that your contributions will be licensed under the MIT License.