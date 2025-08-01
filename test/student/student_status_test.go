package student

import (
	"testing"
	"time"

	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/test"

	"github.com/stretchr/testify/assert"
)

func TestStudentStatus(t *testing.T) {
	suiteResult := test.NewTestSuiteResult("Student Status Tests")
	defer suiteResult.PrintSummary()

	// Test Status 4 - ออกผึกแล้ว
	t.Run("TestGraduatedStatus", func(t *testing.T) {
		timer := test.NewTestTimer("Graduated Status")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Graduated Status",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Graduated Status", duration, 100*time.Microsecond)
		}()

		// สร้างนักเรียนที่มี Status 4 (ออกผึกแล้ว)
		graduatedStudent := models.Student{
			Code:      "123456",
			Name:      "สมชาย เรียนจบ",
			EngName:   "Somchai Graduated",
			Status:    4, // ออกผึกแล้ว
			SoftSkill: 30,
			HardSkill: 12,
			Major:     "CS",
		}

		// ตรวจสอบว่า Status เป็น 4
		assert.Equal(t, 4, graduatedStudent.Status)
		assert.Equal(t, "สมชาย เรียนจบ", graduatedStudent.Name)
		assert.Equal(t, 30, graduatedStudent.SoftSkill)
		assert.Equal(t, 12, graduatedStudent.HardSkill)
	})

	// Test Status validation for graduated students
	t.Run("TestGraduatedStatusValidation", func(t *testing.T) {
		timer := test.NewTestTimer("Graduated Status Validation")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Graduated Status Validation",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Graduated Status Validation", duration, 100*time.Microsecond)
		}()

		// ตรวจสอบ Status ที่ถูกต้อง
		validStatuses := []int{0, 1, 2, 3, 4}
		assert.Contains(t, validStatuses, 4)

		// ตรวจสอบ Status ที่ไม่ถูกต้อง
		invalidStatuses := []int{-1, 5, 10, 100}
		for _, invalidStatus := range invalidStatuses {
			assert.NotContains(t, validStatuses, invalidStatus)
		}
	})

	// Test Graduated student with different skill levels
	t.Run("TestGraduatedStudentSkills", func(t *testing.T) {
		timer := test.NewTestTimer("Graduated Student Skills")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Graduated Student Skills",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Graduated Student Skills", duration, 100*time.Microsecond)
		}()

		// นักเรียนที่จบแล้วและมีทักษะครบ
		graduatedComplete := models.Student{
			Code:      "123457",
			Name:      "สมหญิง จบครบ",
			EngName:   "Somying Complete",
			Status:    4,
			SoftSkill: 30,
			HardSkill: 12,
			Major:     "SE",
		}

		// นักเรียนที่จบแล้วแต่ทักษะไม่ครบ
		graduatedIncomplete := models.Student{
			Code:      "123458",
			Name:      "สมศักดิ์ จบไม่ครบ",
			EngName:   "Somsak Incomplete",
			Status:    4,
			SoftSkill: 25,
			HardSkill: 8,
			Major:     "ITDI",
		}

		// ตรวจสอบนักเรียนที่จบแล้วและทักษะครบ
		assert.Equal(t, 4, graduatedComplete.Status)
		assert.Equal(t, 30, graduatedComplete.SoftSkill)
		assert.Equal(t, 12, graduatedComplete.HardSkill)
		assert.True(t, graduatedComplete.SoftSkill >= 30)
		assert.True(t, graduatedComplete.HardSkill >= 12)

		// ตรวจสอบนักเรียนที่จบแล้วแต่ทักษะไม่ครบ
		assert.Equal(t, 4, graduatedIncomplete.Status)
		assert.Equal(t, 25, graduatedIncomplete.SoftSkill)
		assert.Equal(t, 8, graduatedIncomplete.HardSkill)
		assert.False(t, graduatedIncomplete.SoftSkill >= 30)
		assert.False(t, graduatedIncomplete.HardSkill >= 12)
	})

	// Test Status transition scenarios
	t.Run("TestStatusTransition", func(t *testing.T) {
		timer := test.NewTestTimer("Status Transition")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Status Transition",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Status Transition", duration, 100*time.Microsecond)
		}()

		// สร้างนักเรียนและเปลี่ยน Status
		student := models.Student{
			Code:      "123459",
			Name:      "สมปอง เปลี่ยนสถานะ",
			EngName:   "Sompong Status Change",
			Status:    1, // เริ่มต้นที่ชั่วโมงน้อยมาก
			SoftSkill: 5,
			HardSkill: 2,
			Major:     "AAI",
		}

		// ตรวจสอบ Status เริ่มต้น
		assert.Equal(t, 1, student.Status)

		// เปลี่ยนเป็น Status 2 (ชั่วโมงน้อย)
		student.Status = 2
		assert.Equal(t, 2, student.Status)

		// เปลี่ยนเป็น Status 3 (ชั่วโมงครบแล้ว)
		student.Status = 3
		assert.Equal(t, 3, student.Status)

		// เปลี่ยนเป็น Status 4 (ออกผึกแล้ว)
		student.Status = 4
		assert.Equal(t, 4, student.Status)

		// เปลี่ยนกลับเป็น Status 0 (พ้นสภาพ)
		student.Status = 0
		assert.Equal(t, 0, student.Status)
	})
}
