#!/bin/bash

# Test script for Forms API
# Make sure the server is running on localhost:8888

BASE_URL="http://localhost:8888/api"

echo "🧪 Testing Forms API..."
echo "========================"

# Test 1: Create a form
echo "1. Creating a form..."
CREATE_RESPONSE=$(curl -s -X POST "$BASE_URL/forms" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Feedback Form",
    "description": "A test form for API testing",
    "questions": [
      {
        "type": "short_answer",
        "questionText": "What is your name?",
        "isRequired": true
      },
      {
        "type": "multiple_choice",
        "questionText": "How would you rate this API?",
        "isRequired": true,
        "choices": ["Excellent", "Good", "Fair", "Poor"]
      },
      {
        "type": "grid_multiple_choice",
        "questionText": "Rate the following:",
        "isRequired": true,
        "rows": ["Ease of Use", "Documentation", "Performance"],
        "columns": ["Poor", "Fair", "Good", "Excellent"]
      }
    ]
  }')

echo "Create Form Response:"
echo "$CREATE_RESPONSE" | jq '.'

# Extract form ID from response
FORM_ID=$(echo "$CREATE_RESPONSE" | jq -r '.data.form.id')
echo "Form ID: $FORM_ID"

echo ""
echo "2. Getting all forms..."
GET_FORMS_RESPONSE=$(curl -s -X GET "$BASE_URL/forms?page=1&limit=10")
echo "Get Forms Response:"
echo "$GET_FORMS_RESPONSE" | jq '.'

echo ""
echo "3. Getting form by ID..."
GET_FORM_RESPONSE=$(curl -s -X GET "$BASE_URL/forms/$FORM_ID")
echo "Get Form Response:"
echo "$GET_FORM_RESPONSE" | jq '.'

# Extract question IDs for submission
QUESTION_1_ID=$(echo "$GET_FORM_RESPONSE" | jq -r '.data.questions[0].id')
QUESTION_2_ID=$(echo "$GET_FORM_RESPONSE" | jq -r '.data.questions[1].id')
QUESTION_3_ID=$(echo "$GET_FORM_RESPONSE" | jq -r '.data.questions[2].id')

echo ""
echo "4. Submitting form..."
SUBMIT_RESPONSE=$(curl -s -X POST "$BASE_URL/forms/$FORM_ID/submissions" \
  -H "Content-Type: application/json" \
  -d "{
    \"answers\": [
      {
        \"questionId\": \"$QUESTION_1_ID\",
        \"value\": \"John Doe\"
      },
      {
        \"questionId\": \"$QUESTION_2_ID\",
        \"value\": \"Excellent\"
      },
      {
        \"questionId\": \"$QUESTION_3_ID\",
        \"value\": {
          \"Ease of Use\": \"Excellent\",
          \"Documentation\": \"Good\",
          \"Performance\": \"Excellent\"
        }
      }
    ]
  }")

echo "Submit Form Response:"
echo "$SUBMIT_RESPONSE" | jq '.'

echo ""
echo "5. Getting form submissions..."
SUBMISSIONS_RESPONSE=$(curl -s -X GET "$BASE_URL/forms/$FORM_ID/submissions?page=1&limit=10")
echo "Get Submissions Response:"
echo "$SUBMISSIONS_RESPONSE" | jq '.'

echo ""
echo "✅ Forms API test completed!" 