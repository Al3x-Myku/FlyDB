# Contributing to FlyDB

Thank you for your interest in contributing to FlyDB! This document provides guidelines and information for contributors.

## Project Philosophy

FlyDB is designed with three core principles:

1. **Educational** - Code should be clear, well-documented, and demonstrative of database internals
2. **Minimal** - Standard library only, no external dependencies
3. **Correct** - Robust error handling, proper concurrency, and tested code

## Getting Started

### Prerequisites

- Go 1.22 or later
- Basic understanding of Go and databases
- Familiarity with Git and GitHub

### Setting Up Development Environment

```bash
# Clone the repository
git clone https://github.com/Al3x-Myku/FlyDB.git
cd FlyDB

# Run tests
go test ./pkg/...

# Run the main example
go run cmd/example/main.go
```

## Development Guidelines

### Code Style

- Follow standard Go conventions (use `gofmt`, `golint`)
- Write clear, descriptive comments for all exported functions
- Include package-level documentation
- Use meaningful variable names

### Testing

- All new features must include tests
- Maintain or improve test coverage
- Test files should be in the same package as the code they test
- Use table-driven tests where appropriate

Example:
```go
func TestNewFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"basic", "input", "output"},
        {"edge case", "", ""},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := NewFeature(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Documentation

- Update README.md if adding user-facing features
- Update docs/ if making architectural changes
- Include code examples in documentation
- Keep QUICKSTART.md up to date

### Commit Messages

Use clear, descriptive commit messages:

```
Add bloom filter optimization to block index

- Implement probabilistic filter for block lookups
- Reduces unnecessary disk reads by ~30%
- Add tests for filter accuracy
```

Format: `<type>: <short summary>`

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `test`: Adding or updating tests
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `perf`: Performance improvement

## Pull Request Process

1. **Fork** the repository
2. **Create a branch** for your feature (`git checkout -b feat/amazing-feature`)
3. **Make your changes**
4. **Add tests** for your changes
5. **Run tests** (`go test ./pkg/...`)
6. **Commit** with clear messages
7. **Push** to your fork
8. **Open a Pull Request**

### PR Checklist

- [ ] Code follows project style guidelines
- [ ] All tests pass (`go test ./pkg/...`)
- [ ] Code builds without errors (`go build ./...`)
- [ ] New code includes tests
- [ ] Documentation updated (if needed)
- [ ] Commit messages are clear
- [ ] PR description explains the changes

## Areas for Contribution

### High Priority

- [ ] **Compaction**: Implement background merge of old blocks
- [ ] **Deletion**: Add tombstone records for delete operations
- [ ] **Benchmarks**: Add comprehensive performance benchmarks
- [ ] **Fuzzing**: Add fuzz tests for TOON parser
- [ ] **Error recovery**: Improve handling of corrupted blocks

### Medium Priority

- [ ] **Range queries**: Support for scanning multiple documents
- [ ] **Secondary indexes**: Index on fields other than ID
- [ ] **Compression**: TOON block compression
- [ ] **Bloom filters**: Reduce unnecessary disk reads
- [ ] **WAL**: Write-ahead log for crash recovery

### Lower Priority

- [ ] **CLI tool**: Command-line interface for database operations
- [ ] **HTTP API**: REST API server
- [ ] **Metrics**: Expose Prometheus-style metrics
- [ ] **Docker**: Containerized deployment
- [ ] **More examples**: Additional usage examples

## Code of Conduct

### Our Standards

- Be respectful and inclusive
- Welcome newcomers and help them learn
- Focus on what's best for the project
- Accept constructive criticism gracefully
- Show empathy towards others

### Not Acceptable

- Harassment or discrimination
- Trolling or insulting comments
- Publishing others' private information
- Other unethical or unprofessional conduct

## Questions?

- Open an issue for bugs or feature requests
- Start a discussion for general questions
- Check existing issues before creating new ones

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Recognition

Contributors will be recognized in:
- GitHub contributors page
- Release notes (for significant contributions)
- README.md (for major features)

---

Thank you for contributing to FlyDB! 🚀
