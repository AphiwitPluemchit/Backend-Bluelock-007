# Stage 1: Build Golang Application
FROM golang:1.24.0 AS builder

# ตั้งค่า Working Directory ใน Container
WORKDIR /app

# คัดลอก go.mod และ go.sum ไปก่อนเพื่อลดเวลาการ build
COPY go.mod go.sum ./

# Download dependencies (ใช้ go mod download หรือ go mod tidy ตามความเหมาะสม)
RUN go mod tidy

# คัดลอกโค้ดทั้งหมดเข้า Container
COPY . .


RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN swag init -g src/main.go


RUN CGO_ENABLED=0 GOOS=linux go build -o main ./src/main.go


FROM debian:bookworm-slim

# ตั้งค่า Timezone และติดตั้ง Chromium runtime ที่ chromedp ต้องการ
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
    ca-certificates \
    fonts-liberation \
    libc6 \
    libasound2 \
    libatk-bridge2.0-0 \
    libatk1.0-0 \
    libgbm1 \
    libgtk-3-0 \
    libx11-6 \
    libx11-xcb1 \
    libxcomposite1 \
    libxdamage1 \
    libxrandr2 \
    tzdata \
    wget \
    chromium \
    && rm -rf /var/lib/apt/lists/*

# ตั้งค่า Working Directory
WORKDIR /app

# คัดลอก Binary และ assets จาก Stage 1
COPY --from=builder /app/main .
COPY --from=builder /app/docs ./docs

# ให้ chromedp หา binary ได้เมื่อเรียก `google-chrome`
RUN ln -sf /usr/bin/chromium /usr/bin/google-chrome || true

# ตัวแปรแนะนำสำหรับ library ที่เรียกหา
ENV CHROME_BIN=/usr/bin/chromium \
    CHROME_PATH=/usr/bin/chromium

EXPOSE 8888

# รันแอป
CMD ["./main"]