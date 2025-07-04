# SonarQube Integration Setup

This document explains how to set up and use SonarQube for code quality analysis in the IMAP Backup project.

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose (for local SonarQube)
- SonarQube Scanner CLI (for manual analysis)

## Local SonarQube Setup

### 1. Start Local SonarQube Server

```bash
# Start SonarQube and PostgreSQL
docker-compose -f docker-compose.sonar.yml up -d

# Wait for services to start (may take 2-3 minutes)
docker-compose -f docker-compose.sonar.yml logs -f sonarqube
```

### 2. Access SonarQube Web Interface

- URL: http://localhost:9000
- Default credentials: admin/admin
- You'll be prompted to change the password on first login

### 3. Create Project

1. Click "Create Project" → "Manually"
2. Project key: `imap-backup`
3. Display name: `IMAP Backup Tool`
4. Click "Set Up"

### 4. Generate Token

1. Go to "My Account" → "Security"
2. Generate a new token with name: `imap-backup-token`
3. Copy the token for use in analysis

## Development Tools Installation

Install required linting and analysis tools:

```bash
make install-tools
```

This installs:
- golangci-lint (comprehensive Go linter)
- staticcheck (advanced static analysis)
- gosec (security analysis)

## Running Analysis

### Local Analysis

```bash
# Run all quality checks
make quality

# Generate coverage reports and prepare for SonarQube
make sonar-prepare

# Run SonarQube analysis (requires sonar-scanner CLI)
export SONAR_HOST_URL=http://localhost:9000
export SONAR_TOKEN=your_token_here
make sonar
```

### Docker-based Analysis

```bash
# Run analysis using Docker (no need to install sonar-scanner)
export SONAR_HOST_URL=http://localhost:9000
export SONAR_TOKEN=your_token_here
make sonar-docker
```

## CI/CD Integration

### GitHub Actions

The project includes a GitHub Actions workflow (`.github/workflows/sonarqube.yml`) that:

1. Runs tests with coverage
2. Executes static analysis tools
3. Uploads results to SonarQube

Required GitHub Secrets:
- `SONAR_TOKEN`: Your SonarQube authentication token
- `SONAR_HOST_URL`: Your SonarQube server URL

### Manual CI Pipeline

```bash
# Full CI pipeline
make ci
```

## Quality Gates

The project is configured with the following quality standards:

### Coverage Requirements
- Minimum test coverage: 80%
- New code coverage: 90%

### Code Smells
- Duplicated lines: < 3%
- Maintainability rating: A
- Technical debt ratio: < 5%

### Security
- Security hotspots: 0
- Vulnerabilities: 0
- Security rating: A

### Reliability
- Bugs: 0
- Reliability rating: A

## Tools and Reports

### Generated Reports

1. **Coverage Report**: `coverage.out` (for SonarQube) and `coverage.html` (for viewing)
2. **Lint Report**: `golangci-lint-report.xml` (checkstyle format)
3. **Go Vet Report**: `govet-report.out`
4. **Security Report**: `gosec-report.json` (SonarQube format)

### Quality Checks

```bash
# Individual checks
make lint        # Run golangci-lint
make staticcheck # Run staticcheck
make security    # Run gosec
make test        # Run tests
make coverage    # Generate coverage report

# All checks at once
make quality
```

## Configuration Files

- `sonar-project.properties`: SonarQube project configuration
- `.golangci.yml`: golangci-lint configuration
- `.github/workflows/sonarqube.yml`: GitHub Actions workflow

## Exclusions

The analysis excludes:
- Test files (`**/*_test.go`)
- Vendor directory (`**/vendor/**`)
- Backup data (`**/backup/**`, `**/test-**`)
- Generated code (`**/*.pb.go`)

## Troubleshooting

### Common Issues

1. **SonarQube fails to start**
   ```bash
   # Check system limits
   sysctl vm.max_map_count
   # Should be at least 524288
   
   # Fix if needed (Linux)
   sudo sysctl -w vm.max_map_count=524288
   ```

2. **Analysis fails with authentication error**
   - Verify SONAR_TOKEN is correctly set
   - Check token permissions in SonarQube UI

3. **Tools not found**
   ```bash
   # Install development tools
   make install-tools
   ```

## Best Practices

1. **Run quality checks before commits**
   ```bash
   make quality
   ```

2. **Monitor coverage trends**
   - Aim to maintain or improve coverage with each PR
   - Review coverage reports for uncovered critical paths

3. **Address security findings immediately**
   - Security hotspots should be reviewed and resolved
   - Use `gosec` locally to catch issues early

4. **Keep technical debt low**
   - Address code smells promptly
   - Refactor complex functions (cognitive complexity > 15)

## Local Development Workflow

```bash
# 1. Install tools (once)
make install-tools

# 2. Development cycle
make test        # Quick test
make quality     # Full quality check

# 3. Before commit
make ci          # Full CI pipeline

# 4. SonarQube analysis (optional)
make sonar-prepare
make sonar       # or make sonar-docker
```