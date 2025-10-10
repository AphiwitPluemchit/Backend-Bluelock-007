package services

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// Rate limiting variables
// TODO: สำหรับ Production ควรใช้ Redis แทน memory เพื่อรองรับ multiple servers
var (
	loginAttempts = make(map[string][]time.Time) // key: "email"
	attemptsMutex sync.RWMutex
)

// IsRateLimited ตรวจสอบ rate limiting สำหรับ email
func IsRateLimited(email string) bool {
	attemptsMutex.Lock()
	defer attemptsMutex.Unlock()

	key := strings.ToLower(email)
	now := time.Now()
	window := 5 * time.Minute         // 5 นาที
	maxAttempts := 5                  // สูงสุด 5 ครั้ง
	cooldownPeriod := 5 * time.Minute // รอ 5 นาทีหลังจากเกิน maxAttempts

	// ลบ attempts ที่เก่ากว่า window
	if attempts, exists := loginAttempts[key]; exists {
		var validAttempts []time.Time
		for _, attempt := range attempts {
			if now.Sub(attempt) < window {
				validAttempts = append(validAttempts, attempt)
			}
		}
		loginAttempts[key] = validAttempts
	}

	// ตรวจสอบจำนวน attempts
	if attempts, exists := loginAttempts[key]; exists {
		if len(attempts) >= maxAttempts {
			// ตรวจสอบว่าเกิน cooldown period หรือยัง
			oldestAttempt := attempts[0]
			if now.Sub(oldestAttempt) < cooldownPeriod {
				// ยังอยู่ในช่วง cooldown
				return true
			} else {
				// ผ่าน cooldown period แล้ว ให้รีเซ็ต attempts
				loginAttempts[key] = []time.Time{}
			}
		}
	}

	// เพิ่ม attempt ปัจจุบัน
	loginAttempts[key] = append(loginAttempts[key], now)
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

	// Use the initialized collection from DB package
	_, err := DB.UserCollection.UpdateOne(ctx,
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"lastLogout": time.Now()}},
	)

	if err != nil {
		log.Printf("Failed to update last logout for user %s: %v", userID, err)
	}
}

// GetRemainingCooldownTime คำนวณเวลาที่เหลือสำหรับ email ที่ถูก rate limit
func GetRemainingCooldownTime(email string) time.Duration {
	attemptsMutex.RLock()
	defer attemptsMutex.RUnlock()

	key := strings.ToLower(email)
	now := time.Now()
	cooldownPeriod := 5 * time.Minute

	if attempts, exists := loginAttempts[key]; exists && len(attempts) > 0 {
		oldestAttempt := attempts[0]
		elapsed := now.Sub(oldestAttempt)
		remaining := cooldownPeriod - elapsed

		if remaining > 0 {
			return remaining
		}
	}

	return 0
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

// enrichUserProfile ดึงข้อมูลเพิ่มเติมจาก Student/Admin collection
// ✅ ฟังก์ชันกลางที่ใช้ร่วมกันในทุก auth flow
func enrichUserProfile(ctx context.Context, user *models.User) error {
	switch user.Role {
	case "Student":
		var student models.Student
		err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": user.RefID}).Decode(&student)
		if err != nil {
			log.Printf("Warning: Could not fetch student profile for RefID %s: %v", user.RefID.Hex(), err)
			return err
		}
		user.Name = student.Name
		user.Code = student.Code
		user.Major = student.Major
		user.StudentYear = extractStudentYearFromCode(student.Code)

	case "Admin":
		var admin models.Admin
		err := DB.AdminCollection.FindOne(ctx, bson.M{"_id": user.RefID}).Decode(&admin)
		if err != nil {
			log.Printf("Warning: Could not fetch admin profile for RefID %s: %v", user.RefID.Hex(), err)
			return err
		}
		user.Name = admin.Name
	}

	return nil
}

// AuthenticateUser ตรวจสอบ email และ password
func AuthenticateUser(email, password string) (*models.User, error) {
	ctx := context.Background()

	// 1. หา user จาก email
	var dbUser models.User
	err := DB.UserCollection.FindOne(ctx, bson.M{"email": strings.ToLower(email)}).Decode(&dbUser)
	if err != nil {
		return nil, errors.New("Invalid email or password")
	}

	// 2. ตรวจสอบสถานะการใช้งาน
	if !dbUser.IsActive {
		return nil, errors.New("บัญชีนี้ถูกระงับการใช้งาน")
	}

	// 3. ตรวจสอบ password
	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(password)); err != nil {
		return nil, errors.New("Invalid password")
	}

	// 4. เตรียมข้อมูล response
	result := &models.User{
		ID:          dbUser.ID,
		Email:       dbUser.Email,
		Role:        dbUser.Role,
		RefID:       dbUser.RefID,
		Major:       "",
		StudentYear: 0,
	}

	// 5. ดึงข้อมูลเพิ่มเติมจาก Student/Admin collection
	enrichUserProfile(ctx, result)

	return result, nil
}

// GetUserByEmail ดึงข้อมูล user จาก email (ใช้สำหรับ Google OAuth)
func GetUserByEmail(email string) (*models.User, error) {
	ctx := context.Background()

	// 1. หา user จาก email
	var dbUser models.User
	err := DB.UserCollection.FindOne(ctx, bson.M{"email": strings.ToLower(email)}).Decode(&dbUser)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// 2. ตรวจสอบสถานะการใช้งาน
	if !dbUser.IsActive {
		return nil, errors.New("account is suspended")
	}

	// 3. เตรียมข้อมูล response
	result := &models.User{
		ID:          dbUser.ID,
		Email:       dbUser.Email,
		Role:        dbUser.Role,
		RefID:       dbUser.RefID,
		Major:       "",
		StudentYear: 0,
	}

	// 4. ดึงข้อมูลเพิ่มเติมจาก Student/Admin collection
	enrichUserProfile(ctx, result)

	return result, nil
}

// GetUserProfile ดึงข้อมูล user profile จาก JWT token (ใช้สำหรับ /auth/me)
func GetUserProfile(userID, role string) (*models.User, error) {
	ctx := context.Background()

	// 1. แปลง userID string เป็น ObjectID
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %v", err)
	}

	// 2. หา user จาก RefID (เพราะ JWT เก็บ RefID ซึ่งเป็น Student/Admin ID)
	var dbUser models.User
	err = DB.UserCollection.FindOne(ctx, bson.M{"refId": objID}).Decode(&dbUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %v", err)
	}

	// 3. ตรวจสอบสถานะการใช้งาน
	if !dbUser.IsActive {
		return nil, errors.New("account is suspended")
	}

	// 4. เตรียมข้อมูล response
	result := &models.User{
		ID:          dbUser.ID,
		Email:       dbUser.Email,
		Role:        dbUser.Role,
		RefID:       dbUser.RefID,
		Major:       "",
		StudentYear: 0,
	}

	// 5. ดึงข้อมูลเพิ่มเติมจาก Student/Admin collection
	enrichUserProfile(ctx, result)

	return result, nil
}

// CreateGoogleUser creates a new user from Google OAuth information
func CreateGoogleUser(googleUser *GoogleUserInfo) (*models.User, error) {
	ctx := context.Background()

	// Check if it's a university email (you can customize this logic)
	if !strings.HasSuffix(strings.ToLower(googleUser.Email), "@go.buu.ac.th") {
		return nil, errors.New("only university email addresses are allowed")
	}
	// && !strings.HasSuffix(strings.ToLower(googleUser.Email), "@chula.ac.th")

	// Determine role based on email domain
	// role := "Student"
	// if strings.HasSuffix(strings.ToLower(googleUser.Email), "@chula.ac.th") {
	// 	role = "Admin"
	// }

	// ดึงรหัส
	local := strings.ToLower(strings.TrimSpace(googleUser.Email))
	parts := strings.SplitN(local, "@", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid email format")
	}
	// เอาแบบเข้ม: เฉพาะตัวเลขนำหน้า
	re := regexp.MustCompile(`^\d+`)
	code := re.FindString(parts[0])
	if code == "" {
		return nil, errors.New("no numeric student code found in email")
	}

	// -- เตรียมตัวแปร --
	var refID primitive.ObjectID
	var student models.Student
	err := DB.StudentCollection.FindOne(ctx, bson.M{"code": code}).Decode(&student)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// ไม่เจอ => สร้างใหม่
			student = models.Student{
				EngName: googleUser.Name,
				Code:    code,
				Major:   "",
			}
			fmt.Println("create student => :", student)

			insertRes, err := DB.StudentCollection.InsertOne(ctx, student)
			if err != nil {
				return nil, fmt.Errorf("failed to create student profile: %v", err)
			}

			// InsertedID เป็น primitive.ObjectID (ไม่ใช่ pointer)
			oid, ok := insertRes.InsertedID.(primitive.ObjectID)
			if !ok {
				return nil, fmt.Errorf("unexpected InsertedID type %T", insertRes.InsertedID)
			}
			refID = oid
		} else {
			// เออเรอร์อื่น ๆ
			return nil, fmt.Errorf("failed to query student: %v", err)
		}
	} else {
		// เจออยู่แล้ว
		refID = student.ID
	}

	// Create user account
	user := models.User{
		Email:    strings.ToLower(googleUser.Email),
		Role:     "Student",
		Code:     code,
		RefID:    refID,
		IsActive: true,
	}

	_, err = DB.UserCollection.InsertOne(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user account: %v", err)
	}

	// ดึงข้อมูล student profile ล่าสุด
	var studentProfile models.Student
	err = DB.StudentCollection.FindOne(ctx, bson.M{"_id": refID}).Decode(&studentProfile)
	if err != nil {
		// ถ้า error ให้คืนข้อมูลเท่าที่มี
		return &models.User{
			ID:    refID,
			Name:  googleUser.Name,
			Email: strings.ToLower(googleUser.Email),
			Role:  "Student",
			RefID: refID,
			Code:  code,
			Major: "",
		}, nil
	}

	// ดึงข้อมูล user ล่าสุด
	var dbUser models.User
	err = DB.UserCollection.FindOne(ctx, bson.M{"email": strings.ToLower(googleUser.Email)}).Decode(&dbUser)
	if err != nil {
		// ถ้า error ให้คืนข้อมูลเท่าที่มี
		return &models.User{
			ID:          refID,
			Name:        studentProfile.Name,
			Email:       strings.ToLower(googleUser.Email),
			Role:        "Student",
			RefID:       refID,
			Code:        code,
			Major:       studentProfile.Major,
			StudentYear: extractStudentYearFromCode(studentProfile.Code),
		}, nil
	}

	// คืนข้อมูลครบถ้วนเหมือน Normal Login
	return &models.User{
		ID:          dbUser.ID,
		Name:        studentProfile.Name,
		Email:       dbUser.Email,
		Role:        dbUser.Role,
		RefID:       dbUser.RefID,
		Code:        studentProfile.Code,
		Major:       studentProfile.Major,
		StudentYear: extractStudentYearFromCode(studentProfile.Code),
		LastLogin:   dbUser.LastLogin,
	}, nil
}
