# Stage 1: Build
FROM golang:1.26.1-alpine AS builder
WORKDIR /app

# คัดลอกไฟล์ go.mod และ main.go เข้าไป
COPY go.mod main.go ./

# สั่งให้ Go สแกนโค้ด โหลด Library ที่ขาดหาย และสร้าง go.sum ให้โดยอัตโนมัติ
RUN go mod tidy

# ทำการ Build
RUN CGO_ENABLED=0 GOOS=linux go build -o push-service main.go

# Stage 2: Production Image
FROM alpine:latest
# ติดตั้ง tzdata เผื่อต้องการจัดการ Timezone ใน Log
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

# สร้างโฟลเดอร์สำหรับทำ Volume Mount
RUN mkdir /app/data

# นำ Binary จาก builder มาใส่
COPY --from=builder /app/push-service .

EXPOSE 8080

# สั่งรัน
CMD ["./push-service"]