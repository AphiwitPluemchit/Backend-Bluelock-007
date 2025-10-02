package enrollments

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ✅ Bulk โดยยังคงใช้กฎจาก RegisterStudent เดิมทุกอย่าง
func RegisterStudentsByCodes(ctx context.Context, programItemID primitive.ObjectID, items []models.BulkEnrollItem) (*models.BulkEnrollResult, error) {
	res := &models.BulkEnrollResult{
		ProgramItemID:  programItemID.Hex(),
		TotalRequested: len(items),
		Success:        make([]models.BulkEnrollSuccessItem, 0, len(items)),
		Failed:         make([]models.BulkEnrollFailedItem, 0),
	}

	// 1) เตรียมรหัสที่ normalize และ dedupe (กันส่งซ้ำ)
	codeSet := make(map[string]struct{}, len(items))
	codes := make([]string, 0, len(items))
	for _, it := range items {
		code := strings.TrimSpace(it.StudentCode)
		if code == "" {
			continue
		}
		if _, ok := codeSet[code]; !ok {
			codeSet[code] = struct{}{}
			codes = append(codes, code)
		}
	}
	// ทำให้มีลำดับคงที่ (optional)
	sort.Strings(codes)

	// 2) ดึง student เป็น batch
	cur, err := DB.StudentCollection.Find(ctx, bson.M{"code": bson.M{"$in": codes}})
	if err != nil {
		return res, fmt.Errorf("failed to query students by codes: %w", err)
	}
	defer cur.Close(ctx)

	codeToStudent := make(map[string]models.Student, len(codes))
	for cur.Next(ctx) {
		var s models.Student
		if derr := cur.Decode(&s); derr == nil {
			codeToStudent[strings.TrimSpace(s.Code)] = s
		}
	}
	if err := cur.Err(); err != nil {
		return res, fmt.Errorf("failed to iterate student cursor: %w", err)
	}

	// 3) วนตาม order ที่ client ส่งมา (report ชัดเจน)
	for _, it := range items {
		code := strings.TrimSpace(it.StudentCode)
		if code == "" {
			res.Failed = append(res.Failed, models.BulkEnrollFailedItem{
				StudentCode: code,
				Reason:      "studentCode is empty",
			})
			continue
		}

		stu, ok := codeToStudent[code]
		if !ok {
			res.Failed = append(res.Failed, models.BulkEnrollFailedItem{
				StudentCode: code,
				Reason:      "student not found",
			})
			continue
		}

		// เรียก service เดิมให้ตรวจทุกกฎ (กันชนเวลา/สาขา/เต็มโควต้า/ลงซ้ำ/เพิ่ม foodVotes/เพิ่ม enrollmentcount)
		if err := RegisterStudent(programItemID, stu.ID, it.Food); err != nil {
			res.Failed = append(res.Failed, models.BulkEnrollFailedItem{
				StudentCode: code,
				Reason:      err.Error(),
			})
			continue
		}

		res.Success = append(res.Success, models.BulkEnrollSuccessItem{
			StudentCode: code,
			StudentID:   stu.ID.Hex(),
			Message:     "enrolled",
		})
	}

	return res, nil
}
