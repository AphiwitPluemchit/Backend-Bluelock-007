package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type MinimalOK struct {
	Ok bool `json:"ok"`
}

type buuPayload struct {
	HTML       string `json:"html"`
	StudentTH  string `json:"student_th,omitempty"`
	StudentEN  string `json:"student_en,omitempty"`
	CourseName string `json:"course_name"`
}

func callBUUMoocFastAPI(fastAPIBase string, html, studentTH, studentEN, courseName string) (bool, error) {
	body := buuPayload{
		HTML:       html,
		StudentTH:  studentTH,
		StudentEN:  studentEN,
		CourseName: courseName,
	}
	b, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", fastAPIBase+"/buumooc", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return false, errors.New("fastapi /buumooc returned status " + res.Status)
	}

	var out MinimalOK
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return false, err
	}
	return out.Ok, nil
}

func callThaiMoocFastAPI(fastAPIBase, pdfPath, studentTH, studentEN, courseName string) (bool, error) {
	f, err := os.Open(pdfPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// file part
	fw, _ := w.CreateFormFile("pdf", filepath.Base(pdfPath))
	if _, err := io.Copy(fw, f); err != nil {
		return false, err
	}

	// fields
	_ = w.WriteField("student_th", studentTH)
	_ = w.WriteField("student_en", studentEN)
	_ = w.WriteField("course_name", courseName)

	_ = w.Close()

	req, _ := http.NewRequest("POST", fastAPIBase+"/thaimooc", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return false, errors.New("fastapi /thaimooc returned status " + res.Status)
	}

	var out MinimalOK
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return false, err
	}
	return out.Ok, nil
}
