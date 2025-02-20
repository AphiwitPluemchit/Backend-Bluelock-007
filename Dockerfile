# ใช้ golang image
FROM golang:1.24

# ติดตั้ง MongoDB Client (mongosh)
RUN apt-get update && \
    apt-get install -y wget gnupg && \
    wget -qO - https://www.mongodb.org/static/pgp/server-6.0.asc | tee /etc/apt/trusted.gpg.d/mongodb.asc && \
    echo "deb https://repo.mongodb.org/apt/debian bookworm/mongodb-org/6.0 main" | tee /etc/apt/sources.list.d/mongodb-org-6.0.list && \
    apt-get update && \
    apt-get install -y mongodb-mongosh  # ติดตั้ง mongosh แทน mongodb-org-shell

# ตั้งค่า working directory
WORKDIR /app

# คัดลอก go mod และ go sum
COPY go.mod go.sum ./ 

# ดาวน์โหลด dependencies
RUN go mod download

# คัดลอกไฟล์ทั้งหมดจาก src/
COPY src/ /app/src/

# สร้างโปรเจค
RUN go build -o main /app/src/main.go

# รันแอป
CMD ["/app/main"]
