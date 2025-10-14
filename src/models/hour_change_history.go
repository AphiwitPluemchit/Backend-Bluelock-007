package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HourChangeHistory บันทึกประวัติการเปลี่ยนแปลงชั่วโมง
type HourChangeHistory struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	SkillType    string              `bson:"skillType" json:"skillType"`                           // "soft" | "hard"
	Status       string              `bson:"status" json:"status"`                                 // HCStatus* constants
	HourChange   int                 `bson:"hourChange" json:"hourChange"`                         // จำนวนชั่วโมงที่เปลี่ยน (บวก = เพิ่ม, ลบ = ลด)
	Remark       string              `bson:"remark,omitempty" json:"remark,omitempty"`             // หมายเหตุ
	ChangeAt     time.Time           `bson:"changeAt" json:"changeAt"`                             // เวลาที่เกิดการเปลี่ยนแปลง
	Title        string              `bson:"title" json:"title"`                                   // หัวข้อ/ชื่อของการเปลี่ยนแปลง
	StudentID    primitive.ObjectID  `bson:"studentId" json:"studentId"`                           // นิสิตที่ได้รับผลกระทบ
	EnrollmentID *primitive.ObjectID `bson:"enrollmentId,omitempty" json:"enrollmentId,omitempty"` // enrollment ID (สำหรับ program)
	SourceType   string              `bson:"sourceType" json:"sourceType"`                         // "program" | "certificate"
	SourceID     primitive.ObjectID  `bson:"sourceId" json:"sourceId"`                             // ID ของ program/certificate ที่เป็นต้นเหตุ
}

// enum Status ของ HourChange (สำหรับ program และ certificate)
const (
	HCStatusPending       = "pending"       // ลงทะเบียนแล้ว - รอเข้าร่วมกิจกรรม | certificate รออนุมัติ
	HCStatusParticipating = "participating" // กำลังเข้าร่วมกิจกรรม (เช็คอิน/เช็คเอาท์ก่อนวันสุดท้าย)
	HCStatusVerifying     = "verifying"     // รอระบบดำเนินการตรวจสอบ (เช็คเอาท์วันสุดท้ายแล้ว)
	HCStatusWaitingForm   = "waiting_form"  // ยังไม่ได้ทำแบบฟอร์ม (เข้าร่วมครบแล้ว แต่ยังไม่ได้ส่งฟอร์ม)
	HCStatusAttended      = "attended"      // เข้าร่วมครบถ้วน + ทำฟอร์มแล้ว → ได้รับชั่วโมง
	HCStatusPartial       = "partial"       // เข้าร่วมไม่ครบถ้วน → ไม่ได้รับชั่วโมง
	HCStatusAbsent        = "absent"        // ไม่เข้าร่วม (ไม่ checkin ไม่ checkout เลย)
	HCStatusApproved      = "approved"      // อนุมัติแล้ว (certificate)
	HCStatusRejected      = "rejected"      // ปฏิเสธแล้ว (certificate)
)

// HourHistoryFilters ใช้เก็บค่าการกรองสำหรับ hour history
type HourHistoryFilters struct {
	StudentID  string `json:"studentId" query:"studentId"`   // Student ObjectID
	SourceType string `json:"sourceType" query:"sourceType"` // "program" | "certificate"
	Status     string `json:"status" query:"status"`         // Comma-separated statuses
	Search     string `json:"search" query:"search"`         // Search by title
}

// HourHistoryPaginatedResponse is a concrete type for paginated hour history responses
type HourHistoryPaginatedResponse struct {
	Data []HourChangeHistory `json:"data"`
	Meta PaginationMeta      `json:"meta"`
}
