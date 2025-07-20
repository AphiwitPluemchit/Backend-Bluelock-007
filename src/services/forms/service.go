package forms

import (
	"Backend-Bluelock-007/src/database"
	"context"
	"errors"
	"log"
	"math"
	"time"

	"Backend-Bluelock-007/src/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	formsCollection       *mongo.Collection
	questionsCollection   *mongo.Collection
	submissionsCollection *mongo.Collection
)

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	formsCollection = database.GetCollection("BluelockDB", "forms")
	questionsCollection = database.GetCollection("BluelockDB", "questions")
	submissionsCollection = database.GetCollection("BluelockDB", "submissions")

	if formsCollection == nil || questionsCollection == nil || submissionsCollection == nil {
		log.Fatal("Failed to get the required collections")
	}
}

// CreateForm creates a new form with questions
func CreateForm(ctx context.Context, req *models.CreateFormRequest) (*models.FormWithQuestions, error) {
	now := time.Now()

	// Create form
	form := &models.Form{
		Title:       req.Title,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	formResult, err := formsCollection.InsertOne(ctx, form)
	if err != nil {
		return nil, err
	}

	form.ID = formResult.InsertedID.(primitive.ObjectID)

	// Create questions
	var questions []interface{}
	for i, question := range req.Questions {
		question.ID = primitive.NewObjectID()
		question.FormID = form.ID
		question.Order = i + 1

		// Validate question based on type
		if err := validateQuestion(&question); err != nil {
			return nil, err
		}

		questions = append(questions, question)
	}

	if len(questions) > 0 {
		_, err = questionsCollection.InsertMany(ctx, questions)
		if err != nil {
			return nil, err
		}
	}

	// Convert back to slice of questions
	var createdQuestions []models.Question
	for _, q := range questions {
		createdQuestions = append(createdQuestions, q.(models.Question))
	}

	return &models.FormWithQuestions{
		Form:      *form,
		Questions: createdQuestions,
	}, nil
}

// GetForms retrieves all forms with pagination
func GetForms(ctx context.Context, page, limit int) (*models.PaginatedFormsResponse, error) {
	skip := (page - 1) * limit

	// Get total count
	total, err := formsCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	// Get forms with pagination
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := formsCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var forms []models.Form
	if err = cursor.All(ctx, &forms); err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return &models.PaginatedFormsResponse{
		Forms:      forms,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// GetFormByID retrieves a form with its questions
func GetFormByID(ctx context.Context, formID primitive.ObjectID) (*models.FormWithQuestions, error) {
	// Get form
	var form models.Form
	err := formsCollection.FindOne(ctx, bson.M{"_id": formID}).Decode(&form)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("form not found")
		}
		return nil, err
	}

	// Get questions
	opts := options.Find().SetSort(bson.D{{Key: "order", Value: 1}})
	cursor, err := questionsCollection.Find(ctx, bson.M{"formId": formID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var questions []models.Question
	if err = cursor.All(ctx, &questions); err != nil {
		return nil, err
	}

	return &models.FormWithQuestions{
		Form:      form,
		Questions: questions,
	}, nil
}

// SubmitForm submits answers to a form
func SubmitForm(ctx context.Context, formID primitive.ObjectID, req *models.SubmitFormRequest) (*models.Submission, error) {
	// Verify form exists
	var form models.Form
	err := formsCollection.FindOne(ctx, bson.M{"_id": formID}).Decode(&form)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("form not found")
		}
		return nil, err
	}

	// Get form questions for validation
	questions, err := getFormQuestions(ctx, formID)
	if err != nil {
		return nil, err
	}

	// Validate answers
	if err := validateAnswers(req.Answers, questions); err != nil {
		return nil, err
	}

	// Create submission
	submission := &models.Submission{
		FormID:      formID,
		SubmittedAt: time.Now(),
		Answers:     req.Answers,
	}

	result, err := submissionsCollection.InsertOne(ctx, submission)
	if err != nil {
		return nil, err
	}

	submission.ID = result.InsertedID.(primitive.ObjectID)
	return submission, nil
}

// GetFormSubmissions retrieves all submissions for a form
func GetFormSubmissions(ctx context.Context, formID primitive.ObjectID, page, limit int) (*models.PaginatedSubmissionsResponse, error) {
	// Verify form exists
	var form models.Form
	err := formsCollection.FindOne(ctx, bson.M{"_id": formID}).Decode(&form)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("form not found")
		}
		return nil, err
	}

	skip := (page - 1) * limit

	// Get total count
	total, err := submissionsCollection.CountDocuments(ctx, bson.M{"formId": formID})
	if err != nil {
		return nil, err
	}

	// Get submissions with pagination
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "submittedAt", Value: -1}})

	cursor, err := submissionsCollection.Find(ctx, bson.M{"formId": formID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var submissions []models.Submission
	if err = cursor.All(ctx, &submissions); err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return &models.PaginatedSubmissionsResponse{
		Submissions: submissions,
		Total:       total,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
	}, nil
}

// Helper methods

func validateQuestion(question *models.Question) error {
	switch question.Type {
	case models.MultipleChoice, models.Checkbox, models.Dropdown:
		if len(question.Choices) == 0 {
			return errors.New("choices are required for multiple choice, checkbox, and dropdown questions")
		}
	case models.GridMultipleChoice, models.GridCheckbox:
		if len(question.Rows) == 0 {
			return errors.New("rows are required for grid questions")
		}
		if len(question.Columns) == 0 {
			return errors.New("columns are required for grid questions")
		}
	}
	return nil
}

func getFormQuestions(ctx context.Context, formID primitive.ObjectID) ([]models.Question, error) {
	opts := options.Find().SetSort(bson.D{{Key: "order", Value: 1}})
	cursor, err := questionsCollection.Find(ctx, bson.M{"formId": formID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var questions []models.Question
	if err = cursor.All(ctx, &questions); err != nil {
		return nil, err
	}

	return questions, nil
}

func validateAnswers(answers []models.Answer, questions []models.Question) error {
	// Create a map of questions for quick lookup
	questionMap := make(map[primitive.ObjectID]models.Question)
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	// Create a map of answered questions
	answeredQuestions := make(map[primitive.ObjectID]bool)

	for _, answer := range answers {
		question, exists := questionMap[answer.QuestionID]
		if !exists {
			return errors.New("invalid question ID in answer")
		}

		answeredQuestions[answer.QuestionID] = true

		// Validate answer value based on question type
		if err := validateAnswerValue(answer.Value, question); err != nil {
			return err
		}
	}

	// Check if all required questions are answered
	for _, question := range questions {
		if question.IsRequired && !answeredQuestions[question.ID] {
			return errors.New("required question not answered: " + question.QuestionText)
		}
	}

	return nil
}

func validateAnswerValue(value interface{}, question models.Question) error {
	switch question.Type {
	case models.ShortAnswer, models.Paragraph:
		if str, ok := value.(string); !ok || str == "" {
			return errors.New("string value required for text questions")
		}

	case models.MultipleChoice, models.Dropdown:
		if str, ok := value.(string); !ok {
			return errors.New("string value required for single choice questions")
		} else {
			// Check if choice exists
			found := false
			for _, choice := range question.Choices {
				if choice == str {
					found = true
					break
				}
			}
			if !found {
				return errors.New("invalid choice selected")
			}
		}

	case models.Checkbox:
		if choices, ok := value.([]interface{}); !ok {
			return errors.New("array value required for checkbox questions")
		} else {
			// Check if all choices exist
			for _, choice := range choices {
				if str, ok := choice.(string); !ok {
					return errors.New("string values required in checkbox array")
				} else {
					found := false
					for _, validChoice := range question.Choices {
						if validChoice == str {
							found = true
							break
						}
					}
					if !found {
						return errors.New("invalid choice selected in checkbox")
					}
				}
			}
		}

	case models.GridMultipleChoice:
		if gridMap, ok := value.(map[string]interface{}); !ok {
			return errors.New("map value required for grid multiple choice questions")
		} else {
			// Validate each row has a valid column selection
			for row, column := range gridMap {
				if str, ok := column.(string); !ok {
					return errors.New("string value required for grid column selection")
				} else {
					// Check if row exists
					rowExists := false
					for _, validRow := range question.Rows {
						if validRow == row {
							rowExists = true
							break
						}
					}
					if !rowExists {
						return errors.New("invalid row in grid answer")
					}

					// Check if column exists
					colExists := false
					for _, validCol := range question.Columns {
						if validCol == str {
							colExists = true
							break
						}
					}
					if !colExists {
						return errors.New("invalid column selection in grid answer")
					}
				}
			}
		}

	case models.GridCheckbox:
		if gridMap, ok := value.(map[string]interface{}); !ok {
			return errors.New("map value required for grid checkbox questions")
		} else {
			// Validate each row has valid column selections
			for row, columns := range gridMap {
				if colArray, ok := columns.([]interface{}); !ok {
					return errors.New("array value required for grid checkbox column selections")
				} else {
					// Check if row exists
					rowExists := false
					for _, validRow := range question.Rows {
						if validRow == row {
							rowExists = true
							break
						}
					}
					if !rowExists {
						return errors.New("invalid row in grid checkbox answer")
					}

					// Check if all columns exist
					for _, col := range colArray {
						if str, ok := col.(string); !ok {
							return errors.New("string values required in grid checkbox array")
						} else {
							colExists := false
							for _, validCol := range question.Columns {
								if validCol == str {
									colExists = true
									break
								}
							}
							if !colExists {
								return errors.New("invalid column selection in grid checkbox answer")
							}
						}
					}
				}
			}
		}
	}

	return nil
}
