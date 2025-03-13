package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

func AuthenticateUser(email, password string) (*models.User, error) {
	ctx := context.Background()

	userCollection := database.GetCollection("BluelockDB", "users")
	var dbUser models.User

	err := userCollection.FindOne(ctx, bson.M{"email": strings.ToLower(email)}).Decode(&dbUser)
	if err != nil {
		return nil, errors.New("Invalid email or password")
	}

	// ตรวจสอบ password
	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(password)); err != nil {
		return nil, errors.New("Invalid  password")
	}

	// เตรียมผลลัพธ์เริ่มต้น
	result := &models.User{
		ID:    dbUser.ID,
		Email: dbUser.Email,
		Role:  dbUser.Role,
	}

	// ถ้าเป็น Student → ดึงข้อมูล Student จาก studentId
	if dbUser.Role == "Student" && dbUser.StudentID != nil {
		studentCol := database.GetCollection("BluelockDB", "students")
		var student models.Student
		err := studentCol.FindOne(ctx, bson.M{"_id": dbUser.StudentID}).Decode(&student)
		if err == nil {
			result.Email = student.Email
		}
	}

	// ถ้าเป็น Admin → ดึงข้อมูล Admin จาก adminId
	if dbUser.Role == "Admin" && dbUser.AdminID != nil {
		adminCol := database.GetCollection("BluelockDB", "admins")
		var admin models.Admin
		err := adminCol.FindOne(ctx, bson.M{"_id": dbUser.AdminID}).Decode(&admin)
		if err == nil {
			result.Email = admin.Email
		}
	}

	return result, nil
}
