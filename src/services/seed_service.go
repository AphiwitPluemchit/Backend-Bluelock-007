package services

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// SeedUser represents a user to be seeded
type SeedUser struct {
	Email string
	Role  string // "admin" or "student"
	Name  string
	Code  string
	Major string
	Year  int
}

// GeneratedPassword stores email and its generated password
type GeneratedPassword struct {
	Email    string
	Password string
	Role     string
}

// generateRandomPassword สร้างรหัสผ่านแบบสุ่ม
func generateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	password := make([]byte, length)

	for i := range password {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password[i] = charset[num.Int64()]
	}

	return string(password), nil
}

// hashPassword แปลงรหัสผ่านเป็น bcrypt hash
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// SeedInitialUsers สร้าง users เริ่มต้นในระบบ
func SeedInitialUsers() ([]GeneratedPassword, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	usersCollection := DB.GetDefaultCollection("Users")
	adminsCollection := DB.GetDefaultCollection("Admins")
	studentsCollection := DB.GetDefaultCollection("Students")

	// กำหนด users ที่ต้องการ seed
	seedUsers := []SeedUser{
		{
			Email: "kamonwan@go.buu.ac.th",
			Role:  "Admin",
			Name:  "Admin Kamonwan",
			Code:  "ADMIN001",
			Major: "Computer Science",
			Year:  0,
		},
		{
			Email: "65160000@go.buu.ac.th",
			Role:  "Student",
			Name:  "Student 65160000",
			Code:  "65160000",
			Major: "Computer Science",
			Year:  3,
		},
		{
			Email: "65160309@go.buu.ac.th",
			Role:  "Student",
			Name:  "Student 65160309",
			Code:  "65160309",
			Major: "Computer Science",
			Year:  3,
		},
		{
			Email: "65160289@go.buu.ac.th",
			Role:  "Student",
			Name:  "Student 65160289",
			Code:  "65160289",
			Major: "Computer Science",
			Year:  3,
		},
	}

	var generatedPasswords []GeneratedPassword

	log.Println("🌱 Starting seed process...")

	for _, seedUser := range seedUsers {
		// ตรวจสอบว่ามี user นี้อยู่แล้วหรือไม่
		var existingUser models.User
		err := usersCollection.FindOne(ctx, bson.M{"email": seedUser.Email}).Decode(&existingUser)

		if err == nil {
			log.Printf("⏭️  User %s already exists, skipping...", seedUser.Email)
			continue
		} else if err != mongo.ErrNoDocuments {
			return nil, fmt.Errorf("error checking existing user %s: %v", seedUser.Email, err)
		}

		// สร้างรหัสผ่านแบบสุ่ม
		plainPassword, err := generateRandomPassword(12)
		if err != nil {
			return nil, fmt.Errorf("error generating password for %s: %v", seedUser.Email, err)
		}

		// Hash รหัสผ่าน
		hashedPassword, err := hashPassword(plainPassword)
		if err != nil {
			return nil, fmt.Errorf("error hashing password for %s: %v", seedUser.Email, err)
		}

		var refID primitive.ObjectID

		if seedUser.Role == "Admin" {
			// สร้าง Admin record
			admin := bson.M{
				"name":      seedUser.Name,
				"code":      seedUser.Code,
				"major":     seedUser.Major,
				"isActive":  true,
				"createdAt": time.Now(),
			}

			result, err := adminsCollection.InsertOne(ctx, admin)
			if err != nil {
				return nil, fmt.Errorf("error creating admin record for %s: %v", seedUser.Email, err)
			}
			refID = result.InsertedID.(primitive.ObjectID)
			log.Printf("✅ Created admin record for %s", seedUser.Email)

		} else {
			// สร้าง Student record
			student := bson.M{
				"name":        seedUser.Name,
				"code":        seedUser.Code,
				"major":       seedUser.Major,
				"studentYear": seedUser.Year,
				"isActive":    true,
				"createdAt":   time.Now(),
			}

			result, err := studentsCollection.InsertOne(ctx, student)
			if err != nil {
				return nil, fmt.Errorf("error creating student record for %s: %v", seedUser.Email, err)
			}
			refID = result.InsertedID.(primitive.ObjectID)
			log.Printf("✅ Created student record for %s", seedUser.Email)
		}

		// สร้าง User record
		user := models.User{
			Email:    seedUser.Email,
			Password: hashedPassword,
			Role:     seedUser.Role,
			RefID:    refID,
			IsActive: true,
		}

		_, err = usersCollection.InsertOne(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("error creating user record for %s: %v", seedUser.Email, err)
		}

		log.Printf("✅ Created user: %s (Role: %s)", seedUser.Email, seedUser.Role)

		// เก็บ password ที่สร้างไว้
		generatedPasswords = append(generatedPasswords, GeneratedPassword{
			Email:    seedUser.Email,
			Password: plainPassword,
			Role:     seedUser.Role,
		})
	}

	return generatedPasswords, nil
}

// PrintGeneratedPasswords แสดงรหัสผ่านที่สร้างขึ้น
func PrintGeneratedPasswords(passwords []GeneratedPassword) {
	if len(passwords) == 0 {
		log.Println("ℹ️  No new users were created (all users already exist)")
		return
	}

	log.Println("\n" + "═══════════════════════════════════════════════════════════════")
	log.Println("🔐 GENERATED PASSWORDS FOR SEEDED USERS")
	log.Println("═══════════════════════════════════════════════════════════════")
	log.Println("⚠️  IMPORTANT: Save these passwords securely!")
	log.Println("⚠️  These passwords are hashed in the database and cannot be retrieved again.")
	log.Println("───────────────────────────────────────────────────────────────")

	for _, p := range passwords {
		log.Printf("📧 Email:    %s", p.Email)
		log.Printf("🔑 Password: %s", p.Password)
		log.Printf("👤 Role:     %s", p.Role)
		log.Println("───────────────────────────────────────────────────────────────")
	}

	log.Println("═══════════════════════════════════════════════════════════════")
}

// SavePasswordsToFile บันทึกรหัสผ่านที่สร้างไว้ลงไฟล์ (truncate)
func SavePasswordsToFile(passwords []GeneratedPassword, filePath string) error {
	if len(passwords) == 0 {
		return nil
	}

	header := fmt.Sprintf("Generated user credentials - %s\n", time.Now().Format(time.RFC3339))
	var body string
	for _, p := range passwords {
		body += fmt.Sprintf("Email: %s\nRole: %s\nPassword: %s\n------------------------\n", p.Email, p.Role, p.Password)
	}

	content := header + "\n" + body

	// Write the file (truncate/create)
	// Use 0640 permissions; on Windows this is ignored but harmless.
	if err := os.WriteFile(filePath, []byte(content), 0640); err != nil {
		return fmt.Errorf("failed to write passwords to file %s: %w", filePath, err)
	}
	log.Printf("🔐 Saved generated passwords to %s", filePath)
	return nil
}
