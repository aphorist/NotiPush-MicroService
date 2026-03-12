# NotiPush Queue Microservice

[🇹🇭 อ่านภาษาไทย (Read in Thai)](#-thai-version)

A lightweight, high-performance microservice written in Go (Golang). It acts as a local relay queue to forward push notifications to the NotiPush API (or any other API) with a strict **2 TPS (Transactions Per Second)** rate limit.

## 📌 Core Features (Facts & Architecture)

* **Strict 2 TPS Rate Limiting:** Enforces a hard limit of 2 requests per second using Go's background worker and precise `time.Sleep`.
* **Persistent Queue:** Uses SQLite (`modernc.org/sqlite` - pure Go, no CGO required) to store the queue. Prevents data loss during unexpected container restarts or crashes.
* **Database Optimization (Row Recycling):** Reuses rows where `is_sent = 1` instead of inserting new ones indefinitely. This Object Pooling concept guarantees the `.db` file size will not bloat over time.
* **Smart Retry Mechanism:**
* **Immediate Drop:** Drops the payload immediately on **4xx Client Errors** (Except 400) like `422 Unprocessable Entity` for duplicate messages.
* **Delayed Retry (5 Minutes):** Specifically for `400 Bad Request` errors, the payload is pushed back to the queue and paused for 5 minutes before retrying (Up to 3 times) to prevent spamming the endpoint.
* **Fast Retry:** Retries up to **3 times** on **5xx Server Errors** or Network Timeouts before dropping the payload.


* **Extremely Lightweight:** Compiled as a single static binary inside an Alpine Docker image (~15MB total size).

## 🛠 Prerequisites

* Docker and Docker Compose

## 🚀 Getting Started

1. Clone the repository.
2. Review the `docker-compose.yml` file and adjust the Environment Variables if necessary:
* `NOTIPUSH_URL`: The target API endpoint (Default: `https://notipush.app/api/send`)
* `NOTIPUSH_PORT`: The port for the microservice to listen on (Default: `8880`)
* `NOTIPUSH_TOKEN`: (Optional) Your API token. If provided, the microservice will automatically inject this as an `Authorization` header.


3. Start the service:
```bash
docker-compose up -d --build

```


*(Note: The SQLite database file will be safely persisted in the `./push_data` folder on your host machine).*

## 📡 API Usage

The microservice exposes one internal endpoint. Send your raw JSON payload here, and it will be queued immediately.

**Endpoint:** `POST http://localhost:8880/send-push` (or your configured `NOTIPUSH_PORT`)

**Example cURL:**

```bash
curl -X POST http://localhost:8880/send-push \
-H "Content-Type: application/json" \
-d '{
    "token": "YOUR_NOTIPUSH_API_TOKEN",
    "device_token": "your_device_token_here",
    "title": "Hello World",
    "body": "This message is rate-limited to 2 TPS"
}'

```

**Response:**

```json
{"status":"queued"}

```

---

<a name="-thai-version"></a>

# 🇹🇭 Thai Version

ไมโครเซอร์วิสขนาดเล็กและทำงานได้รวดเร็ว เขียนด้วยภาษา Go (Golang) ทำหน้าที่เป็นคิวตัวกลาง (Relay Queue) เพื่อส่ง Push Notification ไปยัง NotiPush API (หรือประยุกต์ใช้กับ API อื่นๆ ได้) โดยมีการควบคุมอัตราการส่งที่ **2 TPS (Transactions Per Second)** อย่างเด็ดขาด

## 📌 คุณสมบัติทางวิศวกรรมของระบบ

* **คุม 2 TPS เด็ดขาด:** ควบคุมอัตราการส่งไม่ให้เกิน 2 ครั้งต่อวินาทีโดยใช้ Worker และการหน่วงเวลาของ Go
* **คิวไม่สูญหายเมื่อไฟดับ:** ใช้ SQLite (`modernc.org/sqlite` - แบบ Pure Go ไม่ต้องพึ่งพา CGO) บันทึกคิวลงดิสก์ ข้อมูลที่รอส่งจะไม่หายไปแม้ Docker Container จะถูกลบหรือรีสตาร์ท
* **ฐานข้อมูลไม่บวม (Row Recycling):** ใช้หลักการ Object Pooling โดยนำแถวข้อมูลที่ส่งสำเร็จแล้ว (`is_sent = 1`) กลับมาเขียนทับใหม่แทนการเพิ่มแถวใหม่ไปเรื่อยๆ ทำให้ขนาดไฟล์ `.db` คงที่และไม่กินพื้นที่ดิสก์
* **ระบบ Retry อัจฉริยะแบบแยกประเภท:**
* **ทิ้งทันที (Drop):** หากตอบกลับเป็น **4xx Error (ยกเว้น 400)** เช่น `422` ส่งข้อความซ้ำ ระบบจะไม่นำมาส่งซ้ำให้เสียโควต้า
* **หน่วงเวลา 5 นาที (Delayed Retry):** เฉพาะกรณีเกิด Error `400` ระบบจะนำคิวไปต่อท้ายและตั้งเวลาล็อกไว้ไม่ให้ดึงมาทำซ้ำภายใน 5 นาที (สูงสุด 3 ครั้ง)
* **ลองใหม่ (Fast Retry):** หากเป็น **5xx Error หรือเน็ตเวิร์กพัง** ระบบจะนำคิวไปต่อท้ายเพื่อรอส่งใหม่ทันที (สูงสุด 3 ครั้ง)


* **เบาและกินทรัพยากรน้อยมาก:** คอมไพล์เป็น Binary ไฟล์เดียว รันบน Alpine Linux Image ขนาดเพียงประมาณ 15MB

## 🛠 สิ่งที่ต้องติดตั้งไว้ก่อน

* Docker และ Docker Compose

## 🚀 วิธีการรันระบบ

1. Clone โปรเจกต์นี้
2. ตรวจสอบไฟล์ `docker-compose.yml` เพื่อตั้งค่า Environment Variables ที่จำเป็น
* `NOTIPUSH_URL`: URL ของ API ปลายทาง (ค่า Default: `https://notipush.app/api/send`)
* `NOTIPUSH_PORT`: Port ที่ต้องการให้ Microservice รัน (ค่า Default: `8880`)
* `NOTIPUSH_TOKEN`: (Optional) Token สำหรับยืนยันตัวตน ถ้ามีจะถูกใส่ใน Header อัตโนมัติ
3. สั่งรันระบบด้วยคำสั่ง:
```bash
docker-compose up -d --build

```


*(หมายเหตุ: ไฟล์ฐานข้อมูล SQLite จะถูกบันทึกไว้อย่างปลอดภัยที่โฟลเดอร์ `./push_data` ในเครื่องโฮสต์)*

## 📡 วิธีการใช้งาน API

Microservice ตัวนี้เปิดรับ Request เพียง 1 Endpoint คุณสามารถส่ง JSON รูปแบบใดก็ได้เข้ามา ระบบจะนำไปเข้าคิวและส่งต่อให้ตามรูปแบบนั้น 100%

**Endpoint:** `POST http://localhost:8880/send-push` (or your configured `NOTIPUSH_PORT`)

**ตัวอย่างการยิงด้วย cURL:**

```bash
curl -X POST http://localhost:8880/send-push \
-H "Content-Type: application/json" \
-d '{
    "token": "YOUR_NOTIPUSH_API_TOKEN",
    "device_token": "your_device_token_here",
    "title": "Hello World",
    "body": "This message is rate-limited to 2 TPS"
}'

```

**Response:**

```json
{"status":"queued"}

```