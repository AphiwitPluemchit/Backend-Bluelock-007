package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

// Rate limiting variables
var (
	loginAttempts = make(map[string][]time.Time)
	attemptsMutex sync.RWMutex
)

// IsRateLimited ตรวจสอบ rate limiting สำหรับ IP
func IsRateLimited(ip string) bool {
	attemptsMutex.Lock()
	defer attemptsMutex.Unlock()

	now := time.Now()
	window := 5 * time.Minute // 15 นาที
	maxAttempts := 5          // สูงสุด 5 ครั้ง

	// ลบ attempts ที่เก่ากว่า window
	if attempts, exists := loginAttempts[ip]; exists {
		var validAttempts []time.Time
		for _, attempt := range attempts {
			if now.Sub(attempt) < window {
				validAttempts = append(validAttempts, attempt)
			}
		}
		loginAttempts[ip] = validAttempts
	}

	// ตรวจสอบจำนวน attempts
	if attempts, exists := loginAttempts[ip]; exists && len(attempts) >= maxAttempts {
		return true
	}

	// เพิ่ม attempt ปัจจุบัน
	loginAttempts[ip] = append(loginAttempts[ip], now)
	return false
}

// LogLoginAttempt บันทึก login attempt
func LogLoginAttempt(email, ip string, success bool) {
	status := "FAILED"
	if success {
		status = "SUCCESS"
	}

	log.Printf("LOGIN_ATTEMPT: email=%s, ip=%s, status=%s, timestamp=%s",
		email, ip, status, time.Now().Format(time.RFC3339))
}

// LogLogout บันทึก logout
func LogLogout(userID, ip string, timestamp time.Time) {
	log.Printf("LOGOUT: userID=%s, ip=%s, timestamp=%s",
		userID, ip, timestamp.Format(time.RFC3339))
}

// AddToBlacklist เพิ่ม token ลง blacklist (ใช้ Redis ในอนาคต)
func AddToBlacklist(token, userID string) {
	// TODO: ใช้ Redis เพื่อเก็บ blacklisted tokens
	log.Printf("TOKEN_BLACKLISTED: userID=%s, token=%s...", userID, token[:10])
}

// UpdateLastLogout อัปเดต last logout time
func UpdateLastLogout(userID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userCollection := database.GetCollection("BluelockDB", "users")
	_, err := userCollection.UpdateOne(ctx,
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"lastLogout": time.Now()}},
	)

	if err != nil {
		log.Printf("Failed to update last logout for user %s: %v", userID, err)
	}
}

// extractStudentYearFromCode ดึงชั้นปีจากรหัสนิสิต
func extractStudentYearFromCode(code string) int {
	if len(code) < 2 {
		return 0
	}

	// ดึง 2 ตัวแรกของรหัสนิสิต (เช่น 67 จาก 6712345678)
	yearPrefix := code[:2]
	year, err := strconv.Atoi(yearPrefix)
	if err != nil {
		return 0
	}

	// คำนวณปีการศึกษาปัจจุบัน (ปี พ.ศ. - 2500)
	currentYear := time.Now().Year() - 2500 + 543
	studentYear := currentYear - year + 1

	// ตรวจสอบว่าอยู่ในช่วง 1-4 ปี
	if studentYear >= 1 && studentYear <= 4 {
		return studentYear
	}

	return 0
}

func AuthenticateUser(email, password string) (*models.User, error) {
	ctx := context.Background()
	userCollection := database.GetCollection("BluelockDB", "users")

	var dbUser models.User
	err := userCollection.FindOne(ctx, bson.M{"email": strings.ToLower(email)}).Decode(&dbUser)
	if err != nil {
		return nil, errors.New("Invalid email or password")
	}

	// ✅ ตรวจสอบสถานะการใช้งาน
	if !dbUser.IsActive {
		return nil, errors.New("บัญชีนี้ถูกระงับการใช้งาน")
	}

	// ✅ ตรวจสอบ password
	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(password)); err != nil {
		return nil, errors.New("Invalid password")
	}

	// ✅ เตรียมข้อมูล response
	result := &models.User{
		ID:          dbUser.ID,
		Name:        dbUser.Name,
		Email:       dbUser.Email,
		Role:        dbUser.Role,
		RefID:       dbUser.RefID,
		Code:        dbUser.Code,
		Major:       "",
		StudentYear: 0,
	}

	// 🔍 ดึง name จาก profile ตาม role
	switch dbUser.Role {
	case "Student":
		var student models.Student
		studentCol := database.GetCollection("BluelockDB", "students")
		err := studentCol.FindOne(ctx, bson.M{"_id": dbUser.RefID}).Decode(&student)
		if err == nil {
			result.ID = student.ID
			result.Name = student.Name
			result.Code = student.Code
			result.Major = student.Major
			result.StudentYear = extractStudentYearFromCode(student.Code)
		}
	case "Admin":
		var admin models.Admin
		adminCol := database.GetCollection("BluelockDB", "admins")
		err := adminCol.FindOne(ctx, bson.M{"_id": dbUser.RefID}).Decode(&admin)
		if err == nil {
			result.ID = admin.ID
			result.Name = admin.Name

		}
		// Admin ไม่มี major และ studentYear

	}

	return result, nil
}
