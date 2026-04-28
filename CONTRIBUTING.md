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
- Run `go test ./...` to execute all tests
- Maintain test coverage

### Pull Request Process
1. Update the README.md if needed
2. Ensure all tests pass
3. Update documentation for any API changes
4. Request review from maintainers

## Code of Conduct

This project follows the Contributor Covenant Code of Conduct. By participating, you are expected to uphold this code.

## License

By contributing to this project, you agree that your contributions will be licensed under the MIT License.