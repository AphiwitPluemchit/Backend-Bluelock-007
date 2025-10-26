package controllers

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// parseTimeString แปลง string เวลา (HH:mm) และวันที่ (YYYY-MM-DD) เป็น time.Time
func parseTimeString(dateStr string, timeStr string) (*time.Time, error) {
	if dateStr == "" || timeStr == "" {
		return nil, nil
	}

	// Parse date and time together
	dateTimeStr := fmt.Sprintf("%sT%s:00+07:00", dateStr, timeStr) // เพิ่ม timezone +07:00 สำหรับไทย
	parsedTime, err := time.Parse("2006-01-02T15:04:05Z07:00", dateTimeStr)
	if err != nil {
		return nil, err
	}

	return &parsedTime, nil
}

// CreateTestEnrollmentRequest - สำหรับสร้างข้อมูล enrollment
type CreateTestEnrollmentRequest struct {
	StudentCode   string  `json:"studentCode" example:"6516030959"`              // รหัสนิสิต
	ProgramItemID string  `json:"programItemId" example:"507f1f77bcf86cd799439"` // ID ของ programItem
	Food          *string `json:"food,omitempty" example:"vegetarian"`           // อาหาร (optional)
}

// UpdateCheckInOutRequest - สำหรับอัปเดต check-in/out records
type UpdateCheckInOutRequest struct {
	EnrollmentID     string                 `json:"enrollmentId" example:"507f1f77bcf86cd799439"` // ID ของ enrollment
	CheckinoutRecord []CheckInOutRecordItem `json:"checkinoutRecord"`                             // รายการ check-in/out
}

// CheckInOutRecordItem - รายการ check-in/out แต่ละวัน
type CheckInOutRecordItem struct {
	Date     string  `json:"date" example:"2024-10-26"`          // วันที่ (YYYY-MM-DD)
	CheckIn  *string `json:"checkin,omitempty" example:"09:00"`  // เวลา check in (HH:mm format) (optional)
	CheckOut *string `json:"checkout,omitempty" example:"17:30"` // เวลา check out (HH:mm format) (optional)
}

// CreateTestEnrollment สร้าง enrollment สำหรับทดสอบ (ไม่มี check-in/out)
// @Summary สร้างข้อมูลทดสอบ Enrollment
// @Description สร้าง enrollment สำหรับการทดสอบ (ไม่รวม check-in/out)
// @Tags Test Data
// @Accept json
// @Produce json
// @Param body body CreateTestEnrollmentRequest true "ข้อมูลสำหรับสร้าง enrollment"
// @Success 200 {object} models.SuccessResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/test/enrollment [post]
func CreateTestEnrollment(c *fiber.Ctx) error {
	var req CreateTestEnrollmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.StudentCode == "" || req.ProgramItemID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "studentCode and programItemId are required",
		})
	}

	ctx := context.Background()

	// 1. หา Student จาก StudentCode
	var student models.Student
	err := database.StudentCollection.FindOne(ctx, bson.M{"code": req.StudentCode}).Decode(&student)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fmt.Sprintf("Student with code %s not found", req.StudentCode),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error finding student",
		})
	}

	// 2. แปลง ProgramItemID เป็น ObjectID
	programItemID, err := primitive.ObjectIDFromHex(req.ProgramItemID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid programItemId format",
		})
	}

	// 3. หา ProgramItem
	var programItem models.ProgramItem
	err = database.ProgramItemCollection.FindOne(ctx, bson.M{"_id": programItemID}).Decode(&programItem)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Program item not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error finding program item",
		})
	}

	// 4. หา Program
	var program models.Program
	err = database.ProgramCollection.FindOne(ctx, bson.M{"_id": programItem.ProgramID}).Decode(&program)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Program not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error finding program",
		})
	}

	// 5. ตรวจสอบว่ามี enrollment อยู่แล้วหรือไม่
	var existingEnrollment models.Enrollment
	err = database.EnrollmentCollection.FindOne(ctx, bson.M{
		"studentId":     student.ID,
		"programId":     programItem.ProgramID,
		"programItemId": programItemID,
	}).Decode(&existingEnrollment)

	if err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":        "Enrollment already exists",
			"enrollmentId": existingEnrollment.ID.Hex(),
		})
	} else if err != mongo.ErrNoDocuments {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error checking existing enrollment",
		})
	}

	// 6. สร้าง Enrollment (ไม่มี check-in/out records)
	enrollment := models.Enrollment{
		RegistrationDate: time.Now(),
		ProgramID:        programItem.ProgramID,
		ProgramItemID:    programItemID,
		StudentID:        student.ID,
		Food:             req.Food,
		CheckinoutRecord: nil, // ไม่มี check-in/out
		AttendedAllDays:  nil,
	}

	result, err := database.EnrollmentCollection.InsertOne(ctx, enrollment)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating enrollment",
		})
	}

	enrollmentID := result.InsertedID.(primitive.ObjectID)

	// 7. Return response
	return c.Status(http.StatusOK).JSON(models.SuccessResponse{
		Message: "Test enrollment created successfully",
		Data: fiber.Map{
			"enrollmentId": enrollmentID.Hex(),
			"studentId":    student.ID.Hex(),
			"studentCode":  student.Code,
			"programId":    programItem.ProgramID.Hex(),
			"programName":  program.Name,
			"programItem": fiber.Map{
				"id":   programItemID.Hex(),
				"name": programItem.Name,
			},
			"food": req.Food,
		},
	})
}

// UpdateCheckInOutRecords อัปเดต check-in/out records ของ enrollment
// @Summary อัปเดต Check-in/out Records
// @Description อัปเดตรายการ check-in/out ของ enrollment (เพิ่ม/ลบ/แก้ไข)
// @Tags Test Data
// @Accept json
// @Produce json
// @Param body body UpdateCheckInOutRequest true "ข้อมูลสำหรับอัปเดต check-in/out records"
// @Success 200 {object} models.SuccessResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/test/checkinout [put]
func UpdateCheckInOutRecords(c *fiber.Ctx) error {
	var req UpdateCheckInOutRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	fmt.Println("Received request:", req)

	// Validate required fields
	if req.EnrollmentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "enrollmentId is required",
		})
	}

	ctx := context.Background()

	// 1. แปลง EnrollmentID เป็น ObjectID
	enrollmentID, err := primitive.ObjectIDFromHex(req.EnrollmentID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid enrollmentId format",
		})
	}

	// 2. หา Enrollment
	var enrollment models.Enrollment
	err = database.EnrollmentCollection.FindOne(ctx, bson.M{"_id": enrollmentID}).Decode(&enrollment)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Enrollment not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error finding enrollment",
		})
	}

	// 3. หา ProgramItem (สำหรับเช็ค attendedAllDays)
	var programItem models.ProgramItem
	err = database.ProgramItemCollection.FindOne(ctx, bson.M{"_id": enrollment.ProgramItemID}).Decode(&programItem)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error finding program item",
		})
	}

	// 4. สร้าง CheckinoutRecord array ใหม่จาก request
	var checkinoutRecords []models.CheckinoutRecord
	for _, record := range req.CheckinoutRecord {
		// แปลงเวลาจาก string เป็น time.Time
		var checkinTime, checkoutTime *time.Time
		var err error

		if record.CheckIn != nil {
			checkinTime, err = parseTimeString(record.Date, *record.CheckIn)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid checkin time format for date %s: %s. Use HH:mm format (e.g., 09:00)", record.Date, *record.CheckIn),
				})
			}
		}

		if record.CheckOut != nil {
			checkoutTime, err = parseTimeString(record.Date, *record.CheckOut)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid checkout time format for date %s: %s. Use HH:mm format (e.g., 17:30)", record.Date, *record.CheckOut),
				})
			}
		}

		checkinoutRecords = append(checkinoutRecords, models.CheckinoutRecord{
			ID:       primitive.NewObjectID(),
			Checkin:  checkinTime,
			Checkout: checkoutTime,
		})
	}

	// 5. อัปเดต enrollment
	var checkinoutRecordsPtr *[]models.CheckinoutRecord
	if len(checkinoutRecords) > 0 {
		checkinoutRecordsPtr = &checkinoutRecords
	} else {
		checkinoutRecordsPtr = nil
	}

	// Check if attended all days
	attendedAllDays := false
	if len(checkinoutRecords) > 0 {
		allHaveCheckInOut := true
		for _, record := range checkinoutRecords {
			if record.Checkin == nil || record.Checkout == nil {
				allHaveCheckInOut = false
				break
			}
		}
		if allHaveCheckInOut && len(checkinoutRecords) == len(programItem.Dates) {
			attendedAllDays = true
		}
	}

	update := bson.M{
		"$set": bson.M{
			"checkinoutRecord": checkinoutRecordsPtr,
			"attendedAllDays":  attendedAllDays,
		},
	}

	_, err = database.EnrollmentCollection.UpdateOne(ctx, bson.M{"_id": enrollment.ID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating enrollment",
		})
	}

	// 6. อัปเดตหรือสร้าง HourChangeHistory ถ้า attended all days
	if attendedAllDays {
		// หา Program
		var program models.Program
		err = database.ProgramCollection.FindOne(ctx, bson.M{"_id": programItem.ProgramID}).Decode(&program)
		if err == nil {
			// คำนวณชั่วโมงที่ได้รับ
			totalHours := 0
			if programItem.Hour != nil {
				totalHours = *programItem.Hour
			}

			// สร้าง hour history record
			skillType := "soft"
			if program.Skill == "hard" {
				skillType = "hard"
			}

			// หา hour history ที่มีอยู่แล้ว
			var existingHistory models.HourChangeHistory
			err = database.HourChangeHistoryCollection.FindOne(ctx, bson.M{
				"enrollmentId": enrollment.ID,
				"sourceType":   "program",
			}).Decode(&existingHistory)

			if err == mongo.ErrNoDocuments {
				// สร้างใหม่ถ้ายังไม่มี
				hourHistory := models.HourChangeHistory{
					SkillType:    skillType,
					Status:       models.HCStatusAttended,
					HourChange:   totalHours,
					Remark:       fmt.Sprintf("Test data - Attended %s", *program.Name),
					ChangeAt:     time.Now(),
					Title:        *program.Name,
					StudentID:    enrollment.StudentID,
					EnrollmentID: &enrollment.ID,
					SourceType:   "program",
					SourceID:     programItem.ProgramID,
				}
				_, err = database.HourChangeHistoryCollection.InsertOne(ctx, hourHistory)
				if err != nil {
					log.Printf("Error creating hour history: %v", err)
				}
			}
		}
	} else {
		// ถ้าไม่ได้ attended all days ให้ลบ hour history (ถ้ามี)
		_, err = database.HourChangeHistoryCollection.DeleteMany(ctx, bson.M{
			"enrollmentId": enrollment.ID,
			"sourceType":   "program",
		})
		if err != nil {
			log.Printf("Error deleting hour history: %v", err)
		}
	}

	// 7. Return response
	return c.Status(http.StatusOK).JSON(models.SuccessResponse{
		Message: "Check-in/out records updated successfully",
		Data: fiber.Map{
			"enrollmentId":      enrollment.ID.Hex(),
			"recordsCount":      len(checkinoutRecords),
			"attendedAllDays":   attendedAllDays,
			"hourHistoryStatus": map[bool]string{true: "created/updated", false: "deleted"}[attendedAllDays],
		},
	})
}

// DeleteTestEnrollment ลบ enrollment ทดสอบ
// @Summary ลบข้อมูลทดสอบ Enrollment
// @Description ลบ enrollment และ hour history ที่เกี่ยวข้อง
// @Tags Test Data
// @Produce json
// @Param enrollmentId path string true "Enrollment ID"
// @Success 200 {object} models.SuccessResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/test/enrollment/{enrollmentId} [delete]
func DeleteTestEnrollment(c *fiber.Ctx) error {
	enrollmentID, err := primitive.ObjectIDFromHex(c.Params("enrollmentId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid enrollmentId format",
		})
	}

	ctx := context.Background()

	// 1. ลบ enrollment
	result, err := database.EnrollmentCollection.DeleteOne(ctx, bson.M{"_id": enrollmentID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting enrollment",
		})
	}

	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Enrollment not found",
		})
	}

	// 2. ลบ hour history ที่เกี่ยวข้อง
	_, err = database.HourChangeHistoryCollection.DeleteMany(ctx, bson.M{"enrollmentId": enrollmentID})
	if err != nil {
		log.Printf("Error deleting hour histories: %v", err)
	}

	return c.Status(http.StatusOK).JSON(models.SuccessResponse{
		Message: "Test enrollment deleted successfully",
		Data: fiber.Map{
			"enrollmentId": enrollmentID.Hex(),
		},
	})
}
