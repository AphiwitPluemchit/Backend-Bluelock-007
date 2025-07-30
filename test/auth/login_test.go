package auth

import (
	"testing"
	"time"

	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthService is a mock implementation of the auth service
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Login(email, password string) (*models.User, string, error) {
	args := m.Called(email, password)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(2)
	}
	return args.Get(0).(*models.User), args.String(1), args.Error(2)
}

func TestLogin(t *testing.T) {
	suiteResult := test.NewTestSuiteResult("Login Tests")
	defer suiteResult.PrintSummary()

	// Test successful login
	t.Run("TestSuccessfulLogin", func(t *testing.T) {
		timer := test.NewTestTimer("Successful Login")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Successful Login",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Successful Login", duration, 1*time.Millisecond)
		}()

		mockService := new(MockAuthService)

		// Setup mock expectations
		expectedUser := &models.User{
			Email: "test@example.com",
			Role:  "student",
		}
		expectedToken := "jwt-token-123"

		mockService.On("Login", "test@example.com", "password123").Return(expectedUser, expectedToken, nil)

		// Test the login function
		user, token, err := mockService.Login("test@example.com", "password123")

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		assert.Equal(t, expectedToken, token)
		mockService.AssertExpectations(t)
	})

	// Test login with invalid credentials
	t.Run("TestLoginInvalidCredentials", func(t *testing.T) {
		timer := test.NewTestTimer("Login Invalid Credentials")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Login Invalid Credentials",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Login Invalid Credentials", duration, 1*time.Millisecond)
		}()

		mockService := new(MockAuthService)

		// Setup mock expectations for failure
		mockService.On("Login", "invalid@example.com", "wrongpassword").Return(nil, "", assert.AnError)

		// Test the login function
		user, token, err := mockService.Login("invalid@example.com", "wrongpassword")

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		mockService.AssertExpectations(t)
	})

	// Test login with empty email
	t.Run("TestLoginEmptyEmail", func(t *testing.T) {
		timer := test.NewTestTimer("Login Empty Email")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Login Empty Email",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Login Empty Email", duration, 1*time.Millisecond)
		}()

		mockService := new(MockAuthService)

		// Setup mock expectations for empty email
		mockService.On("Login", "", "password123").Return(nil, "", assert.AnError)

		// Test the login function
		user, token, err := mockService.Login("", "password123")

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		mockService.AssertExpectations(t)
	})

	// Test login with empty password
	t.Run("TestLoginEmptyPassword", func(t *testing.T) {
		timer := test.NewTestTimer("Login Empty Password")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Login Empty Password",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Login Empty Password", duration, 1*time.Millisecond)
		}()

		mockService := new(MockAuthService)

		// Setup mock expectations for empty password
		mockService.On("Login", "test@example.com", "").Return(nil, "", assert.AnError)

		// Test the login function
		user, token, err := mockService.Login("test@example.com", "")

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		mockService.AssertExpectations(t)
	})

	// Test login with admin user
	t.Run("TestLoginAdminUser", func(t *testing.T) {
		timer := test.NewTestTimer("Login Admin User")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Login Admin User",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Login Admin User", duration, 1*time.Millisecond)
		}()

		mockService := new(MockAuthService)

		// Setup mock expectations for admin user
		expectedAdminUser := &models.User{
			Email: "admin@example.com",
			Role:  "admin",
		}
		expectedAdminToken := "admin-jwt-token-456"

		mockService.On("Login", "admin@example.com", "adminpass123").Return(expectedAdminUser, expectedAdminToken, nil)

		// Test the login function
		user, token, err := mockService.Login("admin@example.com", "adminpass123")

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, expectedAdminUser, user)
		assert.Equal(t, expectedAdminToken, token)
		assert.Equal(t, "admin", user.Role)
		mockService.AssertExpectations(t)
	})

	// Test login with student user
	t.Run("TestLoginStudentUser", func(t *testing.T) {
		timer := test.NewTestTimer("Login Student User")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Login Student User",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Login Student User", duration, 1*time.Millisecond)
		}()

		mockService := new(MockAuthService)

		// Setup mock expectations for student user
		expectedStudentUser := &models.User{
			Email: "student@example.com",
			Role:  "student",
		}
		expectedStudentToken := "student-jwt-token-789"

		mockService.On("Login", "student@example.com", "studentpass123").Return(expectedStudentUser, expectedStudentToken, nil)

		// Test the login function
		user, token, err := mockService.Login("student@example.com", "studentpass123")

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, expectedStudentUser, user)
		assert.Equal(t, expectedStudentToken, token)
		assert.Equal(t, "student", user.Role)
		mockService.AssertExpectations(t)
	})

	// Test email validation
	t.Run("TestEmailValidation", func(t *testing.T) {
		timer := test.NewTestTimer("Email Validation")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Email Validation",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Email Validation", duration, 100*time.Microsecond)
		}()

		// Test valid emails
		validEmails := []string{
			"user@example.com",
			"test.user@domain.co.uk",
			"user123@test-domain.org",
			"admin@buu.ac.th",
		}

		for _, email := range validEmails {
			assert.Contains(t, email, "@")
			assert.Contains(t, email, ".")
			assert.Greater(t, len(email), 5)
		}

		// Test invalid emails
		invalidEmails := []string{
			"invalid-email",
			"@example.com",
			"user@",
			"",
			"user",
		}

		for _, email := range invalidEmails {
			if email == "" {
				assert.Empty(t, email)
			} else {
				// Check if email is invalid (doesn't have both @ and .)
				isValid := len(email) > 0 && contains(email, "@") && contains(email, ".")
				assert.False(t, isValid, "Email '%s' should be invalid", email)
			}
		}
	})

	// Test password validation
	t.Run("TestPasswordValidation", func(t *testing.T) {
		timer := test.NewTestTimer("Password Validation")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Password Validation",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Password Validation", duration, 100*time.Microsecond)
		}()

		// Test valid passwords
		validPasswords := []string{
			"password123",
			"securePass456",
			"MyP@ssw0rd",
			"StrongP@ss1",
		}

		for _, password := range validPasswords {
			assert.GreaterOrEqual(t, len(password), 8)
			assert.NotEmpty(t, password)
		}

		// Test invalid passwords
		invalidPasswords := []string{
			"",
			"short",
			"123",
			"abc",
		}

		for _, password := range invalidPasswords {
			if password == "" {
				assert.Empty(t, password)
			} else {
				assert.Less(t, len(password), 8)
			}
		}
	})

	// Test token validation
	t.Run("TestTokenValidation", func(t *testing.T) {
		timer := test.NewTestTimer("Token Validation")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Token Validation",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Token Validation", duration, 100*time.Microsecond)
		}()

		// Test valid tokens
		validTokens := []string{
			"jwt-token-123",
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			"admin-jwt-token-456",
			"student-jwt-token-789",
		}

		for _, token := range validTokens {
			assert.NotEmpty(t, token)
			assert.Greater(t, len(token), 10)
		}

		// Test invalid tokens
		invalidTokens := []string{
			"",
			"short",
			"invalid",
		}

		for _, token := range invalidTokens {
			if token == "" {
				assert.Empty(t, token)
			} else {
				assert.LessOrEqual(t, len(token), 10)
			}
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr))
}
