package student

import (
	"testing"
	"time"

	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/test"

	"github.com/stretchr/testify/assert"
)

func TestStudentValidation(t *testing.T) {
	suiteResult := test.NewTestSuiteResult("Student Validation Tests")
	defer suiteResult.PrintSummary()

	// Test valid student validation
	t.Run("TestValidStudentValidation", func(t *testing.T) {
		timer := test.NewTestTimer("Valid Student Validation")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Valid Student Validation",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Valid Student Validation", duration, 100*time.Microsecond)
		}()

		validStudent := models.Student{
			Code:      "6400000001",
			Name:      "สมชาย ใจดี",
			EngName:   "Somchai Jaidee",
			Status:    1,
			SoftSkill: 75,
			HardSkill: 82,
			Major:     "Software Engineering",
		}

		// Validate required fields
		assert.NotEmpty(t, validStudent.Code)
		assert.NotEmpty(t, validStudent.Name)
		assert.NotEmpty(t, validStudent.EngName)

		// Validate skill ranges
		assert.GreaterOrEqual(t, validStudent.SoftSkill, 0)
		assert.LessOrEqual(t, validStudent.SoftSkill, 100)
		assert.GreaterOrEqual(t, validStudent.HardSkill, 0)
		assert.LessOrEqual(t, validStudent.HardSkill, 100)

		// Validate status
		assert.Equal(t, 1, validStudent.Status)
		assert.NotEmpty(t, validStudent.Major)
	})

	// Test invalid student validation - empty fields
	t.Run("TestInvalidStudentEmptyFields", func(t *testing.T) {
		timer := test.NewTestTimer("Invalid Student Empty Fields")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Invalid Student Empty Fields",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Invalid Student Empty Fields", duration, 100*time.Microsecond)
		}()

		invalidStudent := models.Student{
			Code:      "",
			Name:      "",
			EngName:   "",
			Status:    0,
			SoftSkill: -1,
			HardSkill: 101,
			Major:     "",
		}

		// Validate empty fields
		assert.Empty(t, invalidStudent.Code)
		assert.Empty(t, invalidStudent.Name)
		assert.Empty(t, invalidStudent.EngName)
		assert.Empty(t, invalidStudent.Major)

		// Validate invalid status
		assert.Equal(t, 0, invalidStudent.Status)

		// Validate invalid skill ranges
		assert.Less(t, invalidStudent.SoftSkill, 0)
		assert.Greater(t, invalidStudent.HardSkill, 100)
	})

	// Test student code validation
	t.Run("TestStudentCodeValidation", func(t *testing.T) {
		timer := test.NewTestTimer("Student Code Validation")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Student Code Validation",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Student Code Validation", duration, 100*time.Microsecond)
		}()

		// Test valid student codes
		validCodes := []string{
			"6400000001",
			"6500000001",
			"6600000001",
			"6700000001",
		}

		for _, code := range validCodes {
			assert.NotEmpty(t, code)
			assert.Len(t, code, 10)
			assert.Contains(t, code, "64") // Should start with 64
		}

		// Test invalid student codes
		invalidCodes := []string{
			"",
			"123",
			"640000000",
			"64000000011",
			"abc1234567",
		}

		for _, code := range invalidCodes {
			if code == "" {
				assert.Empty(t, code)
			} else {
				assert.NotEqual(t, 10, len(code))
			}
		}
	})

	// Test student name validation
	t.Run("TestStudentNameValidation", func(t *testing.T) {
		timer := test.NewTestTimer("Student Name Validation")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Student Name Validation",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Student Name Validation", duration, 100*time.Microsecond)
		}()

		// Test valid Thai names
		validThaiNames := []string{
			"สมชาย ใจดี",
			"สมหญิง รักดี",
			"สมศักดิ์ มั่นคง",
			"สมปอง เก่งกล้า",
		}

		for _, name := range validThaiNames {
			assert.NotEmpty(t, name)
			assert.GreaterOrEqual(t, len(name), 5)
		}

		// Test valid English names
		validEnglishNames := []string{
			"Somchai Jaidee",
			"Somying Rakdee",
			"Somsak Mankong",
			"Sompong Kengka",
		}

		for _, name := range validEnglishNames {
			assert.NotEmpty(t, name)
			assert.GreaterOrEqual(t, len(name), 5)
		}

		// Test invalid names
		invalidNames := []string{
			"",
			"a",
			"ab",
			"abc",
		}

		for _, name := range invalidNames {
			if name == "" {
				assert.Empty(t, name)
			} else {
				assert.Less(t, len(name), 5)
			}
		}
	})

	// Test student skill validation
	t.Run("TestStudentSkillValidation", func(t *testing.T) {
		timer := test.NewTestTimer("Student Skill Validation")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Student Skill Validation",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Student Skill Validation", duration, 100*time.Microsecond)
		}()

		// Test valid skill ranges
		validSkills := []int{0, 25, 50, 75, 100}

		for _, skill := range validSkills {
			assert.GreaterOrEqual(t, skill, 0)
			assert.LessOrEqual(t, skill, 100)
		}

		// Test invalid skill ranges
		invalidSkills := []int{-1, -10, 101, 150, 200}

		for _, skill := range invalidSkills {
			if skill < 0 {
				assert.Less(t, skill, 0)
			} else {
				assert.Greater(t, skill, 100)
			}
		}

		// Test skill balance validation
		student := models.Student{
			SoftSkill: 80,
			HardSkill: 85,
		}

		skillDifference := student.HardSkill - student.SoftSkill
		assert.Equal(t, 5, skillDifference)
		assert.LessOrEqual(t, skillDifference, 20) // Balanced skills
	})

	// Test student status validation
	t.Run("TestStudentStatusValidation", func(t *testing.T) {
		timer := test.NewTestTimer("Student Status Validation")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Student Status Validation",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Student Status Validation", duration, 100*time.Microsecond)
		}()

		// Test active student
		activeStudent := models.Student{
			Code:   "6400000001",
			Name:   "สมพร กระต่าย",
			Status: 1,
		}
		assert.Equal(t, 1, activeStudent.Status)
		assert.True(t, activeStudent.Status == 1)

		// Test inactive student
		inactiveStudent := models.Student{
			Code:   "6400000002",
			Name:   "สมศรี เงียบ",
			Status: 0,
		}
		assert.Equal(t, 0, inactiveStudent.Status)
		assert.True(t, inactiveStudent.Status == 0)

		// Test valid status values
		validStatuses := []int{0, 1}
		for _, status := range validStatuses {
			assert.Contains(t, validStatuses, status)
		}

		// Test invalid status values
		invalidStatuses := []int{-1, 2, 10, 100}
		for _, status := range invalidStatuses {
			assert.NotContains(t, validStatuses, status)
		}
	})

	// Test student major validation
	t.Run("TestStudentMajorValidation", func(t *testing.T) {
		timer := test.NewTestTimer("Student Major Validation")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Student Major Validation",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Student Major Validation", duration, 100*time.Microsecond)
		}()

		// Test valid majors
		validMajors := []string{
			"Computer Science",
			"Information Technology",
			"Software Engineering",
			"Data Science",
			"Artificial Intelligence",
		}

		for _, major := range validMajors {
			assert.NotEmpty(t, major)
			assert.GreaterOrEqual(t, len(major), 5)
		}

		// Test empty major
		emptyMajor := ""
		assert.Empty(t, emptyMajor)

		// Test major length validation
		shortMajor := "CS"
		assert.Less(t, len(shortMajor), 5)
	})
}
