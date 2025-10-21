package programs

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hibiken/asynq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleNotifyOpenProgram(sender MailSender, registerURLBuilder func(programID string) string) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p NotifyOpenProgramPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}

		// 1) โหลด Program + Items
		prog, err := GetProgramByID(p.ProgramID)
		if err != nil || prog == nil {
			return fmt.Errorf("program not found: %s", p.ProgramID)
		}

		// 2) รวม majors + years
		majorsSet := map[string]struct{}{}
		yearsSet := map[int]struct{}{}
		for _, it := range prog.ProgramItems {
			for _, m := range it.Majors {
				if m != "" {
					majorsSet[m] = struct{}{}
				}
			}
			for _, y := range it.StudentYears {
				if y > 0 {
					yearsSet[y] = struct{}{}
				}
			}
		}
		if len(majorsSet) == 0 || len(yearsSet) == 0 {
			log.Println("notify-open: no majors/years, skip")
			return nil
		}

		majors := make([]string, 0, len(majorsSet))
		for k := range majorsSet {
			majors = append(majors, k)
		}
		years := make([]int, 0, len(yearsSet))
		for k := range yearsSet {
			years = append(years, k)
		}

		// 3) สร้าง prefix จากปีการศึกษา (เช่น 67,66,...)
		prefixes := GenerateStudentCodeFilter(years)
		if len(prefixes) == 0 {
			log.Println("notify-open: empty prefixes, skip")
			return nil
		}
		re := "^(" + strings.Join(prefixes, "|") + ")"
		if _, err := regexp.Compile(re); err != nil {
			return fmt.Errorf("bad student code regex: %s", re)
		}

		// (ทางเลือก) ถ้ากังวลเรื่องตัวพิมพ์ของ major ใน DB ไม่ตรง
		// ให้สร้าง $or เป็น regex ไม่สนตัวพิมพ์แทน:
		// ors := make([]bson.M, 0, len(majors))
		// for _, m := range majors {
		// 	ors = append(ors, bson.M{"major": bson.M{"$regex": "^" + regexp.QuoteMeta(m) + "$", "$options": "i"}})
		// }
		// majorCond := bson.M{"$or": ors}
		// ถ้าเชื่อว่าฟอร์แมตตรงกัน ใช้ $in ได้เลย:
		majorCond := bson.M{"$in": majors}

		// 4) query students ตาม major + code prefix
		match := bson.M{
			"major": majorCond,
			"code":  bson.M{"$regex": re},
		}

		// ✅ ดีบัก: นับจำนวนก่อนส่ง
		total, _ := DB.StudentCollection.CountDocuments(ctx, match)
		log.Printf("notify-open: matched students=%d majors=%v regex=%s", total, majors, re)
		if total == 0 {
			// ไม่ถือว่า error แต่อย่างน้อยรู้ว่าเงื่อนไขค้นหาไม่เจอใคร
			return nil
		}

		findOpts := options.Find().
			SetProjection(bson.M{"name": 1, "code": 1, "major": 1}). // ใช้ฟิลด์เท่าที่จำเป็น
			SetBatchSize(500)

		cur, err := DB.StudentCollection.Find(ctx, match, findOpts) // ← ชื่อคอลเลกชันแก้เป็น StudentsCollection
		if err != nil {
			return err
		}
		defer cur.Close(ctx)

		const emailDomain = "@go.buu.ac.th"

		send := func(s models.Student) {
			html, err := RenderOpenEmailHTML(OpenEmailData{
				StudentName:   s.Name,
				Major:         s.Major,
				ProgramName:   p.ProgramName,
				EndDateEnroll: prog.EndDateEnroll,
				RegisterLink:  registerURLBuilder(p.ProgramID),
				ProgramItems:  prog.ProgramItems,
			})
			if err != nil {
				log.Printf("render email failed: %v", err)
				return
			}
			to := s.Code + emailDomain
			subject := "เปิดลงทะเบียน: " + p.ProgramName
			if err := sender.Send(to, subject, html); err != nil {
				log.Printf("send mail failed to %s: %v", to, err)
			}
		}

		// 5) ส่งเป็น batch
		batch := 100
		buf := make([]models.Student, 0, batch)

		for cur.Next(ctx) {
			var st models.Student
			if err := cur.Decode(&st); err != nil {
				continue
			}
			buf = append(buf, st)
			if len(buf) >= batch {
				for _, x := range buf {
					send(x)
				}
				buf = buf[:0]
			}
		}
		for _, x := range buf {
			send(x)
		}

		log.Printf("notify-open done program=%s recipients majors=%v years=%v", p.ProgramID, majors, years)
		return nil
	}
}
