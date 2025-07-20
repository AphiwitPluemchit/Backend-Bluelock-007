# Forms API Documentation

This document describes the RESTful API for the Google Forms-like system built with Go, Fiber, and MongoDB.

## Features

- Create forms with various question types
- Submit answers to forms
- View form submissions with pagination
- Support for grid question types
- Comprehensive validation and error handling

## Question Types

1. **short_answer** - Single line text input
2. **paragraph** - Multi-line text input
3. **multiple_choice** - Single selection from choices
4. **checkbox** - Multiple selections from choices
5. **dropdown** - Single selection from dropdown
6. **grid_multiple_choice** - Grid where each row allows one column selection
7. **grid_checkbox** - Grid where each row allows multiple column selections

## Architecture

The Forms API follows the same architectural pattern as other services in this project:

### Service Layer (`src/services/forms/service.go`)
- Uses `init()` function to initialize MongoDB collections
- Standalone functions (not methods on a struct)
- Business logic and data validation
- Database operations

### Controller Layer (`src/controllers/form_controller.go`)
- Standalone functions (not methods on a struct)
- HTTP request/response handling
- Input validation and error handling
- Calls service functions directly

### Routes (`src/routes/form_routes.go`)
- Route definitions following project pattern
- Grouped under `/forms` endpoint

## API Endpoints

### Base URL
```
http://localhost:8888/api
```

### 1. Create Form
**POST** `/forms`

Creates a new form with questions in a single API call.

**Request Body:**
```json
{
  "title": "Student Feedback Form",
  "description": "Please provide your feedback about the course",
  "questions": [
    {
      "type": "short_answer",
      "questionText": "What is your student ID?",
      "isRequired": true
    },
    {
      "type": "multiple_choice",
      "questionText": "How would you rate the course?",
      "isRequired": true,
      "choices": ["Excellent", "Good", "Fair", "Poor"]
    },
    {
      "type": "grid_multiple_choice",
      "questionText": "Rate the following aspects:",
      "isRequired": true,
      "rows": ["Content", "Instructor", "Assignments"],
      "columns": ["Poor", "Fair", "Good", "Excellent"]
    }
  ]
}
```

**Response:**
```json
{
  "status": 200,
  "message": "Form created successfully",
  "data": {
    "form": {
      "id": "507f1f77bcf86cd799439011",
      "title": "Student Feedback Form",
      "description": "Please provide your feedback about the course",
      "createdAt": "2024-01-15T10:30:00Z",
      "updatedAt": "2024-01-15T10:30:00Z"
    },
    "questions": [
      {
        "id": "507f1f77bcf86cd799439012",
        "formId": "507f1f77bcf86cd799439011",
        "type": "short_answer",
        "questionText": "What is your student ID?",
        "isRequired": true,
        "order": 1
      }
    ]
  }
}
```

### 2. Get All Forms
**GET** `/forms?page=1&limit=10`

Retrieves all forms with pagination.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 10, max: 100)

**Response:**
```json
{
  "status": 200,
  "message": "Forms retrieved successfully",
  "data": {
    "forms": [
      {
        "id": "507f1f77bcf86cd799439011",
        "title": "Student Feedback Form",
        "description": "Please provide your feedback about the course",
        "createdAt": "2024-01-15T10:30:00Z",
        "updatedAt": "2024-01-15T10:30:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 10,
    "totalPages": 1
  }
}
```

### 3. Get Form by ID
**GET** `/forms/{id}`

Retrieves a specific form with all its questions.

**Response:**
```json
{
  "status": 200,
  "message": "Form retrieved successfully",
  "data": {
    "form": {
      "id": "507f1f77bcf86cd799439011",
      "title": "Student Feedback Form",
      "description": "Please provide your feedback about the course",
      "createdAt": "2024-01-15T10:30:00Z",
      "updatedAt": "2024-01-15T10:30:00Z"
    },
    "questions": [
      {
        "id": "507f1f77bcf86cd799439012",
        "formId": "507f1f77bcf86cd799439011",
        "type": "short_answer",
        "questionText": "What is your student ID?",
        "isRequired": true,
        "order": 1
      }
    ]
  }
}
```

### 4. Submit Form
**POST** `/forms/{id}/submissions`

Submits answers to a specific form.

**Request Body:**
```json
{
  "answers": [
    {
      "questionId": "507f1f77bcf86cd799439012",
      "value": "12345"
    },
    {
      "questionId": "507f1f77bcf86cd799439013",
      "value": "Good"
    },
    {
      "questionId": "507f1f77bcf86cd799439014",
      "value": {
        "Content": "Good",
        "Instructor": "Excellent",
        "Assignments": "Fair"
      }
    }
  ]
}
```

**Response:**
```json
{
  "status": 200,
  "message": "Form submitted successfully",
  "data": {
    "id": "507f1f77bcf86cd799439015",
    "formId": "507f1f77bcf86cd799439011",
    "submittedAt": "2024-01-15T11:00:00Z",
    "answers": [
      {
        "questionId": "507f1f77bcf86cd799439012",
        "value": "12345"
      }
    ]
  }
}
```

### 5. Get Form Submissions
**GET** `/forms/{id}/submissions?page=1&limit=10`

Retrieves all submissions for a specific form with pagination.

**Response:**
```json
{
  "status": 200,
  "message": "Submissions retrieved successfully",
  "data": {
    "submissions": [
      {
        "id": "507f1f77bcf86cd799439015",
        "formId": "507f1f77bcf86cd799439011",
        "submittedAt": "2024-01-15T11:00:00Z",
        "answers": [
          {
            "questionId": "507f1f77bcf86cd799439012",
            "value": "12345"
          }
        ]
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 10,
    "totalPages": 1
  }
}
```

## Answer Value Formats

### Text Questions (short_answer, paragraph)
```json
{
  "questionId": "507f1f77bcf86cd799439012",
  "value": "This is a text answer"
}
```

### Single Choice Questions (multiple_choice, dropdown)
```json
{
  "questionId": "507f1f77bcf86cd799439013",
  "value": "Excellent"
}
```

### Multiple Choice Questions (checkbox)
```json
{
  "questionId": "507f1f77bcf86cd799439014",
  "value": ["Choice 1", "Choice 3", "Choice 5"]
}
```

### Grid Multiple Choice
```json
{
  "questionId": "507f1f77bcf86cd799439015",
  "value": {
    "Row 1": "Column 2",
    "Row 2": "Column 1",
    "Row 3": "Column 3"
  }
}
```

### Grid Checkbox
```json
{
  "questionId": "507f1f77bcf86cd799439016",
  "value": {
    "Row 1": ["Column 1", "Column 3"],
    "Row 2": ["Column 2"],
    "Row 3": ["Column 1", "Column 2", "Column 4"]
  }
}
```

## Error Responses

All endpoints return consistent error responses:

```json
{
  "status": 400,
  "message": "Invalid request body"
}
```

Common HTTP status codes:
- `200` - Success
- `400` - Bad Request (validation errors)
- `404` - Not Found (form not found)
- `500` - Internal Server Error

## CURL Examples

### Create a Form
```bash
curl -X POST http://localhost:8888/api/forms \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Student Feedback Form",
    "description": "Please provide your feedback about the course",
    "questions": [
      {
        "type": "short_answer",
        "questionText": "What is your student ID?",
        "isRequired": true
      },
      {
        "type": "multiple_choice",
        "questionText": "How would you rate the course?",
        "isRequired": true,
        "choices": ["Excellent", "Good", "Fair", "Poor"]
      }
    ]
  }'
```

### Get All Forms
```bash
curl -X GET "http://localhost:8888/api/forms?page=1&limit=10"
```

### Get Form by ID
```bash
curl -X GET http://localhost:8888/api/forms/507f1f77bcf86cd799439011
```

### Submit Form
```bash
curl -X POST http://localhost:8888/api/forms/507f1f77bcf86cd799439011/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "answers": [
      {
        "questionId": "507f1f77bcf86cd799439012",
        "value": "12345"
      },
      {
        "questionId": "507f1f77bcf86cd799439013",
        "value": "Good"
      }
    ]
  }'
```

### Get Form Submissions
```bash
curl -X GET "http://localhost:8888/api/forms/507f1f77bcf86cd799439011/submissions?page=1&limit=10"
```

## Running the Application

1. **Install dependencies:**
   ```bash
   go mod tidy
   ```

2. **Set up environment variables:**
   Create a `.env` file with:
   ```
   MONGODB_URI=mongodb://localhost:27017
   MONGODB_DATABASE=bluelock
   APP_URI=8888
   ```

3. **Run the application:**
   ```bash
   go run src/main.go
   ```

4. **Seed sample data (optional):**
   ```bash
   # Add this to main.go temporarily
   seeder.SeedSampleForms(database.MongoDB)
   seeder.SeedSampleSubmissions(database.MongoDB)
   ```

## Database Collections

The system uses three MongoDB collections in the `BluelockDB` database:

1. **forms** - Stores form metadata
2. **questions** - Stores form questions
3. **submissions** - Stores form submissions with answers

## Validation Rules

- Form title is required
- At least one question is required per form
- Question text is required
- Choices are required for multiple_choice, checkbox, and dropdown questions
- Rows and columns are required for grid questions
- All required questions must be answered in submissions
- Answer values must match the question type format

## Project Structure

```
src/
├── models/
│   └── form.go                 # Data models and DTOs
├── services/forms/
│   └── service.go              # Business logic (standalone functions)
├── controllers/
│   └── form_controller.go      # HTTP handlers (standalone functions)
├── routes/
│   └── form_routes.go          # Route definitions
├── utils/
│   └── error_utils.go          # Utility functions
├── seeder/
│   └── form_seeder.go          # Sample data
└── main.go                     # Application entry point
```

## Key Architectural Decisions

1. **Service Pattern**: Uses standalone functions instead of struct methods, following the existing project pattern
2. **Initialization**: Services use `init()` function to initialize MongoDB collections
3. **No Dependency Injection**: Controllers call service functions directly
4. **Consistent Error Handling**: All endpoints return standardized error responses
5. **Validation**: Comprehensive validation at both request and business logic levels 