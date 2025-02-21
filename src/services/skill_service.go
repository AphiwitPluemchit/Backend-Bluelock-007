package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var skillCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	skillCollection = database.GetCollection("BluelockDB", "skills")
	if skillCollection == nil {
		log.Fatal("Failed to get the skills collection")
	}
}

// CreateSkill - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateSkill(skill *models.Skill) error {
	skill.ID = primitive.NewObjectID() // กำหนด ID อัตโนมัติ
	_, err := skillCollection.InsertOne(context.Background(), skill)
	return err
}

// GetAllSkills - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllSkills() ([]models.Skill, error) {
	var skills []models.Skill
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := skillCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var skill models.Skill
		if err := cursor.Decode(&skill); err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

// GetSkillByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetSkillByID(id string) (*models.Skill, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid skill ID")
	}

	var skill models.Skill
	err = skillCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&skill)
	if err != nil {
		return nil, err
	}

	return &skill, nil
}

// UpdateSkill - อัปเดตข้อมูลผู้ใช้
func UpdateSkill(id string, skill *models.Skill) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid skill ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": skill}

	_, err = skillCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteSkill - ลบข้อมูลผู้ใช้
func DeleteSkill(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid skill ID")
	}

	_, err = skillCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
