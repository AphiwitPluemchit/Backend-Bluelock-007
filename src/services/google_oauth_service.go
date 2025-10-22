package services

import (
	"Backend-Bluelock-007/src/models"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleUserInfo represents the user information from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// GetGoogleOAuthConfig returns the Google OAuth2 configuration
func GetGoogleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

// GetGoogleUserInfo retrieves user information from Google using the access token
func GetGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	url := "https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %v", err)
	}

	return &userInfo, nil
}

// ProcessGoogleLogin handles the Google OAuth login process
func ProcessGoogleLogin(code string) (*models.User, error) {

	config := GetGoogleOAuthConfig()

	// Exchange code for token
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		fmt.Printf("❌ Token exchange failed: %v\n", err)
		return nil, fmt.Errorf("failed to exchange code for token: %v", err)
	}
	fmt.Printf("✅ Token exchange successful\n")

	// Get user info from Google
	userInfo, err := GetGoogleUserInfo(token.AccessToken)
	if err != nil {
		fmt.Printf("❌ Failed to get user info: %v\n", err)
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}
	fmt.Printf("✅ User info retrieved: %s (%s)\n", userInfo.Email, userInfo.Name)

	// Check if user exists in database
	fmt.Printf("🔄 Checking if user exists in database...\n")
	user, err := GetUserByEmail(userInfo.Email)
	if err != nil {
		fmt.Printf("❌ User not found in system: %s\n", userInfo.Email)
		return nil, fmt.Errorf("ผู้ใช้ยังไม่ได้ลงทะเบียนในระบบ กรุณาติดต่อผู้ดูแลระบบ")
	}

	fmt.Printf("✅ Existing user found: %s\n", user.Email)

	return user, nil
}
