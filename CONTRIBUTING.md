# 🤝 Contributing to Telescopio API

Thank you for your interest in contributing to Telescopio API! This project implements cutting-edge research in distributed voting systems and mechanism design, and we welcome contributions from developers, researchers, and domain experts.

---

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Testing](#testing)
- [Documentation](#documentation)
- [Areas for Contribution](#areas-for-contribution)
- [Community & Support](#community--support)

---

## 🤖 Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct:

- **Be respectful** and inclusive to all contributors, regardless of experience level
- **Be collaborative** and constructive in code reviews and discussions
- **Be patient** with newcomers and help them understand the mathematical foundations
- **Respect intellectual property** and properly cite academic sources
- **Focus on the science** - decisions should be based on mathematical rigor and empirical evidence

---

## 🚀 Getting Started

### Prerequisites

- **Go 1.23+**
- **PostgreSQL 15+**
- **Docker & Docker Compose** (optional but recommended)
- **Git**
- Basic understanding of voting theory and mechanism design is helpful but not required

### Setup Development Environment

1. **Fork and clone the repository**:
   ```bash
   git clone https://github.com/your-username/telescopio-api.git
   cd telescopio-api
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Set up environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your database configuration
   ```

4. **Start development environment**:
   ```bash
   # Option 1: Using Docker (recommended)
   docker-compose up -d postgres
   
   # Option 2: Local PostgreSQL
   # Make sure PostgreSQL is running and configured in .env
   ```

5. **Run database migrations**:
   ```bash
   go run cmd/migrate/main.go
   ```

6. **Start the development server**:
   ```bash
   go run cmd/api/main.go
   ```

7. **Verify setup**:
   ```bash
   curl http://localhost:8080/api/v1/health
   ```

---

## 🔄 Development Workflow

### Branch Naming Convention

We use a clear, descriptive branching strategy:

- **`main`**: Production-ready code (protected)
- **`develop`**: Integration branch for features (protected)
- **`feature/description`**: New features (`feature/quality-assessment-improvements`)
- **`fix/description`**: Bug fixes (`fix/assignment-generation-edge-case`)
- **`docs/description`**: Documentation updates (`docs/api-reference-update`)
- **`refactor/description`**: Code refactoring (`refactor/voting-service-cleanup`)
- **`perf/description`**: Performance improvements (`perf/database-query-optimization`)

### Development Process

1. **Check existing issues** before starting work to avoid duplication

2. **Create or claim an issue** describing what you plan to work on

3. **Create feature branch**:
   ```bash
   git checkout develop
   git pull origin develop
   git checkout -b feature/your-descriptive-feature-name
   ```

4. **Make your changes** following our coding standards

5. **Write comprehensive tests** (required for all new features)

6. **Update documentation** if you've changed APIs or algorithms

7. **Test thoroughly** - run all tests and verify functionality

8. **Commit** using conventional commit format

9. **Push and create Pull Request** against the `develop` branch

---

## 💻 Coding Standards

### Go Style Guidelines

We follow standard Go conventions plus additional project-specific standards:

#### General Standards
- Use `gofmt` for formatting
- Pass `go vet` without warnings
- Use meaningful variable names (prefer clarity over brevity)
- Write self-documenting code with clear logic flow
- Handle errors explicitly and appropriately

#### Mathematical Code
- Include references to academic papers in comments
- Use variable names that match mathematical notation when possible
- Document complex algorithms with step-by-step comments
- Include numerical examples in comments for clarity

#### Example:
```go
// CalculateModifiedBordaCount implements the MBC formula from Merrifield & Saari (2009)
// Formula: MBC(f_j) = (1 / m(m-1)) × Σ(m - R_i(f_j))
// where m = attachments per evaluator, R_i(f_j) = rank given to submission f_j by evaluator i
func (vs *VotingService) CalculateModifiedBordaCount(
    eventID uuid.UUID, 
    config *VotingConfiguration,
) (*VotingResults, error) {
    // Implementation with clear variable names matching the formula
    m := config.AttachmentsPerEvaluator // m from the formula
    // ... rest of implementation
}
```

#### Project Structure
- Follow the established domain-driven design patterns
- Keep business logic in domain packages
- Use repositories for data access
- Separate HTTP handlers from business logic

#### Database
- Use UUIDs for all primary keys
- Follow GORM conventions
- Write proper database migrations
- Include proper constraints and indexes

---

## 📝 Commit Guidelines

### Conventional Commits Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Commit Types

- **`feat`**: New feature or enhancement
- **`fix`**: Bug fix
- **`docs`**: Documentation changes only
- **`refactor`**: Code refactoring without functionality change
- **`test`**: Adding or updating tests
- **`perf`**: Performance improvements
- **`chore`**: Maintenance tasks, dependency updates
- **`style`**: Code style changes (formatting, etc.)

### Examples

```bash
# Feature commits
feat(voting): implement Modified Borda Count algorithm with quality assessment
feat(api): add distributed assignment generation endpoint
feat(database): add participant quality tracking with historical metrics

# Fix commits
fix(assignments): prevent self-evaluation in distributed assignment generation
fix(voting): correct edge case in quality assessment for tied ranks
fix(database): resolve UUID foreign key constraint violations

# Documentation commits
docs(api): add comprehensive endpoint examples with curl commands
docs(math): explain Modified Borda Count formula with step-by-step examples
docs(contributing): improve development setup instructions

# Performance commits
perf(database): optimize vote aggregation queries with proper indexing
perf(algorithm): reduce MBC calculation complexity from O(n²) to O(n log n)
```

---

## 🔍 Pull Request Process

### Before Creating PR

1. **Ensure your branch is up to date**:
   ```bash
   git checkout develop
   git pull origin develop
   git checkout your-branch
   git rebase develop
   ```

2. **Run comprehensive tests**:
   ```bash
   # Run all tests
   go test ./...
   
   # Run with race detection
   go test -race ./...
   
   # Check test coverage
   go test -cover ./...
   ```

3. **Run code quality checks**:
   ```bash
   # Format code
   go fmt ./...
   
   # Vet code
   go vet ./...
   
   # Run linter (if available)
   golangci-lint run
   ```

4. **Verify functionality**:
   ```bash
   # Start the server
   go run cmd/api/main.go
   
   # Test your changes manually
   curl http://localhost:8080/api/v1/your-endpoint
   ```

### PR Requirements Checklist

**Before submitting, ensure your PR meets these requirements:**

- [ ] **Clear, descriptive title** following conventional commit format
- [ ] **Detailed description** explaining what changed and why
- [ ] **Tests added/updated** for all new functionality (mandatory for `feat` commits)
- [ ] **Documentation updated** if APIs, algorithms, or behavior changed
- [ ] **No merge conflicts** with develop branch
- [ ] **All CI checks passing**
- [ ] **Mathematical accuracy verified** if algorithms were changed
- [ ] **Academic references cited** for new algorithms or modifications

### PR Template

When creating your PR, please use this template:

```markdown
## 📝 Description
Brief description of changes and motivation. Reference any related issues.

Closes #[issue_number]

## 🔄 Type of Change
- [ ] 🐛 Bug fix (non-breaking change that fixes an issue)
- [ ] ✨ New feature (non-breaking change that adds functionality)
- [ ] 💥 Breaking change (fix or feature that causes existing functionality to change)
- [ ] 📚 Documentation update
- [ ] 🔧 Refactoring (no functional changes)
- [ ] ⚡ Performance improvement

## 🧮 Mathematical Impact
- [ ] No mathematical changes
- [ ] Algorithm improvement (explain what improved)
- [ ] New mathematical feature (cite academic sources)
- [ ] Mathematical validation fix (explain the issue)

## 🧪 Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated  
- [ ] Mathematical accuracy tests added
- [ ] Manual testing completed
- [ ] Edge cases considered and tested

## 📚 Documentation
- [ ] API documentation updated
- [ ] Code comments added/improved
- [ ] Mathematical explanations added
- [ ] README updated if needed
- [ ] Academic references cited

## ⚡ Performance Impact
- [ ] No performance impact
- [ ] Performance improved (provide metrics)
- [ ] Performance considerations documented

## 🔒 Breaking Changes
If this is a breaking change, describe what breaks and how to migrate:
- 
```

---

## 🧪 Testing

### Test Categories

#### 1. Unit Tests ⚡
Test individual functions in isolation. **Required for all new features.**

```go
func TestCalculateModifiedBordaCount(t *testing.T) {
    tests := []struct {
        name     string
        votes    []*Vote
        config   *VotingConfiguration
        expected *VotingResults
        wantErr  bool
    }{
        {
            name: "basic MBC calculation with 3 attachments",
            votes: []*Vote{
                // Test data
            },
            config: &VotingConfiguration{
                AttachmentsPerEvaluator: 3,
            },
            expected: &VotingResults{
                // Expected results
            },
            wantErr: false,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

#### 2. Integration Tests 🔗
Test component interactions, especially with the database.

```go
func TestVotingWorkflowIntegration(t *testing.T) {
    // Set up test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    // Test complete voting workflow
    // 1. Create event
    // 2. Generate assignments
    // 3. Submit votes
    // 4. Calculate results
}
```

#### 3. Mathematical Accuracy Tests 📊
Verify algorithms against known results from academic literature.

```go
func TestMBCAgainstPaperResults(t *testing.T) {
    // Test using data from Merrifield & Saari (2009)
    // Verify our implementation matches their examples
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage report
go test -cover ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific test package
go test ./internal/domain/vote/...

# Run tests with race detection
go test -race ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestCalculateModifiedBordaCount ./internal/domain/vote/
```

### Coverage Requirements

- **New features**: Minimum 80% test coverage
- **Critical algorithms**: Minimum 95% test coverage
- **Mathematical functions**: 100% test coverage with edge cases

---

## 📚 Documentation

### Required Documentation Updates

All public functions must have comprehensive Go doc comments:

```go
// Package vote implements the mathematical models and algorithms
// for distributed voting based on the Modified Borda Count mechanism
// as described in Merrifield & Saari (2009).
//
// The package provides functionality for:
//   - Generating fair assignment distributions
//   - Calculating Modified Borda Count scores
//   - Assessing participant evaluation quality
//   - Implementing incentive mechanisms
package vote

// CalculateQualityAssessment implements the evaluator quality assessment
// algorithm described in Section 4.2 of Merrifield & Saari (2009).
//
// The quality metric Q_i measures how well evaluator i's rankings align
// with the community consensus, calculated as:
//
//   Q_i = 1 - (2/(m(m-1))) × Σ|R_i(f_j) - RelativeRank_G(f_j, A(p_i))|
//
// Parameters:
//   - eventID: UUID of the voting event
//   - globalResults: The community consensus ranking
//   - config: Voting configuration parameters
//
// Returns:
//   - qualityMap: Map of participant UUID to quality score [0,1]
//   - error: Computation errors or validation failures
//
// Quality scores interpretation:
//   - Q_i = 1.0: Perfect alignment with consensus
//   - Q_i = 0.5: Random/average alignment
//   - Q_i = 0.0: Complete opposite of consensus
func CalculateQualityAssessment(
    eventID uuid.UUID,
    globalResults []AttachmentResult,
    config *VotingConfiguration,
) (map[uuid.UUID]float64, error) {
    // Implementation...
}
```

---

## 🌟 Good First Issues

Perfect for newcomers to the project:

- **Documentation improvements**: Fix typos, add examples, improve clarity
- **Code comments**: Add explanatory comments to complex functions  
- **Test coverage**: Add test cases for existing functionality
- **Error messages**: Improve error message clarity and helpfulness
- **Validation**: Add input validation for edge cases
- **Examples**: Create more usage examples and tutorials

---

## 🌟 Community & Support

### Getting Help

- **📋 Issues**: [Create a GitHub issue](https://github.com/gravadigital/telescopio-api/issues) for bugs, feature requests, or questions
- **💬 Discussions**: Use [GitHub Discussions](https://github.com/gravadigital/telescopio-api/discussions) for general questions and community interaction
- **📧 Email**: [maintainer-email@domain.com] for private inquiries
- **📚 Documentation**: Check our [comprehensive docs](docs/) for detailed information

### Communication Guidelines

- **Be specific**: Include relevant details, error messages, and steps to reproduce
- **Search first**: Check existing issues and discussions before creating new ones
- **Use appropriate channels**: Bugs go to Issues, questions to Discussions
- **Be patient**: Maintainers are often volunteers with other commitments

### Recognition

Contributors are recognized in several ways:
- Listed in `CONTRIBUTORS.md`
- Mentioned in release notes for significant contributions
- Invited to become maintainers for consistent high-quality contributions

---

## 📄 License

By contributing to Telescopio API, you agree that your contributions will be licensed under the same [MIT License](LICENSE) as the project.

Your contributions should be your own work or properly attributed if building on others' work. When implementing academic algorithms, ensure proper citation and respect for intellectual property.

---

## 🙏 Acknowledgments

This project builds on the groundbreaking research of:
- **Merrifield, M. R. & Saari, D. G.** for the theoretical foundation
- **The broader voting theory community** for decades of research
- **Open source contributors** who make projects like this possible

---

**Thank you for contributing to making resource allocation more fair and scalable! 🚀**

*Whether you're fixing a typo, optimizing an algorithm, or implementing a new feature, every contribution helps advance the science of distributed decision-making.*
