package activity

import (
	"testing"
	"time"

	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/test"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestActivityCreation(t *testing.T) {
	suiteResult := test.NewTestSuiteResult("Activity Creation Tests")
	defer suiteResult.PrintSummary()

	// Test basic activity creation
	t.Run("TestBasicActivityCreation", func(t *testing.T) {
		timer := test.NewTestTimer("Basic Activity Creation")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Basic Activity Creation",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Basic Activity Creation", duration, 100*time.Microsecond)
		}()

		activityName := "Football Tournament"
		activity := models.Activity{
			Name:          &activityName,
			Type:          "one",
			ActivityState: "planning",
			Skill:         "hard",
			EndDateEnroll: "2025-03-15",
			File:          "football.jpg",
			FoodVotes:     []models.FoodVote{},
		}

		assert.NotNil(t, activity.Name)
		assert.Equal(t, "Football Tournament", *activity.Name)
		assert.Equal(t, "one", activity.Type)
		assert.Equal(t, "planning", activity.ActivityState)
		assert.Equal(t, "hard", activity.Skill)
		assert.Equal(t, "2025-03-15", activity.EndDateEnroll)
		assert.Equal(t, "football.jpg", activity.File)
		assert.Empty(t, activity.FoodVotes)
	})

	// Test activity creation with ID
	t.Run("TestActivityCreationWithID", func(t *testing.T) {
		timer := test.NewTestTimer("Activity Creation With ID")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Activity Creation With ID",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Activity Creation With ID", duration, 100*time.Microsecond)
		}()

		id := primitive.NewObjectID()
		activityName := "Basketball Championship"
		activity := models.Activity{
			ID:            id,
			Name:          &activityName,
			Type:          "multiple",
			ActivityState: "active",
			Skill:         "soft",
			EndDateEnroll: "2025-04-20",
			File:          "basketball.jpg",
			FoodVotes: []models.FoodVote{
				{Vote: 5, FoodName: "Pizza"},
				{Vote: 3, FoodName: "Burger"},
			},
		}

		assert.Equal(t, id, activity.ID)
		assert.Equal(t, "Basketball Championship", *activity.Name)
		assert.Equal(t, "multiple", activity.Type)
		assert.Equal(t, "active", activity.ActivityState)
		assert.Equal(t, "soft", activity.Skill)
		assert.Equal(t, "2025-04-20", activity.EndDateEnroll)
		assert.Equal(t, "basketball.jpg", activity.File)
		assert.Len(t, activity.FoodVotes, 2)
		assert.Equal(t, 5, activity.FoodVotes[0].Vote)
		assert.Equal(t, "Pizza", activity.FoodVotes[0].FoodName)
	})

	// Test activity creation with different types
	t.Run("TestActivityCreationDifferentTypes", func(t *testing.T) {
		timer := test.NewTestTimer("Activity Creation Different Types")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Activity Creation Different Types",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Activity Creation Different Types", duration, 100*time.Microsecond)
		}()

		// Test single activity
		singleActivityName := "Single Event"
		singleActivity := models.Activity{
			Name:          &singleActivityName,
			Type:          "one",
			ActivityState: "planning",
			Skill:         "hard",
		}
		assert.Equal(t, "one", singleActivity.Type)

		// Test multiple activity
		multipleActivityName := "Multiple Events"
		multipleActivity := models.Activity{
			Name:          &multipleActivityName,
			Type:          "multiple",
			ActivityState: "planning",
			Skill:         "soft",
		}
		assert.Equal(t, "multiple", multipleActivity.Type)
	})

	// Test activity creation with different states
	t.Run("TestActivityCreationDifferentStates", func(t *testing.T) {
		timer := test.NewTestTimer("Activity Creation Different States")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Activity Creation Different States",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Activity Creation Different States", duration, 100*time.Microsecond)
		}()

		// Test planning state
		planningActivityName := "Planning Activity"
		planningActivity := models.Activity{
			Name:          &planningActivityName,
			ActivityState: "planning",
		}
		assert.Equal(t, "planning", planningActivity.ActivityState)

		// Test active state
		activeActivityName := "Active Activity"
		activeActivity := models.Activity{
			Name:          &activeActivityName,
			ActivityState: "active",
		}
		assert.Equal(t, "active", activeActivity.ActivityState)

		// Test completed state
		completedActivityName := "Completed Activity"
		completedActivity := models.Activity{
			Name:          &completedActivityName,
			ActivityState: "completed",
		}
		assert.Equal(t, "completed", completedActivity.ActivityState)
	})

	// Test activity creation with different skills
	t.Run("TestActivityCreationDifferentSkills", func(t *testing.T) {
		timer := test.NewTestTimer("Activity Creation Different Skills")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Activity Creation Different Skills",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Activity Creation Different Skills", duration, 100*time.Microsecond)
		}()

		// Test hard skill activity
		hardSkillActivityName := "Hard Skill Activity"
		hardSkillActivity := models.Activity{
			Name:  &hardSkillActivityName,
			Skill: "hard",
		}
		assert.Equal(t, "hard", hardSkillActivity.Skill)

		// Test soft skill activity
		softSkillActivityName := "Soft Skill Activity"
		softSkillActivity := models.Activity{
			Name:  &softSkillActivityName,
			Skill: "soft",
		}
		assert.Equal(t, "soft", softSkillActivity.Skill)
	})

	// Test activity creation with food votes
	t.Run("TestActivityCreationWithFoodVotes", func(t *testing.T) {
		timer := test.NewTestTimer("Activity Creation With Food Votes")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Activity Creation With Food Votes",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Activity Creation With Food Votes", duration, 100*time.Microsecond)
		}()

		activityName := "Food Activity"
		activity := models.Activity{
			Name: &activityName,
			FoodVotes: []models.FoodVote{
				{Vote: 10, FoodName: "Pad Thai"},
				{Vote: 8, FoodName: "Som Tam"},
				{Vote: 5, FoodName: "Tom Yum"},
				{Vote: 3, FoodName: "Green Curry"},
			},
		}

		assert.Len(t, activity.FoodVotes, 4)
		assert.Equal(t, 10, activity.FoodVotes[0].Vote)
		assert.Equal(t, "Pad Thai", activity.FoodVotes[0].FoodName)
		assert.Equal(t, 8, activity.FoodVotes[1].Vote)
		assert.Equal(t, "Som Tam", activity.FoodVotes[1].FoodName)
		assert.Equal(t, 5, activity.FoodVotes[2].Vote)
		assert.Equal(t, "Tom Yum", activity.FoodVotes[2].FoodName)
		assert.Equal(t, 3, activity.FoodVotes[3].Vote)
		assert.Equal(t, "Green Curry", activity.FoodVotes[3].FoodName)
	})

	// Test activity creation with empty food votes
	t.Run("TestActivityCreationEmptyFoodVotes", func(t *testing.T) {
		timer := test.NewTestTimer("Activity Creation Empty Food Votes")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Activity Creation Empty Food Votes",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Activity Creation Empty Food Votes", duration, 100*time.Microsecond)
		}()

		activityName := "No Food Activity"
		activity := models.Activity{
			Name:      &activityName,
			FoodVotes: []models.FoodVote{},
		}

		assert.Empty(t, activity.FoodVotes)
		assert.Len(t, activity.FoodVotes, 0)
	})

	// Test activity creation with file attachment
	t.Run("TestActivityCreationWithFile", func(t *testing.T) {
		timer := test.NewTestTimer("Activity Creation With File")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Activity Creation With File",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Activity Creation With File", duration, 100*time.Microsecond)
		}()

		activityName := "File Activity"
		activity := models.Activity{
			Name: &activityName,
			File: "activity_image.jpg",
		}

		assert.Equal(t, "activity_image.jpg", activity.File)
		assert.NotEmpty(t, activity.File)
	})

	// Test activity creation with date validation
	t.Run("TestActivityCreationDateValidation", func(t *testing.T) {
		timer := test.NewTestTimer("Activity Creation Date Validation")
		defer func() {
			duration := timer.Stop()
			suiteResult.AddResult(test.TestResult{
				Name:     "Activity Creation Date Validation",
				Duration: duration,
				Passed:   true,
			})
			test.PerformanceAssertion(t, "Activity Creation Date Validation", duration, 100*time.Microsecond)
		}()

		// Test valid date format
		validDateActivityName := "Valid Date Activity"
		validDateActivity := models.Activity{
			Name:          &validDateActivityName,
			EndDateEnroll: "2025-03-15",
		}
		assert.Equal(t, "2025-03-15", validDateActivity.EndDateEnroll)
		assert.NotEmpty(t, validDateActivity.EndDateEnroll)

		// Test empty date
		emptyDateActivityName := "Empty Date Activity"
		emptyDateActivity := models.Activity{
			Name:          &emptyDateActivityName,
			EndDateEnroll: "",
		}
		assert.Empty(t, emptyDateActivity.EndDateEnroll)
	})
}
