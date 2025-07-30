# Backend Unit Test Suite - New Structure

This directory contains comprehensive unit tests for the Backend-Bluelock-007 project with a new modular structure organized by module and function.

## 📁 New Directory Structure

```
test/
├── README_NEW_STRUCTURE.md      # This file
├── test_utils.go                # Test utilities and performance measurement
├── run_tests_new.sh             # New shell script for running tests
├── student/                     # Student module tests
│   ├── student_creation_test.go # Student creation functionality tests
│   ├── student_validation_test.go # Student validation tests
│   └── student_skills_test.go   # Student skills calculation tests
├── activity/                    # Activity module tests
│   └── activity_creation_test.go # Activity creation tests
├── auth/                        # Auth module tests
│   └── login_test.go            # Login functionality tests
├── form/                        # Form module tests
│   ├── form_creation_test.go    # Form creation tests
│   ├── question_test.go         # Question handling tests
│   └── submission_test.go       # Form submission tests
├── course/                      # Course module tests
│   └── course_management_test.go # Course management tests
├── enrollment/                  # Enrollment module tests
│   └── enrollment_test.go       # Enrollment tests
├── food/                        # Food module tests
│   └── food_test.go             # Food management tests
└── admin/                       # Admin module tests
    └── admin_test.go            # Admin functionality tests
```

## 🚀 Quick Start

### Prerequisites

- Go 1.23.6 or higher
- Required dependencies (run `go mod tidy`)

### Running Tests

#### 1. Run All Tests
```bash
# From the project root directory
./test/run_tests_new.sh
```

#### 2. Run Specific Module Tests
```bash
# Run all student tests
./test/run_tests_new.sh module student

# Run all activity tests
./test/run_tests_new.sh module activity

# Run all auth tests
./test/run_tests_new.sh module auth
```

#### 3. Run Specific Function Tests
```bash
# Run student creation tests only
./test/run_tests_new.sh function student creation

# Run student validation tests only
./test/run_tests_new.sh function student validation

# Run student skills tests only
./test/run_tests_new.sh function student skills

# Run activity creation tests only
./test/run_tests_new.sh function activity creation

# Run auth login tests only
./test/run_tests_new.sh function auth login
```

#### 4. List Available Tests
```bash
./test/run_tests_new.sh list
```

#### 5. Show Help
```bash
./test/run_tests_new.sh help
```

### Using Go Test Directly

```bash
# Run all tests
go test ./test/...

# Run specific module tests
go test ./test/student/...
go test ./test/activity/...
go test ./test/auth/...

# Run specific test file
go test ./test/student/student_creation_test.go

# Run with verbose output
go test -v ./test/...

# Run with coverage
go test -cover ./test/...
```

## 📊 Performance Measurement

The test suite includes built-in performance measurement that tracks:

- **Individual Test Duration**: Each test is timed and reported
- **Module Performance**: Aggregated timing for each module
- **Function Performance**: Timing for specific functions within modules
- **Overall Suite Performance**: Total execution time
- **Performance Assertions**: Tests can include performance requirements

### Performance Metrics

- ⏱️ **Test Timer**: Measures individual test execution time
- 📈 **Performance Assertion**: Validates test performance against thresholds
- 📊 **Test Suite Results**: Aggregated performance statistics
- 🎯 **Performance Analysis**: Automatic performance recommendations

### Example Performance Output

```
⏱️  Basic Student Creation took 45.2µs
⏱️  Student Creation With ID took 23.1µs
⏱️  Student Creation Minimal took 67.8µs

📊 Test Suite Summary: Student Creation Tests
   Total Tests: 5
   Passed: 5 ✅
   Failed: 0 ❌
   Total Time: 156.1µs
   Average Time: 31.2µs
   Success Rate: 100.00%
```

## 🧪 Test Categories by Module

### 1. Student Module (`test/student/`)

Tests for student-related functionality:

- **student_creation_test.go**: Student creation and initialization
- **student_validation_test.go**: Student data validation
- **student_skills_test.go**: Student skills calculation and analysis

**Key Test Areas:**
- Student data creation and initialization
- Field validation (code, name, email, etc.)
- Skill calculations and rankings
- Performance analysis

### 2. Activity Module (`test/activity/`)

Tests for activity management:

- **activity_creation_test.go**: Activity creation and setup

**Key Test Areas:**
- Activity creation and initialization
- Activity type validation
- State management
- Food voting system

### 3. Auth Module (`test/auth/`)

Tests for authentication and authorization:

- **login_test.go**: Login functionality

**Key Test Areas:**
- User authentication
- Credential validation
- Token generation and validation
- Role-based access control

### 4. Form Module (`test/form/`)

Tests for form management:

- **form_creation_test.go**: Form creation and setup
- **question_test.go**: Question handling
- **submission_test.go**: Form submission processing

**Key Test Areas:**
- Form creation and validation
- Question type handling
- Form submission processing
- Data validation

### 5. Course Module (`test/course/`)

Tests for course management:

- **course_management_test.go**: Course CRUD operations

**Key Test Areas:**
- Course creation and management
- Course validation
- Course relationships

### 6. Enrollment Module (`test/enrollment/`)

Tests for enrollment management:

- **enrollment_test.go**: Enrollment operations

**Key Test Areas:**
- Student enrollment
- Enrollment validation
- Capacity management

### 7. Food Module (`test/food/`)

Tests for food management:

- **food_test.go**: Food-related operations

**Key Test Areas:**
- Food item management
- Food voting system
- Food preferences

### 8. Admin Module (`test/admin/`)

Tests for admin functionality:

- **admin_test.go**: Admin operations

**Key Test Areas:**
- Admin user management
- System administration
- Access control

## 📈 Test Coverage

The test suite aims for comprehensive coverage including:

- ✅ **Happy Path**: Normal operation scenarios
- ❌ **Error Cases**: Error handling and edge cases
- 🔒 **Security**: Input validation and security checks
- ⚡ **Performance**: Performance requirements and benchmarks
- 🔄 **Integration**: Component interaction testing

## 🛠️ Adding New Tests

### 1. Create Module Directory

```bash
mkdir -p test/newmodule
```

### 2. Create Function Test File

```go
// test/newmodule/newmodule_function_test.go
package newmodule

import (
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "Backend-Bluelock-007/test"
    "Backend-Bluelock-007/src/models"
)

func TestNewModuleFunction(t *testing.T) {
    suiteResult := test.NewTestSuiteResult("New Module Function Tests")
    defer suiteResult.PrintSummary()
    
    t.Run("TestNewFunction", func(t *testing.T) {
        timer := test.NewTestTimer("New Function")
        defer func() {
            duration := timer.Stop()
            suiteResult.AddResult(test.TestResult{
                Name:     "New Function",
                Duration: duration,
                Passed:   true,
            })
            test.PerformanceAssertion(t, "New Function", duration, 100*time.Microsecond)
        }()
        
        // Your test logic here
        assert.True(t, true)
    })
}
```

### 3. Add Performance Measurement

```go
// Use test timer for performance measurement
timer := test.NewTestTimer("Test Name")
defer func() {
    duration := timer.Stop()
    suiteResult.AddResult(test.TestResult{
        Name:     "Test Name",
        Duration: duration,
        Passed:   true,
    })
    test.PerformanceAssertion(t, "Test Name", duration, 1*time.Millisecond)
}()
```

### 4. Update Test Runner

Add your new module to the test runner in `run_tests_new.sh`:

```bash
# In the modules array
declare -A modules=(
    ["NewModule"]="./test/newmodule"
    # ... other modules
)

# In run_specific_module function
case $module_name in
    "newmodule")
        run_module_tests "NewModule" "./test/newmodule"
        ;;
    # ... other cases
esac

# In run_specific_function function
case $module_name in
    "newmodule")
        case $function_name in
            "function")
                run_function_tests "NewModule" "Function" "./test/newmodule/newmodule_function_test.go"
                ;;
            # ... other functions
        esac
        ;;
    # ... other modules
esac
```

## 🔧 Configuration

### Environment Variables

Set these environment variables for testing:

```bash
export GO_ENV=test
export TEST_MODE=true
```

### Test Timeouts

Default test timeout is 30 seconds. You can modify this in `run_tests_new.sh`:

```bash
go test -v -timeout 30s "$test_file"
```

### Performance Thresholds

Adjust performance thresholds in your tests:

```go
// For fast operations (microseconds)
test.PerformanceAssertion(t, "Fast Operation", duration, 100*time.Microsecond)

// For medium operations (milliseconds)
test.PerformanceAssertion(t, "Medium Operation", duration, 1*time.Millisecond)

// For slow operations (seconds)
test.PerformanceAssertion(t, "Slow Operation", duration, 1*time.Second)
```

## 📊 Coverage Reports

Generate coverage reports to identify untested code:

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./test/...

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# View coverage percentage
go tool cover -func=coverage.out
```

Coverage recommendations:
- **80%+**: Excellent coverage
- **60-79%**: Good coverage, consider adding more tests
- **<60%**: Low coverage, add more tests

## 🐛 Troubleshooting

### Common Issues

1. **Import Errors**: Run `go mod tidy` to resolve dependencies
2. **Permission Denied**: Make script executable: `chmod +x test/run_tests_new.sh`
3. **Test Timeouts**: Increase timeout in `run_tests_new.sh`
4. **Performance Failures**: Adjust performance thresholds in tests

### Debug Mode

Run tests with verbose output for debugging:

```bash
go test -v -timeout 60s ./test/...
```

## 📝 Best Practices

1. **Test Naming**: Use descriptive test names that explain the scenario
2. **Performance**: Always include performance measurement for critical paths
3. **Coverage**: Aim for high test coverage, especially for business logic
4. **Mocking**: Use mocks for external dependencies
5. **Isolation**: Tests should be independent and not rely on each other
6. **Documentation**: Document complex test scenarios
7. **Modular Structure**: Organize tests by module and function
8. **Consistent Naming**: Use consistent naming conventions across all test files

## 🤝 Contributing

When adding new tests:

1. Follow the existing naming conventions
2. Include performance measurement
3. Add comprehensive test cases
4. Update this README if adding new test categories
5. Ensure all tests pass before committing
6. Use the modular structure for organization

## 📞 Support

For issues with the test suite:

1. Check the troubleshooting section
2. Review test logs for specific errors
3. Verify Go version and dependencies
4. Check file permissions for shell scripts

## 🔄 Migration from Old Structure

If migrating from the old test structure:

1. **Backup**: Keep the old test files as backup
2. **Gradual Migration**: Move tests one module at a time
3. **Update Scripts**: Use the new test runner script
4. **Verify**: Ensure all tests still pass after migration

---

**Happy Testing with the New Structure! 🧪✨** 