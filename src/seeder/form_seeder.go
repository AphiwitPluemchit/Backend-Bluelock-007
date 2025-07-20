package seeder

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/forms"
	"context"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
)

// SeedSampleForms creates sample forms for testing
func SeedSampleForms(db *mongo.Database) error {
	ctx := context.Background()

	// Sample Form 1: Student Feedback Form
	feedbackForm := &models.CreateFormRequest{
		Title:       "Student Feedback Form",
		Description: "Please provide your feedback about the course and instructor",
		Questions: []models.Question{
			{
				Type:         models.ShortAnswer,
				QuestionText: "What is your student ID?",
				IsRequired:   true,
			},
			{
				Type:         models.Paragraph,
				QuestionText: "Please describe your overall experience with this course.",
				IsRequired:   true,
			},
			{
				Type:         models.MultipleChoice,
				QuestionText: "How would you rate the course difficulty?",
				IsRequired:   true,
				Choices:      []string{"Very Easy", "Easy", "Moderate", "Difficult", "Very Difficult"},
			},
			{
				Type:         models.Checkbox,
				QuestionText: "Which aspects of the course did you find most helpful? (Select all that apply)",
				IsRequired:   false,
				Choices:      []string{"Lectures", "Assignments", "Group Projects", "Office Hours", "Online Resources", "Textbook"},
			},
			{
				Type:         models.Dropdown,
				QuestionText: "What is your major?",
				IsRequired:   true,
				Choices:      []string{"Computer Science", "Engineering", "Business", "Arts", "Science", "Other"},
			},
			{
				Type:         models.GridMultipleChoice,
				QuestionText: "Rate the following aspects of the course:",
				IsRequired:   true,
				Rows:         []string{"Course Content", "Instructor", "Assignments", "Grading", "Course Materials"},
				Columns:      []string{"Poor", "Fair", "Good", "Very Good", "Excellent"},
			},
			{
				Type:         models.GridCheckbox,
				QuestionText: "Which learning activities did you participate in? (Select all that apply for each)",
				IsRequired:   false,
				Rows:         []string{"Class Discussions", "Group Projects", "Individual Assignments", "Presentations", "Research Papers"},
				Columns:      []string{"Regularly", "Sometimes", "Rarely", "Never"},
			},
		},
	}

	// Sample Form 2: Event Registration Form
	eventForm := &models.CreateFormRequest{
		Title:       "Tech Conference Registration",
		Description: "Register for the annual technology conference",
		Questions: []models.Question{
			{
				Type:         models.ShortAnswer,
				QuestionText: "Full Name",
				IsRequired:   true,
			},
			{
				Type:         models.ShortAnswer,
				QuestionText: "Email Address",
				IsRequired:   true,
			},
			{
				Type:         models.ShortAnswer,
				QuestionText: "Company/Organization",
				IsRequired:   false,
			},
			{
				Type:         models.MultipleChoice,
				QuestionText: "What is your primary role?",
				IsRequired:   true,
				Choices:      []string{"Developer", "Designer", "Manager", "Student", "Other"},
			},
			{
				Type:         models.Checkbox,
				QuestionText: "Which sessions are you interested in attending?",
				IsRequired:   true,
				Choices:      []string{"AI/ML", "Web Development", "Mobile Development", "DevOps", "UI/UX Design", "Data Science"},
			},
			{
				Type:         models.Dropdown,
				QuestionText: "How did you hear about this conference?",
				IsRequired:   false,
				Choices:      []string{"Social Media", "Email Newsletter", "Friend/Colleague", "Website", "Advertisement", "Other"},
			},
			{
				Type:         models.Paragraph,
				QuestionText: "Any special dietary requirements or accessibility needs?",
				IsRequired:   false,
			},
		},
	}

	// Create the forms
	formsList := []*models.CreateFormRequest{feedbackForm, eventForm}

	for _, form := range formsList {
		result, err := forms.CreateForm(ctx, form)
		if err != nil {
			log.Printf("Error creating form '%s': %v", form.Title, err)
			continue
		}
		log.Printf("✅ Created form: %s (ID: %s)", result.Form.Title, result.Form.ID.Hex())
	}

	return nil
}

// SeedSampleSubmissions creates sample submissions for testing
func SeedSampleSubmissions(db *mongo.Database) error {
	ctx := context.Background()

	// Get the first form to create submissions for
	formsList, err := forms.GetForms(ctx, 1, 1)
	if err != nil || len(formsList.Forms) == 0 {
		log.Println("No forms found to create submissions for")
		return nil
	}

	formID := formsList.Forms[0].ID
	formWithQuestions, err := forms.GetFormByID(ctx, formID)
	if err != nil {
		log.Printf("Error getting form questions: %v", err)
		return err
	}

	// Create sample submissions
	submissions := []*models.SubmitFormRequest{
		{
			Answers: []models.Answer{
				{
					QuestionID: formWithQuestions.Questions[0].ID, // Student ID
					Value:      "12345",
				},
				{
					QuestionID: formWithQuestions.Questions[1].ID, // Overall experience
					Value:      "The course was very informative and well-structured. I learned a lot about the subject matter.",
				},
				{
					QuestionID: formWithQuestions.Questions[2].ID, // Course difficulty
					Value:      "Moderate",
				},
				{
					QuestionID: formWithQuestions.Questions[3].ID, // Helpful aspects
					Value:      []interface{}{"Lectures", "Assignments", "Online Resources"},
				},
				{
					QuestionID: formWithQuestions.Questions[4].ID, // Major
					Value:      "Computer Science",
				},
				{
					QuestionID: formWithQuestions.Questions[5].ID, // Grid rating
					Value: map[string]interface{}{
						"Course Content":   "Very Good",
						"Instructor":       "Excellent",
						"Assignments":      "Good",
						"Grading":          "Fair",
						"Course Materials": "Very Good",
					},
				},
				{
					QuestionID: formWithQuestions.Questions[6].ID, // Grid checkbox
					Value: map[string]interface{}{
						"Class Discussions":      []interface{}{"Regularly", "Sometimes"},
						"Group Projects":         []interface{}{"Sometimes"},
						"Individual Assignments": []interface{}{"Regularly"},
						"Presentations":          []interface{}{"Rarely"},
						"Research Papers":        []interface{}{"Never"},
					},
				},
			},
		},
		{
			Answers: []models.Answer{
				{
					QuestionID: formWithQuestions.Questions[0].ID,
					Value:      "67890",
				},
				{
					QuestionID: formWithQuestions.Questions[1].ID,
					Value:      "The course was challenging but rewarding. The instructor was very knowledgeable.",
				},
				{
					QuestionID: formWithQuestions.Questions[2].ID,
					Value:      "Difficult",
				},
				{
					QuestionID: formWithQuestions.Questions[3].ID,
					Value:      []interface{}{"Group Projects", "Office Hours", "Textbook"},
				},
				{
					QuestionID: formWithQuestions.Questions[4].ID,
					Value:      "Engineering",
				},
				{
					QuestionID: formWithQuestions.Questions[5].ID,
					Value: map[string]interface{}{
						"Course Content":   "Good",
						"Instructor":       "Very Good",
						"Assignments":      "Excellent",
						"Grading":          "Good",
						"Course Materials": "Fair",
					},
				},
				{
					QuestionID: formWithQuestions.Questions[6].ID,
					Value: map[string]interface{}{
						"Class Discussions":      []interface{}{"Sometimes"},
						"Group Projects":         []interface{}{"Regularly", "Sometimes"},
						"Individual Assignments": []interface{}{"Regularly"},
						"Presentations":          []interface{}{"Sometimes"},
						"Research Papers":        []interface{}{"Rarely"},
					},
				},
			},
		},
	}

	for i, submission := range submissions {
		result, err := forms.SubmitForm(ctx, formID, submission)
		if err != nil {
			log.Printf("Error creating submission %d: %v", i+1, err)
			continue
		}
		log.Printf("✅ Created submission %d (ID: %s)", i+1, result.ID.Hex())
	}

	return nil
}
