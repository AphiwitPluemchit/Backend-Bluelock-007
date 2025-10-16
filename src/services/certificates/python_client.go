package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type FastAPIResp struct {
	IsVerified    bool `json:"isVerified"`
	IsNameMatch   bool `json:"isNameMatch"`
	IsCourseMatch bool `json:"isCourseMatch"`
	// score fields can be null when optional values are not provided by the verifier
	NameScoreTh   *int  `json:"nameScoreTh"`
	NameScoreEn   *int  `json:"nameScoreEn"`
	CourseScore   *int  `json:"courseScore"`
	CourseScoreEn *int  `json:"courseScoreEn"`
	UsedOCR       *bool `json:"usedOcr"`
}

type buuPayload struct {
	HTML         string `json:"html"`
	StudentTH    string `json:"student_th,omitempty"`
	StudentEN    string `json:"student_en,omitempty"`
	CourseName   string `json:"course_name"`
	CourseNameEN string `json:"course_name_en,omitempty"`
}

func callBUUMoocFastAPI(fastAPIBase string, html, studentTH, studentEN, courseName string, courseNameEN string) (*FastAPIResp, error) {
	body := buuPayload{
		HTML:         html,
		StudentTH:    studentTH,
		StudentEN:    studentEN,
		CourseName:   courseName,
		CourseNameEN: courseNameEN,
	}
	b, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", fastAPIBase+"/buumooc", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("fastapi /buumooc returned status " + res.Status)
	}

	var out FastAPIResp
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func callThaiMoocFastAPI(fastAPIBase string, pdfData []byte, studentTH, studentEN, courseName, courseNameEN string) (*FastAPIResp, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	fw, _ := w.CreateFormFile("pdf", "certificate.pdf")
	if _, err := io.Copy(fw, bytes.NewReader(pdfData)); err != nil {
		return nil, err
	}
	_ = w.WriteField("student_th", studentTH)
	_ = w.WriteField("student_en", studentEN)
	_ = w.WriteField("course_name", courseName)
	_ = w.WriteField("course_name_en", courseNameEN)
	_ = w.Close()

	req, _ := http.NewRequest("POST", fastAPIBase+"/thaimooc", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	// ป้องกันบาง proxy/เซิร์ฟเวอร์ที่เกลียด keep-alive
	req.Close = true

	client := &http.Client{Timeout: 120 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// อ่านทั้งก้อนก่อน
	b, _ := io.ReadAll(res.Body)
	// fmt.Printf("[fastapi status] %s\n", res.Status)
	// fmt.Printf("[fastapi ctype ] %s\n", res.Header.Get("Content-Type"))
	// fmt.Printf("[fastapi raw   ] %s\n", string(b))

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fastapi /thaimooc %s — %s", res.Status, string(b))
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil, fmt.Errorf("fastapi /thaimooc returned empty body with %s", res.Status)
	}

	var out FastAPIResp
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("decode error: %w — body=%s", err, string(b))
	}
	return &out, nil
}
