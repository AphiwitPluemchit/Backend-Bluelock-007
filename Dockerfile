# Stage 1: Build Golang Application
FROM golang:1.24.0 AS builder

# ตั้งค่า Working Directory ใน Container
WORKDIR /app

# คัดลอก go.mod และ go.sum ไปก่อนเพื่อลดเวลาการ build
COPY go.mod go.sum ./

# Download dependencies
RUN go mod tidy

# คัดลอกโค้ดทั้งหมดเข้า Container
COPY . .

# ติดตั้ง Swag CLI สำหรับสร้าง API Docs
RUN go install github.com/swaggo/swag/cmd/swag@latest

# สร้าง Swagger Docs
RUN swag init -g src/main.go

# คอมไพล์โปรเจกต์ Golang
RUN go build -o main src/main.go

# Stage 2: Run Application (ใช้ Image เล็กลง)
FROM golang:1.24.0

# ตั้งค่า Working Directory
WORKDIR /app

# คัดลอก Binary จาก Stage 1
COPY --from=builder /app/main .
COPY --from=builder /app/docs ./docs
# ✅ คัดลอก .env เข้า Container
COPY .env . 

# เปิด Port 8080
EXPOSE 8888

# รันแอป
CMD ["./main"]
