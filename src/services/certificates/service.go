package services

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateUploadCertificate(uploadCertificate *models.UploadCertificate) (*mongo.InsertOneResult, error) {
	ctx := context.Background()
	return DB.UploadCertificateCollection.InsertOne(ctx, uploadCertificate)
}

func UpdateUploadCertificate(id string, uploadCertificate *models.UploadCertificate) (*mongo.UpdateResult, error) {
	ctx := context.Background()
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid upload certificate ID")
	}
	return DB.UploadCertificateCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": uploadCertificate})
}

func IsVerifiedDuplicate(ctx context.Context, url string) (bool, error) {
	if url == "" {
		return false, nil
	}

	filter := bson.M{
		"url":           url,
		"isNameMatch":   true,
		"isCourseMatch": true,
	}

	err := DB.UploadCertificateCollection.FindOne(ctx, filter).Err()
	switch err {
	case nil:
		// พบแล้ว → เป็นซ้ำ
		return true, nil
	case mongo.ErrNoDocuments:
		// ไม่พบ → ไม่ซ้ำ
		return false, nil
	default:
		// error อื่นจาก DB
		return false, err
	}
}

func VerifyURL(publicPageURL  string) (string, error) {
	// context พร้อม timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// // find student
	// var student models.Student
	// err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": studentId}).Decode(&student)
	// if err != nil {
	// 	if err == mongo.ErrNoDocuments {
	// 		return "", fmt.Errorf("student not found")
	// 	}
	// 	return "", err
	// }

	// // find course
	// var course models.Course
	// err = DB.CourseCollection.FindOne(ctx, bson.M{"_id": courseId}).Decode(&course)
	// if err != nil {
	// 	if err == mongo.ErrNoDocuments {
	// 		return "", fmt.Errorf("course not found")
	// 	}
	// 	return "", err
	// }

	// // แสกนข้อมูลจาก html
	// log.Println("url: ", publicPageURL)
	// log.Println("studentId: ", studentId)
	// log.Println("courseId: ", courseId)


		// สร้าง browser context (headless)
		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),      // ถ้ารันใน container เป็น root ให้เปิดอันนี้
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		)
		allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
		defer allocCancel()
	
		// สร้างแท็บใหม่
		taskCtx, taskCancel := chromedp.NewContext(allocCtx)
		defer taskCancel()
	
		var pdfSrc string
		err := chromedp.Run(taskCtx,
			chromedp.Navigate(publicPageURL),
			// รอจน network เงียบลงหน่อย
			chromedp.Sleep(500*time.Millisecond),
			// รอให้ <embed type="application/pdf"> โผล่ใน DOM
			chromedp.WaitVisible(`embed[type="application/pdf"]`, chromedp.ByQuery),
			// ดึงค่า attribute src
			chromedp.AttributeValue(`embed[type="application/pdf"]`, "src", &pdfSrc, nil, chromedp.ByQuery),
		)
		if err != nil {
			return "", err
		}
		if pdfSrc == "" {
			return "", errors.New("pdf <embed> not found or empty src")
		}
		// ตัดพารามิเตอร์ viewer ออก (#toolbar/navpanes/scrollbar)
		if i := strings.IndexByte(pdfSrc, '#'); i >= 0 {
			pdfSrc = pdfSrc[:i]
		}

		fmt.Println("pdfSrc: ", pdfSrc)


		return pdfSrc, nil
	
	
}
