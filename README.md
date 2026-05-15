# NotiPush Queue Microservice

[🇹🇭 อ่านภาษาไทย (Read in Thai)](#-thai-version)

A lightweight, high-performance microservice written in Go (Golang). It acts as a local relay queue to forward push notifications to the NotiPush API (or any other API) with a configurable **TPS (Transactions Per Second)** rate limit via `TPS` in `.env` or the process environment (default: `2`).

## 📌 Core Features (Facts & Architecture)

* **Configurable TPS Rate Limiting:** Enforces the request rate using the `TPS` environment variable (default: `2`) with Go's background worker and precise `time.Sleep`.
* **Persistent Queue:** Uses SQLite (`modernc.org/sqlite` - pure Go, no CGO required) to store the queue. Prevents data loss during unexpected container restarts or crashes.
* **Database Optimization (Row Recycling):** Reuses rows where `is_sent = 1` instead of inserting new ones indefinitely. This Object Pooling concept guarantees the `.db` file size will not bloat over time.
* **Smart Retry Mechanism:**
* **Immediate Drop:** Drops the payload immediately on **4xx Client Errors** (Except 400) like `422 Unprocessable Entity` for duplicate messages.
* **Delayed Retry (5 Minutes):** Specifically for `400 Bad Request` errors, the payload is pushed back to the queue and paused for 5 minutes before retrying (Up to 3 times) to prevent spamming the endpoint.
* **Fast Retry:** Retries up to **3 times** on **5xx Server Errors** or Network Timeouts before dropping the payload.


* **Extremely Lightweight:** Compiled as a single static binary inside an Alpine Docker image (~15MB total size).

## 🔒 Security Features

* **Bearer or IP Access Control:** Accepts inbound requests when the caller sends a valid bearer token in `Authorization: Bearer ...` or when the source IP matches `ALLOWED_IPS`
* **Inbound Bearer Token:** Configure `INBOUND_BEARER_TOKEN` for callers that run behind dynamic IP addresses and cannot be safely allowlisted
* **Separate Inbound/Outbound Tokens:** `INBOUND_BEARER_TOKEN` protects `POST /send-push`, while `NOTIPUSH_TOKEN` is forwarded to the downstream NotiPush API as an outbound `Authorization` header
* **CIDR Subnet Support:** Supports network ranges using CIDR notation (e.g., `192.168.1.0/24`, `10.99.0.0/16`) for flexible network access control
* **Proxy-aware IP Detection:** Supports `X-Forwarded-For` and `X-Real-IP` headers to correctly identify client IPs behind load balancers or proxies
* **Flexible IP Configuration:** Supports individual IPs, hostnames (localhost), IPv6 addresses, and CIDR subnets with comma-separated values as a backward-compatible fallback

## 🛠 Prerequisites

* Docker and Docker Compose

## 🚀 Getting Started

1. Clone the repository.
2. Update your `.env` or `docker-compose.yml` values before starting the container:
* `NOTIPUSH_URL`: Downstream API endpoint (Current default: `https://notipush.app/api/send-push`)
* `NOTIPUSH_PORT`: Port exposed by the microservice. The example `.env` in this repository uses `8280`
* `TPS`: Request rate limit for outbound delivery. The service reads this from `.env` or the process environment, falls back to `2` if unset or invalid, and the example `.env` in this repository uses `10`
* `INBOUND_BEARER_TOKEN`: Bearer token for inbound requests to `POST /send-push`. Use a long random secret when the caller has a dynamic IP
* `ALLOWED_IPS`: Comma-separated allowlist for fallback access when the caller does not send a bearer token. Supports single IPs, hostnames, IPv6, and CIDR blocks
* `NOTIPUSH_TOKEN`: Optional outbound `Authorization` header sent from this microservice to the downstream NotiPush API
* `DEBUG`: Set to `true` to log incoming payloads for troubleshooting

3. Recommended auth setup:
* Keep `INBOUND_BEARER_TOKEN` set for callers behind dynamic IPs
* Keep `ALLOWED_IPS` for trusted fixed-IP callers or as a migration fallback
* Avoid committing real secrets from `.env` into version control


4. Start the service:
```bash
docker compose up -d --build

```


The Compose setup mounts `.env` into the container so runtime settings like `TPS` are read from the same file you edit locally.


*(Note: The SQLite database file will be safely persisted in the `./push_data` folder on your host machine).*

5. Verify locally:
```bash
curl -X POST http://localhost:8280/send-push \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer YOUR_INBOUND_BEARER_TOKEN" \
    -d '{"title":"hello","body":"test"}'
```

## 📡 API Usage

The microservice exposes one internal endpoint. Send your raw JSON payload here, and it will be queued immediately.

**Endpoint:** `POST http://localhost:8280/send-push` when using the example `.env` in this repository, or `POST http://localhost:<NOTIPUSH_PORT>/send-push` for your own configuration.

Inbound authentication accepts either of these:

* `Authorization: Bearer <INBOUND_BEARER_TOKEN>`
* A source IP/hostname/subnet that matches `ALLOWED_IPS`

Authorization behavior is:

* Valid bearer token: request is accepted even if the caller IP changes
* No bearer token but IP is allowlisted: request is accepted
* Invalid bearer token and IP not allowlisted: request is rejected with `401 Unauthorized`

**Example cURL:**

```bash
curl -X POST http://localhost:8280/send-push \
-H "Content-Type: application/json" \
-H "Authorization: Bearer YOUR_INBOUND_BEARER_TOKEN" \
-d '{
    "token": "YOUR_NOTIPUSH_API_TOKEN",
    "device_token": "your_device_token_here",
    "title": "Hello World",
    "body": "This message uses the configured TPS limit"
}'

```

**Response:**

```json
{"status":"queued"}

```

If authorization fails, the service responds with `401 Unauthorized` and returns a `WWW-Authenticate: Bearer realm="send-push"` header.

## CI

GitHub Actions is configured in `.github/workflows/ci.yml` to run on every push and pull request:

* `gofmt -l .` to catch formatting drift
* `go test ./...`
* `go build ./...`

---

<a name="-thai-version"></a>

# 🇹🇭 Thai Version

ไมโครเซอร์วิสขนาดเล็กและทำงานได้รวดเร็ว เขียนด้วยภาษา Go (Golang) ทำหน้าที่เป็นคิวตัวกลาง (Relay Queue) เพื่อส่ง Push Notification ไปยัง NotiPush API (หรือประยุกต์ใช้กับ API อื่นๆ ได้) โดยมีการควบคุมอัตราการส่งแบบกำหนดค่าได้ผ่าน `TPS` จาก `.env` หรือ process environment (ค่าเริ่มต้น `2`)

## 📌 คุณสมบัติทางวิศวกรรมของระบบ

* **คุม TPS ได้จาก Environment:** ควบคุมอัตราการส่งตามค่า `TPS` (ค่าเริ่มต้น `2`) โดยใช้ Worker และการหน่วงเวลาของ Go
* **คิวไม่สูญหายเมื่อไฟดับ:** ใช้ SQLite (`modernc.org/sqlite` - แบบ Pure Go ไม่ต้องพึ่งพา CGO) บันทึกคิวลงดิสก์ ข้อมูลที่รอส่งจะไม่หายไปแม้ Docker Container จะถูกลบหรือรีสตาร์ท
* **ฐานข้อมูลไม่บวม (Row Recycling):** ใช้หลักการ Object Pooling โดยนำแถวข้อมูลที่ส่งสำเร็จแล้ว (`is_sent = 1`) กลับมาเขียนทับใหม่แทนการเพิ่มแถวใหม่ไปเรื่อยๆ ทำให้ขนาดไฟล์ `.db` คงที่และไม่กินพื้นที่ดิสก์
* **ระบบ Retry อัจฉริยะแบบแยกประเภท:**
* **ทิ้งทันที (Drop):** หากตอบกลับเป็น **4xx Error (ยกเว้น 400)** เช่น `422` ส่งข้อความซ้ำ ระบบจะไม่นำมาส่งซ้ำให้เสียโควต้า
* **หน่วงเวลา 5 นาที (Delayed Retry):** เฉพาะกรณีเกิด Error `400` ระบบจะนำคิวไปต่อท้ายและตั้งเวลาล็อกไว้ไม่ให้ดึงมาทำซ้ำภายใน 5 นาที (สูงสุด 3 ครั้ง)
* **ลองใหม่ (Fast Retry):** หากเป็น **5xx Error หรือเน็ตเวิร์กพัง** ระบบจะนำคิวไปต่อท้ายเพื่อรอส่งใหม่ทันที (สูงสุด 3 ครั้ง)


* **เบาและกินทรัพยากรน้อยมาก:** คอมไพล์เป็น Binary ไฟล์เดียว รันบน Alpine Linux Image ขนาดเพียงประมาณ 15MB

## 🔒 ความปลอดภัย

* **ควบคุมการเข้าถึงแบบ Bearer หรือ IP:** อนุญาตให้ส่ง Push ได้หากมี `Authorization: Bearer ...` ที่ตรงกับ `INBOUND_BEARER_TOKEN` หรือมี IP/Hostname/Subnet ตรงกับ `ALLOWED_IPS`
* **รองรับ Inbound Bearer Token:** ใช้ `INBOUND_BEARER_TOKEN` สำหรับเครื่องต้นทางที่มี Dynamic IP และไม่สะดวก allowlist IP
* **แยก Token ขาเข้าและขาออกชัดเจน:** `INBOUND_BEARER_TOKEN` ใช้ป้องกัน `POST /send-push` ส่วน `NOTIPUSH_TOKEN` ใช้เป็น `Authorization` header ตอน service นี้ยิงต่อไปยัง NotiPush ปลายทาง
* **รองรับ Subnet แบบ CIDR:** รองรับช่วงเครือข่ายโดยใช้ CIDR notation (เช่น `192.168.1.0/24`, `10.99.0.0/16`) สำหรับควบคุมการเข้าถึงแบบยืดหยุ่น
* **ตรวจจับ IP ข้างหลัง Proxy:** รองรับ Header `X-Forwarded-For` และ `X-Real-IP` เพื่อระบุ Client IP ที่แท้จริงเมื่ออยู่หลัง Load Balancer หรือ Proxy
* **การตั้งค่า IP ยืดหยุ่น:** รองรับทั้ง IPv4, IPv6, Hostname (localhost) และ CIDR Subnets โดยคั่นรายการด้วย comma และยังใช้เป็น fallback ได้

## 🛠 สิ่งที่ต้องติดตั้งไว้ก่อน

* Docker และ Docker Compose

## 🚀 วิธีการรันระบบ

1. Clone โปรเจกต์นี้
2. แก้ค่าที่ `.env` หรือ `docker-compose.yml` ให้ตรงกับระบบจริงก่อนรัน
* `NOTIPUSH_URL`: URL ของ API ปลายทาง (ค่าปัจจุบัน: `https://notipush.app/api/send-push`)
* `NOTIPUSH_PORT`: Port ที่ต้องการให้ Microservice รัน โดยตัวอย่าง `.env` ใน repo นี้ใช้ `8280`
* `TPS`: อัตราการยิง request ขาออกต่อวินาที ระบบจะอ่านจาก `.env` หรือ process environment โดยตัวอย่าง `.env` ใน repo นี้ใช้ `10` และจะ fallback เป็น `2` หากค่าไม่ถูกต้องหรือไม่ได้ตั้ง
* `INBOUND_BEARER_TOKEN`: Bearer token สำหรับ request ที่ยิงเข้า `POST /send-push` แนะนำให้ตั้งเป็นค่ายาวและสุ่มเมื่อ client ใช้ Dynamic IP
* `ALLOWED_IPS`: รายการ IP/Hostname/Subnet ที่อนุญาตให้ส่ง Push คั่นด้วย comma รองรับ CIDR notation และยังใช้เป็น fallback ได้
* `NOTIPUSH_TOKEN`: (Optional) Token สำหรับขาออก ถ้า API ปลายทางบังคับให้ใส่ `Authorization` header
* `DEBUG`: ตั้งเป็น `true` เมื่อต้องการ log payload สำหรับ debug

3. แนวทางการตั้งค่า auth ที่แนะนำ:
* ตั้ง `INBOUND_BEARER_TOKEN` ไว้สำหรับ client ที่ IP เปลี่ยนบ่อย
* เก็บ `ALLOWED_IPS` ไว้สำหรับเครื่องที่มี IP คงที่ หรือใช้เป็น fallback ช่วง migration
* หลีกเลี่ยงการ commit secret จริงจาก `.env` เข้า git

4. สั่งรันระบบด้วยคำสั่ง:
```bash
docker compose up -d --build

```


Compose ใน repo นี้จะ mount ไฟล์ `.env` เข้าไปใน container ด้วย เพื่อให้ runtime settings อย่าง `TPS` อ่านจากไฟล์เดียวกับที่แก้ในเครื่องได้ตรง ๆ


*(หมายเหตุ: ไฟล์ฐานข้อมูล SQLite จะถูกบันทึกไว้อย่างปลอดภัยที่โฟลเดอร์ `./push_data` ในเครื่องโฮสต์)*

5. ทดสอบแบบเร็วในเครื่อง:
```bash
curl -X POST http://localhost:8280/send-push \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer YOUR_INBOUND_BEARER_TOKEN" \
    -d '{"title":"hello","body":"test"}'
```

## 📡 วิธีการใช้งาน API

Microservice ตัวนี้เปิดรับ Request เพียง 1 Endpoint คุณสามารถส่ง JSON รูปแบบใดก็ได้เข้ามา ระบบจะนำไปเข้าคิวและส่งต่อให้ตามรูปแบบนั้น 100%

**Endpoint:** `POST http://localhost:8280/send-push` หากใช้ `.env` ตัวอย่างใน repo นี้ หรือ `POST http://localhost:<NOTIPUSH_PORT>/send-push` ตามค่าที่คุณตั้งเอง

การยืนยันตัวตนฝั่ง inbound ใช้ได้ 2 แบบ:

* ใส่ `Authorization: Bearer <INBOUND_BEARER_TOKEN>`
* ยิงมาจาก IP/Hostname/Subnet ที่อยู่ใน `ALLOWED_IPS`

พฤติกรรมการอนุญาตมีดังนี้:

* Bearer token ถูกต้อง: อนุญาต แม้ IP ต้นทางจะเปลี่ยน
* ไม่มี bearer token แต่ IP อยู่ใน allowlist: อนุญาต
* Bearer token ไม่ถูกต้องและ IP ไม่อยู่ใน allowlist: ปฏิเสธด้วย `401 Unauthorized`

**ตัวอย่างการยิงด้วย cURL:**

```bash
curl -X POST http://localhost:8280/send-push \
-H "Content-Type: application/json" \
-H "Authorization: Bearer YOUR_INBOUND_BEARER_TOKEN" \
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

ถ้า auth ไม่ผ่าน ระบบจะตอบ `401 Unauthorized` และส่ง header `WWW-Authenticate: Bearer realm="send-push"` กลับไป

## CI

มี GitHub Actions ที่ `.github/workflows/ci.yml` ให้รันอัตโนมัติทุกครั้งที่ push หรือเปิด pull request:

* `gofmt -l .` สำหรับจับไฟล์ที่ format ไม่ตรง
* `go test ./...`
* `go build ./...`