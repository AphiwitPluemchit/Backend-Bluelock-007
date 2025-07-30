package student

import (
	"testing"
	"time"

	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/test"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestStudentCreation(t *testing.T) {
	suiteResult := test.NewTestSuiteResult("Student Creation Tests")
	defer suiteResult.PrintSummary()

	// Test basic student creation
	t.Run("TestBasicStudentCreation", func(t *testing.T) {
		timer := test.NewTestTimer("Basic Student Creation")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Basic Student Creation",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Basic Student Creation", duration, 100*time.Microsecond)
		}()

		student := models.Student{
			Code:      "6400000001",
			Name:      "สมชาย ใจดี",
			EngName:   "Somchai Jaidee",
			Status:    1,
			SoftSkill: 80,
			HardSkill: 85,
			Major:     "Computer Science",
		}

		assert.NotEmpty(t, student.Code)
		assert.NotEmpty(t, student.Name)
		assert.NotEmpty(t, student.EngName)
		assert.Equal(t, 1, student.Status)
		assert.Equal(t, 80, student.SoftSkill)
		assert.Equal(t, 85, student.HardSkill)
		assert.Equal(t, "Computer Science", student.Major)
	})

	// Test student creation with ID
	t.Run("TestStudentCreationWithID", func(t *testing.T) {
		timer := test.NewTestTimer("Student Creation With ID")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Student Creation With ID",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Student Creation With ID", duration, 100*time.Microsecond)
		}()

		id := primitive.NewObjectID()
		student := models.Student{
			ID:        id,
			Code:      "6400000002",
			Name:      "สมหญิง รักดี",
			EngName:   "Somying Rakdee",
			Status:    1,
			SoftSkill: 90,
			HardSkill: 88,
			Major:     "Information Technology",
		}

		assert.Equal(t, id, student.ID)
		assert.Equal(t, "6400000002", student.Code)
		assert.Equal(t, "สมหญิง รักดี", student.Name)
		assert.Equal(t, "Somying Rakdee", student.EngName)
		assert.Equal(t, 1, student.Status)
		assert.Equal(t, 90, student.SoftSkill)
		assert.Equal(t, 88, student.HardSkill)
		assert.Equal(t, "Information Technology", student.Major)
	})

	// Test student creation with minimum required fields
	t.Run("TestStudentCreationMinimal", func(t *testing.T) {
		timer := test.NewTestTimer("Student Creation Minimal")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Student Creation Minimal",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Student Creation Minimal", duration, 100*time.Microsecond)
		}()

		student := models.Student{
			Code:   "6400000003",
			Name:   "สมศักดิ์ มั่นคง",
			Status: 1,
		}

		assert.NotEmpty(t, student.Code)
		assert.NotEmpty(t, student.Name)
		assert.Equal(t, 1, student.Status)
		assert.Empty(t, student.EngName)
		assert.Equal(t, 0, student.SoftSkill)
		assert.Equal(t, 0, student.HardSkill)
		assert.Empty(t, student.Major)
	})

	// Test student creation with high skills
	t.Run("TestStudentCreationHighSkills", func(t *testing.T) {
		timer := test.NewTestTimer("Student Creation High Skills")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Student Creation High Skills",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Student Creation High Skills", duration, 100*time.Microsecond)
		}()

		student := models.Student{
			Code:      "6400000004",
			Name:      "สมปอง เก่งกล้า",
			EngName:   "Sompong Kengka",
			Status:    1,
			SoftSkill: 95,
			HardSkill: 98,
			Major:     "Data Science",
		}

		assert.Equal(t, 95, student.SoftSkill)
		assert.Equal(t, 98, student.HardSkill)
		assert.GreaterOrEqual(t, student.SoftSkill, 90)
		assert.GreaterOrEqual(t, student.HardSkill, 90)
		assert.Equal(t, "Data Science", student.Major)
	})

	// Test student creation with zero skills
	t.Run("TestStudentCreationZeroSkills", func(t *testing.T) {
		timer := test.NewTestTimer("Student Creation Zero Skills")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Student Creation Zero Skills",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Student Creation Zero Skills", duration, 100*time.Microsecond)
		}()

		student := models.Student{
			Code:      "6400000005",
			Name:      "สมศรี ใหม่",
			EngName:   "Somsri Mai",
			Status:    1,
			SoftSkill: 0,
			HardSkill: 0,
			Major:     "General",
		}

		assert.Equal(t, 0, student.SoftSkill)
		assert.Equal(t, 0, student.HardSkill)
		assert.Equal(t, "General", student.Major)
	})
}
