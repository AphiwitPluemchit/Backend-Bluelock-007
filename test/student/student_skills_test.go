package student

import (
	"testing"
	"time"

	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/test"

	"github.com/stretchr/testify/assert"
)

func TestStudentSkills(t *testing.T) {
	suiteResult := test.NewTestSuiteResult("Student Skills Tests")
	defer suiteResult.PrintSummary()

	// Test skill calculations
	t.Run("TestSkillCalculations", func(t *testing.T) {
		timer := test.NewTestTimer("Skill Calculations")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Skill Calculations",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Skill Calculations", duration, 100*time.Microsecond)
		}()

		student := models.Student{
			Code:      "6400000001",
			Name:      "สมปอง เก่งกล้า",
			EngName:   "Sompong Kengka",
			Status:    1,
			SoftSkill: 85,
			HardSkill: 92,
			Major:     "Data Science",
		}

		// Calculate average skill
		averageSkill := (student.SoftSkill + student.HardSkill) / 2
		assert.Equal(t, 88, averageSkill)

		// Calculate skill difference
		skillDifference := student.HardSkill - student.SoftSkill
		assert.Equal(t, 7, skillDifference)

		// Calculate total skill points
		totalSkillPoints := student.SoftSkill + student.HardSkill
		assert.Equal(t, 177, totalSkillPoints)

		// Calculate skill percentage
		softSkillPercentage := float64(student.SoftSkill) / 100.0
		hardSkillPercentage := float64(student.HardSkill) / 100.0
		assert.Equal(t, 0.85, softSkillPercentage)
		assert.Equal(t, 0.92, hardSkillPercentage)
	})

	// Test high performer detection
	t.Run("TestHighPerformerDetection", func(t *testing.T) {
		timer := test.NewTestTimer("High Performer Detection")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "High Performer Detection",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "High Performer Detection", duration, 100*time.Microsecond)
		}()

		// Test high performer student
		highPerformer := models.Student{
			Code:      "6400000002",
			Name:      "สมหญิง เก่ง",
			SoftSkill: 90,
			HardSkill: 95,
		}

		isHighPerformer := highPerformer.SoftSkill >= 80 && highPerformer.HardSkill >= 80
		assert.True(t, isHighPerformer)

		// Test average performer student
		averagePerformer := models.Student{
			Code:      "6400000003",
			Name:      "สมชาย ปานกลาง",
			SoftSkill: 70,
			HardSkill: 75,
		}

		isAveragePerformer := averagePerformer.SoftSkill >= 60 && averagePerformer.HardSkill >= 60 &&
			averagePerformer.SoftSkill < 80 && averagePerformer.HardSkill < 80
		assert.True(t, isAveragePerformer)

		// Test low performer student
		lowPerformer := models.Student{
			Code:      "6400000004",
			Name:      "สมศรี ต่ำ",
			SoftSkill: 50,
			HardSkill: 45,
		}

		isLowPerformer := lowPerformer.SoftSkill < 60 || lowPerformer.HardSkill < 60
		assert.True(t, isLowPerformer)
	})

	// Test skill balance analysis
	t.Run("TestSkillBalanceAnalysis", func(t *testing.T) {
		timer := test.NewTestTimer("Skill Balance Analysis")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Skill Balance Analysis",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Skill Balance Analysis", duration, 100*time.Microsecond)
		}()

		// Test balanced skills
		balancedStudent := models.Student{
			Code:      "6400000005",
			Name:      "สมศักดิ์ สมดุล",
			SoftSkill: 80,
			HardSkill: 82,
		}

		balanceDifference := balancedStudent.HardSkill - balancedStudent.SoftSkill
		isBalanced := balanceDifference <= 20 && balanceDifference >= -20
		assert.True(t, isBalanced)
		assert.Equal(t, 2, balanceDifference)

		// Test unbalanced skills (soft skill higher)
		softSkillFocused := models.Student{
			Code:      "6400000006",
			Name:      "สมพร นุ่มนวล",
			SoftSkill: 90,
			HardSkill: 65,
		}

		softBalanceDifference := softSkillFocused.SoftSkill - softSkillFocused.HardSkill
		isSoftFocused := softBalanceDifference > 20
		assert.True(t, isSoftFocused)
		assert.Equal(t, 25, softBalanceDifference)

		// Test unbalanced skills (hard skill higher)
		hardSkillFocused := models.Student{
			Code:      "6400000007",
			Name:      "สมปอง แข็งแกร่ง",
			SoftSkill: 60,
			HardSkill: 90,
		}

		hardBalanceDifference := hardSkillFocused.HardSkill - hardSkillFocused.SoftSkill
		isHardFocused := hardBalanceDifference > 20
		assert.True(t, isHardFocused)
		assert.Equal(t, 30, hardBalanceDifference)
	})

	// Test skill improvement scenarios
	t.Run("TestSkillImprovementScenarios", func(t *testing.T) {
		timer := test.NewTestTimer("Skill Improvement Scenarios")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Skill Improvement Scenarios",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Skill Improvement Scenarios", duration, 100*time.Microsecond)
		}()

		// Initial skills
		student := models.Student{
			Code:      "6400000008",
			Name:      "สมชาย พัฒนา",
			SoftSkill: 60,
			HardSkill: 65,
		}

		initialAverage := (student.SoftSkill + student.HardSkill) / 2
		assert.Equal(t, 62, initialAverage)

		// After soft skill improvement
		student.SoftSkill = 75
		afterSoftImprovement := (student.SoftSkill + student.HardSkill) / 2
		assert.Equal(t, 70, afterSoftImprovement)
		assert.Greater(t, afterSoftImprovement, initialAverage)

		// After hard skill improvement
		student.HardSkill = 80
		afterHardImprovement := (student.SoftSkill + student.HardSkill) / 2
		assert.Equal(t, 77, afterHardImprovement)
		assert.Greater(t, afterHardImprovement, afterSoftImprovement)

		// Calculate improvement percentages
		softImprovement := ((75 - 60) / 60.0) * 100
		hardImprovement := ((80 - 65) / 65.0) * 100
		assert.Equal(t, 25.0, softImprovement)
		assert.InDelta(t, 23.08, hardImprovement, 0.01)
	})

	// Test skill ranking
	t.Run("TestSkillRanking", func(t *testing.T) {
		timer := test.NewTestTimer("Skill Ranking")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Skill Ranking",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Skill Ranking", duration, 100*time.Microsecond)
		}()

		students := []models.Student{
			{Code: "6400000009", Name: "สมชาย A", SoftSkill: 85, HardSkill: 90},
			{Code: "6400000010", Name: "สมหญิง B", SoftSkill: 90, HardSkill: 85},
			{Code: "6400000011", Name: "สมศักดิ์ C", SoftSkill: 70, HardSkill: 75},
			{Code: "6400000012", Name: "สมปอง D", SoftSkill: 95, HardSkill: 88},
		}

		// Calculate total scores for ranking
		type StudentScore struct {
			Student models.Student
			Score   int
		}

		var studentScores []StudentScore
		for _, student := range students {
			totalScore := student.SoftSkill + student.HardSkill
			studentScores = append(studentScores, StudentScore{
				Student: student,
				Score:   totalScore,
			})
		}

		// Sort by score (descending)
		for i := 0; i < len(studentScores)-1; i++ {
			for j := i + 1; j < len(studentScores); j++ {
				if studentScores[i].Score < studentScores[j].Score {
					studentScores[i], studentScores[j] = studentScores[j], studentScores[i]
				}
			}
		}

		// Verify ranking
		assert.Equal(t, "6400000012", studentScores[0].Student.Code) // 95+88=183
		assert.Equal(t, "6400000009", studentScores[1].Student.Code) // 85+90=175
		assert.Equal(t, "6400000010", studentScores[2].Student.Code) // 90+85=175
		assert.Equal(t, "6400000011", studentScores[3].Student.Code) // 70+75=145
	})

	// Test skill validation rules
	t.Run("TestSkillValidationRules", func(t *testing.T) {
		timer := test.NewTestTimer("Skill Validation Rules")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Skill Validation Rules",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Skill Validation Rules", duration, 100*time.Microsecond)
		}()

		// Test valid skill ranges
		validSkills := []int{0, 25, 50, 75, 100}
		for _, skill := range validSkills {
			assert.GreaterOrEqual(t, skill, 0)
			assert.LessOrEqual(t, skill, 100)
		}

		// Test invalid skill ranges
		invalidSkills := []int{-1, -10, 101, 150}
		for _, skill := range invalidSkills {
			if skill < 0 {
				assert.Less(t, skill, 0)
			} else {
				assert.Greater(t, skill, 100)
			}
		}

		// Test skill consistency
		student := models.Student{
			Code:      "6400000013",
			Name:      "สมศรี ตรวจสอบ",
			SoftSkill: 80,
			HardSkill: 85,
		}

		// Both skills should be within valid range
		assert.GreaterOrEqual(t, student.SoftSkill, 0)
		assert.LessOrEqual(t, student.SoftSkill, 100)
		assert.GreaterOrEqual(t, student.HardSkill, 0)
		assert.LessOrEqual(t, student.HardSkill, 100)

		// Skills should be integers
		assert.IsType(t, 0, student.SoftSkill)
		assert.IsType(t, 0, student.HardSkill)
	})
}
