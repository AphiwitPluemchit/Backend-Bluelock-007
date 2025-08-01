package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// QuestionType represents the type of question
type QuestionType string

const (
	ShortAnswer        QuestionType = "short_answer"
	Paragraph          QuestionType = "paragraph"
	MultipleChoice     QuestionType = "multiple_choice"
	Checkbox           QuestionType = "checkbox"
	Dropdown           QuestionType = "dropdown"
	GridMultipleChoice QuestionType = "grid_multiple_choice"
	GridCheckbox       QuestionType = "grid_checkbox"
)

// Form represents a form with its metadata
type Form struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Title       string             `json:"title" bson:"title" validate:"required"`
	Description string             `json:"description" bson:"description"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// Question represents a question within a form
type Question struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	FormID       primitive.ObjectID `json:"formId" bson:"formId" validate:"required"`
	Type         QuestionType       `json:"type" bson:"type" validate:"required"`
	QuestionText string             `json:"questionText" bson:"questionText" validate:"required"`
	IsRequired   bool               `json:"isRequired" bson:"isRequired"`
	Choices      []string           `json:"choices,omitempty" bson:"choices,omitempty"`
	Rows         []string           `json:"rows,omitempty" bson:"rows,omitempty"`
	Columns      []string           `json:"columns,omitempty" bson:"columns,omitempty"`
	Order        int                `json:"order" bson:"order"`
}

// Answer represents an answer to a specific question
type Answer struct {
	QuestionID primitive.ObjectID `json:"questionId" bson:"questionId" validate:"required"`
	Value      interface{}        `json:"value" bson:"value" validate:"required"`
}

// Submission represents a form submission with answers
type Submission struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	FormID      primitive.ObjectID `json:"formId" bson:"formId" validate:"required"`
	SubmittedAt time.Time          `json:"submittedAt" bson:"submittedAt"`
	Answers     []Answer           `json:"answers" bson:"answers" validate:"required"`
}

// FormWithQuestions represents a form with its questions
type FormWithQuestions struct {
	Form      Form       `json:"form"`
	Questions []Question `json:"questions"`
}

// CreateFormRequest represents the request to create a form
type CreateFormRequest struct {
	Title       string     `json:"title" validate:"required"`
	Description string     `json:"description"`
	Questions   []Question `json:"questions" validate:"required,min=1"`
}

// SubmitFormRequest represents the request to submit a form
type SubmitFormRequest struct {
	Answers []Answer `json:"answers" validate:"required,min=1"`
}

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Page  int `json:"page" query:"page" validate:"min=1"`
	Limit int `json:"limit" query:"limit" validate:"min=1,max=100"`
}

// PaginatedFormsResponse represents paginated forms response
type PaginatedFormsResponse struct {
	Forms      []Form `json:"forms"`
	Total      int64  `json:"total"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	TotalPages int    `json:"totalPages"`
}

// PaginatedSubmissionsResponse represents paginated submissions response
type PaginatedSubmissionsResponse struct {
	Submissions []Submission `json:"submissions"`
	Total       int64        `json:"total"`
	Page        int          `json:"page"`
	Limit       int          `json:"limit"`
	TotalPages  int          `json:"totalPages"`
}
