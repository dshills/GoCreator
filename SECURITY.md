# Security Policy

## Supported Versions

GoCreator is currently in active development (v0.1.0-dev). Security updates are provided for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1.0 | :x:                |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

### How to Report

If you discover a security vulnerability in GoCreator, please report it by:

1. **Email**: Send details to the project maintainers (create a private security advisory on GitHub)
2. **GitHub Security Advisory**: Use GitHub's [private security vulnerability reporting](../../security/advisories/new)

### What to Include

Please include the following information in your report:

- **Description**: Clear description of the vulnerability
- **Impact**: What can an attacker do? What is at risk?
- **Reproduction Steps**: Detailed steps to reproduce the issue
- **Affected Versions**: Which versions are affected?
- **Proof of Concept**: Example code or commands that demonstrate the issue
- **Suggested Fix**: If you have ideas for how to fix it (optional)
- **Your Contact Information**: So we can follow up with questions

### Example Report

```
Title: Command Injection in File Path Validation

Description:
The file path validation in pkg/fsops/validator.go does not properly
sanitize user input, allowing arbitrary command execution.

Impact:
An attacker can execute arbitrary shell commands by crafting a malicious
specification file with specially crafted file paths.

Reproduction:
1. Create a specification with file path: `test.go; rm -rf /`
2. Run: gocreator generate malicious-spec.yaml
3. Observe command execution

Affected Versions: 0.1.0-dev and earlier

Proof of Concept:
[Attach minimal example or code snippet]
```

## Response Timeline

- **Initial Response**: Within 48 hours of receiving the report
- **Assessment**: Within 7 days, we will confirm the vulnerability and its severity
- **Fix Development**: Depends on severity and complexity
- **Disclosure**: Coordinated disclosure after fix is available

## Security Best Practices for Users

### API Key Protection

**Never commit API keys to version control:**

```bash
# Use environment variables
export ANTHROPIC_API_KEY=sk-ant-...

# Or use a .env file (already in .gitignore)
echo "ANTHROPIC_API_KEY=sk-ant-..." > .env
```

### File System Security

**GoCreator has built-in file system restrictions:**

- All file operations are bounded to the configured root directory
- Path traversal attacks (`../`) are prevented
- File operations are logged in execution audit logs

**Additional precautions:**

```yaml
# In .gocreator.yaml, restrict the generation root
workflow:
  root_dir: ./generated  # Bounded directory
  allow_commands:        # Whitelist of allowed commands
    - go
    - git
    - golangci-lint
```

### Specification Validation

**Validate specifications before generation:**

```bash
# Review specification content
cat my-spec.yaml

# Use dump-fcs to see what will be generated
gocreator dump-fcs my-spec.yaml

# Use dry-run mode (when available)
gocreator generate my-spec.yaml --dry-run
```

### LLM Provider Security

**Protect your LLM provider credentials:**

- Never share your API keys
- Use separate API keys for development and production
- Rotate keys regularly
- Monitor API usage for anomalies
- Use provider-specific security features (rate limiting, IP allowlists)

### Generated Code Review

**Always review generated code before use:**

- Run validation: `gocreator validate ./generated`
- Review for sensitive data leaks
- Check for insecure patterns
- Run security scanners: `gosec ./...`

## Known Security Considerations

### LLM-Generated Code

**GoCreator uses LLMs to generate code. Consider:**

- Generated code should be reviewed before production use
- LLMs can occasionally produce insecure patterns
- Validation (build, lint, test) catches many issues, but not all
- Use `gosec` or similar tools for security scanning

### Execution Audit Logs

**GoCreator logs all operations for security and debugging:**

- Logs are stored in `.gocreator/execution.jsonl`
- Logs may contain specification details
- Review and secure logs appropriately
- Logs are excluded from git by default

### Dependency Security

**Keep dependencies updated:**

```bash
# Check for vulnerabilities
go list -json -m all | go run github.com/sonatard/go-mod-graph@latest

# Update dependencies
go get -u ./...
go mod tidy
```

## Security Features

### Bounded File Operations

- All file writes are restricted to the configured root directory
- Path traversal prevention
- Permission validation before operations

### Command Execution Restrictions

- Only whitelisted commands can be executed
- Commands are logged in execution audit
- No arbitrary command execution

### Deterministic Generation

- Same specification always produces the same output
- No random or unpredictable behavior
- Reproducible for security audits

### Logging and Auditability

- All LLM calls are logged
- All file operations are logged
- Execution audit trail in JSONL format

## Disclosure Policy

When a security vulnerability is confirmed:

1. **Private Fix**: We develop and test a fix privately
2. **Security Advisory**: We create a GitHub Security Advisory
3. **Patch Release**: We release a patched version
4. **Public Disclosure**: We publish the advisory with details and fix
5. **Credit**: We credit the reporter (unless they prefer anonymity)

## Contact

For security concerns that are not vulnerabilities (questions, best practices, etc.), please:

- Open a [GitHub Discussion](../../discussions)
- Check the [documentation](docs/)
- Review [CONTRIBUTING.md](CONTRIBUTING.md)

---

Thank you for helping keep GoCreator and its users safe!
