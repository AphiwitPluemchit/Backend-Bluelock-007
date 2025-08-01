#!/bin/bash

# Backend Unit Test Runner with Performance Measurement
# New structure: test/module/function_test.go

echo "🚀 Starting Backend Unit Test Suite (New Structure)"
echo "=================================================="
echo ""

# Set environment variables for testing
export GO_ENV=test
export TEST_MODE=true

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${PURPLE}$1${NC}"
}

# Function to run tests for a specific module
run_module_tests() {
    local module_name=$1
    local module_path=$2
    
    print_header "Testing $module_name Module..."
    
    if [ ! -d "$module_path" ]; then
        print_warning "Module directory $module_path not found, skipping..."
        return 1
    }
    
    start_time=$(date +%s.%N)
    
    # Run all test files in the module directory
    if go test -v -timeout 30s "$module_path/..." 2>/dev/null; then
        end_time=$(date +%s.%N)
        duration=$(echo "$end_time - $start_time" | bc -l)
        print_success "$module_name module tests completed in ${duration}s"
        return 0
    else
        end_time=$(date +%s.%N)
        duration=$(echo "$end_time - $start_time" | bc -l)
        print_error "$module_name module tests failed after ${duration}s"
        return 1
    fi
}

# Function to run tests for a specific function within a module
run_function_tests() {
    local module_name=$1
    local function_name=$2
    local test_file=$3
    
    print_header "Testing $module_name - $function_name..."
    
    if [ ! -f "$test_file" ]; then
        print_warning "Test file $test_file not found, skipping..."
        return 1
    }
    
    start_time=$(date +%s.%N)
    
    # Run the specific test file
    if go test -v -timeout 30s "$test_file" 2>/dev/null; then
        end_time=$(date +%s.%N)
        duration=$(echo "$end_time - $start_time" | bc -l)
        print_success "$module_name - $function_name tests completed in ${duration}s"
        return 0
    else
        end_time=$(date +%s.%N)
        duration=$(echo "$end_time - $start_time" | bc -l)
        print_error "$module_name - $function_name tests failed after ${duration}s"
        return 1
    fi
}

# Function to run all tests
run_all_tests() {
    print_header "Running Complete Test Suite (New Structure)..."
    
    total_start_time=$(date +%s.%N)
    total_modules=0
    passed_modules=0
    failed_modules=0
    
    # Define modules and their test directories
    declare -A modules=(
        ["Student"]="./test/student"
        ["Activity"]="./test/activity"
        ["Auth"]="./test/auth"
        ["Form"]="./test/form"
        ["Course"]="./test/course"
        ["Enrollment"]="./test/enrollment"
        ["Food"]="./test/food"
        ["Admin"]="./test/admin"
    )
    
    # Run each module
    for module_name in "${!modules[@]}"; do
        module_path="${modules[$module_name]}"
        
        if run_module_tests "$module_name" "$module_path"; then
            ((passed_modules++))
        else
            ((failed_modules++))
        fi
        ((total_modules++))
        
        echo ""
    done
    
    total_end_time=$(date +%s.%N)
    total_duration=$(echo "$total_end_time - $total_start_time" | bc -l)
    
    # Print final summary
    print_header "Test Suite Summary (New Structure)"
    echo "========================================="
    echo "Total Modules: $total_modules"
    echo "Passed: $passed_modules ✅"
    echo "Failed: $failed_modules ❌"
    echo "Total Time: ${total_duration}s"
    
    if [ $failed_modules -eq 0 ]; then
        print_success "All test modules passed!"
        return 0
    else
        print_error "$failed_modules test module(s) failed!"
        return 1
    fi
}

# Function to run specific module tests
run_specific_module() {
    local module_name=$1
    
    case $module_name in
        "student")
            run_module_tests "Student" "./test/student"
            ;;
        "activity")
            run_module_tests "Activity" "./test/activity"
            ;;
        "auth")
            run_module_tests "Auth" "./test/auth"
            ;;
        "form")
            run_module_tests "Form" "./test/form"
            ;;
        "course")
            run_module_tests "Course" "./test/course"
            ;;
        "enrollment")
            run_module_tests "Enrollment" "./test/enrollment"
            ;;
        "food")
            run_module_tests "Food" "./test/food"
            ;;
        "admin")
            run_module_tests "Admin" "./test/admin"
            ;;
        *)
            print_error "Unknown module: $module_name"
            print_status "Available modules: student, activity, auth, form, course, enrollment, food, admin"
            exit 1
            ;;
    esac
}

# Function to run specific function tests
run_specific_function() {
    local module_name=$1
    local function_name=$2
    
    case $module_name in
        "student")
            case $function_name in
                "creation")
                    run_function_tests "Student" "Creation" "./test/student/student_creation_test.go"
                    ;;
                "validation")
                    run_function_tests "Student" "Validation" "./test/student/student_validation_test.go"
                    ;;
                "skills")
                    run_function_tests "Student" "Skills" "./test/student/student_skills_test.go"
                    ;;
                "status")
                    run_function_tests "Student" "Status" "./test/student/student_status_test.go"
                    ;;
                *)
                    print_error "Unknown function: $function_name"
                    print_status "Available functions: creation, validation, skills, status"
                    exit 1
                    ;;
            esac
            ;;
        "activity")
            case $function_name in
                "creation")
                    run_function_tests "Activity" "Creation" "./test/activity/activity_creation_test.go"
                    ;;
                *)
                    print_error "Unknown function: $function_name"
                    print_status "Available functions: creation"
                    exit 1
                    ;;
            esac
            ;;
        "auth")
            case $function_name in
                "login")
                    run_function_tests "Auth" "Login" "./test/auth/login_test.go"
                    ;;
                *)
                    print_error "Unknown function: $function_name"
                    print_status "Available functions: login"
                    exit 1
                    ;;
            esac
            ;;
        *)
            print_error "Unknown module: $module_name"
            print_status "Available modules: student, activity, auth"
            exit 1
            ;;
    esac
}

# Function to list available tests
list_tests() {
    print_header "Available Test Modules and Functions"
    echo "=========================================="
    echo ""
    
    echo "📁 Student Module (./test/student/)"
    echo "   ├── student_creation_test.go"
    echo "   ├── student_validation_test.go"
    echo "   └── student_skills_test.go"
    echo ""
    
    echo "📁 Activity Module (./test/activity/)"
    echo "   └── activity_creation_test.go"
    echo ""
    
    echo "📁 Auth Module (./test/auth/)"
    echo "   └── login_test.go"
    echo ""
    
    echo "📁 Other Modules"
    echo "   ├── Form Module (./test/form/)"
    echo "   ├── Course Module (./test/course/)"
    echo "   ├── Enrollment Module (./test/enrollment/)"
    echo "   ├── Food Module (./test/food/)"
    echo "   └── Admin Module (./test/admin/)"
    echo ""
    
    echo "Usage Examples:"
    echo "  $0 all                           # Run all tests"
    echo "  $0 module student                # Run all student tests"
    echo "  $0 function student creation     # Run student creation tests"
    echo "  $0 function auth login           # Run auth login tests"
    echo "  $0 list                          # List available tests"
}

# Function to show help
show_help() {
    echo "Backend Unit Test Runner (New Structure)"
    echo ""
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  all                    Run all test modules (default)"
    echo "  module <name>          Run specific module tests"
    echo "  function <module> <function>  Run specific function tests"
    echo "  list                   List available test modules and functions"
    echo "  help                   Show this help message"
    echo ""
    echo "Module Options:"
    echo "  student                Student module tests"
    echo "  activity               Activity module tests"
    echo "  auth                   Auth module tests"
    echo "  form                   Form module tests"
    echo "  course                 Course module tests"
    echo "  enrollment             Enrollment module tests"
    echo "  food                   Food module tests"
    echo "  admin                  Admin module tests"
    echo ""
    echo "Function Examples:"
    echo "  student creation       Student creation tests"
    echo "  student validation     Student validation tests"
    echo "  student skills         Student skills tests"
    echo "  activity creation      Activity creation tests"
    echo "  auth login             Auth login tests"
    echo ""
    echo "Examples:"
    echo "  $0                     # Run all tests"
    echo "  $0 module student      # Run all student tests"
    echo "  $0 function student creation  # Run student creation tests"
    echo "  $0 list                # List available tests"
}

# Main script logic
main() {
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we're in the right directory
    if [ ! -f "go.mod" ]; then
        print_error "go.mod not found. Please run this script from the project root"
        exit 1
    fi
    
    # Parse command line arguments
    case "${1:-all}" in
        "all")
            run_all_tests
            ;;
        "module")
            if [ -z "$2" ]; then
                print_error "Module name required"
                show_help
                exit 1
            fi
            run_specific_module "$2"
            ;;
        "function")
            if [ -z "$2" ] || [ -z "$3" ]; then
                print_error "Module name and function name required"
                show_help
                exit 1
            fi
            run_specific_function "$2" "$3"
            ;;
        "list")
            list_tests
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            print_error "Unknown command: $1"
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@" 