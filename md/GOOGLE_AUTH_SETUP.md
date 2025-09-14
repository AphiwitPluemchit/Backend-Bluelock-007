# Google OAuth Setup Guide

## Overview
This implementation provides Google OAuth authentication for the Bluelock backend system. Users can login with their university Gmail accounts, and the system will automatically create user accounts if they don't exist.

## Features
- ✅ Google OAuth 2.0 integration
- ✅ Automatic user creation for university emails
- ✅ JWT token generation after successful authentication
- ✅ Email domain validation (only @student.chula.ac.th and @chula.ac.th)
- ✅ Role assignment based on email domain

## API Endpoints

### 1. Start Google OAuth Flow
```
GET /auth/google
```
**Response:**
```json
{
  "url": "https://accounts.google.com/oauth/authorize?..."
}
```

### 2. Google OAuth Callback
```
GET /auth/google/redirect?code=...
```
This endpoint handles the callback from Google and redirects to frontend with token.

**Redirect URL:** `{FRONTEND_URL}/auth/callback?token={jwt_token}`

## Environment Variables
Make sure these are set in your `.env` file:

```env
# Google OAuth Configuration
.............................
GOOGLE_OAUTH_CLIENT_ID=your-client-id-here
GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret-here
GOOGLE_REDIRECT=http://localhost:8888/auth/google/redirect
FRONTEND_URL=http://localhost:9000

# JWT Configuration
JWT_SECRET=your-development-secret-key-here
JWT_EXPIRY=64h
```

## User Creation Logic

### Email Domain Rules:
- `@go.buu.ac.th` → Creates Student role
<!-- - `@chula.ac.th` → Creates Admin role -->
- Other domains → Rejected

### Automatic User Creation:
1. User authenticates with Google
2. System checks if user exists by email
3. If not exists:
   - Creates Student/Admin profile based on email domain
   - Creates User account linking to the profile
   - Sets account as active
4. Returns JWT token with user information

## Testing

### Option 1: Use the Test HTML File
1. Start your backend server: `go run ./src`
2. Open `test_google_auth.html` in your browser
3. Click "Get Google Auth URL"
4. Follow the Google OAuth flow
5. Check the callback URL for the token

### Option 2: Manual Testing
1. GET `http://localhost:8888/auth/google`
2. Copy the returned URL and open in browser
3. Complete Google OAuth
4. Check the redirect URL for the token

## Frontend Integration

### Step 1: Initiate Login
```javascript
// Get Google Auth URL
const response = await fetch('/auth/google');
const { url } = await response.json();

// Redirect user to Google
window.location.href = url;
```

### Step 2: Handle Callback
Create a callback page at `/auth/callback` in your frontend:

```javascript
// Extract token from URL
const urlParams = new URLSearchParams(window.location.search);
const token = urlParams.get('token');

if (token) {
    // Store token
    localStorage.setItem('authToken', token);
    
    // Redirect to dashboard or home
    window.location.href = '/dashboard';
} else {
    // Handle error
    console.error('No token received');
}
```

### Step 3: Use Token for API Calls
```javascript
// Include token in API requests
const response = await fetch('/api/protected-endpoint', {
    headers: {
        'Authorization': `Bearer ${token}`
    }
});
```

## Security Notes

1. **HTTPS Required in Production**: Google OAuth requires HTTPS for production URLs
2. **State Parameter**: Consider implementing state parameter validation for additional security
3. **Token Expiry**: JWT tokens expire based on JWT_EXPIRY setting
4. **Email Validation**: Only university email domains are allowed

## Troubleshooting

### Common Issues:

1. **"Invalid credentials" error**
   - Check GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET
   - Verify redirect URL matches Google Console settings

2. **"Only university email addresses are allowed"**
   - User tried to login with non-university email
   - Update email domain validation in `CreateGoogleUser` function

3. **"Token generation failed"**
   - Check JWT_SECRET environment variable
   - Verify JWT utility functions

4. **CORS issues**
   - Ensure ALLOWED_ORIGINS includes your frontend URL
   - Check CORS middleware configuration

## Google Console Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create or select a project
3. Enable Google+ API
4. Create OAuth 2.0 credentials
5. Add authorized redirect URIs:
   - `http://localhost:8888/auth/google/redirect` (development)
   - `https://yourdomain.com/auth/google/redirect` (production)

## Database Collections

The system uses these MongoDB collections:
- `Users` - Main user accounts
- `Students` - Student profiles
- `Admins` - Admin profiles

User accounts reference the appropriate profile via `refId` field.