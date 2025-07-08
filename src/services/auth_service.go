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
		ID:    dbUser.ID,
		Name:  dbUser.Name,
		Email: dbUser.Email,
		Role:  dbUser.Role,
		RefID: dbUser.RefID,
		Code:  dbUser.Code,
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
		}
	case "Admin":
		var admin models.Admin
		adminCol := database.GetCollection("BluelockDB", "admins")
		err := adminCol.FindOne(ctx, bson.M{"_id": dbUser.RefID}).Decode(&admin)
		if err == nil {
			result.ID = admin.ID
			result.Name = admin.Name
		}
	}

	return result, nil
}
