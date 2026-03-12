package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver (No CGO required)
)

var (
	db *sql.DB
	mu sync.Mutex // ตัวล็อกป้องกัน Race condition ตอนหา MAX(queue_index)

	// ตัวแปรสำหรับเก็บตั้งค่าจาก Environment
	targetURL string
	authToken string
)

func main() {
	// ดึงค่า URL ปลายทางจาก Environment Variable (ถ้าไม่มีใช้ค่า Default)
	targetURL = os.Getenv("NOTIPUSH_URL")
	if targetURL == "" {
		targetURL = "https://notipush.app/api/send"
	}

	// ดึงค่า Token สำหรับ Header
	authToken = os.Getenv("NOTIPUSH_TOKEN")

	initDB()
	defer db.Close()

	// รัน Worker 1 ตัวเพื่อคุม 2 TPS เด็ดขาด
	go queueWorker()

	http.HandleFunc("/send-push", enqueuePushHandler)

	fmt.Println("Push Microservice is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initDB() {
	var err error
	// เปิดไฟล์ SQLite พร้อมตั้งค่า WAL mode ให้ทำงานพร้อมกันได้ดีขึ้น และตั้ง timeout
	db, err = sql.Open("sqlite", "./data/queue.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		log.Fatal("Failed to connect to SQLite:", err)
	}

	// สร้างตาราง
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS push_queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		queue_index INTEGER NOT NULL,
		payload TEXT NOT NULL,
		is_sent INTEGER DEFAULT 0,
		retry_count INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_is_sent_index ON push_queue(is_sent, queue_index);
	`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	// อัปเดตตารางเดิม (กรณีมีไฟล์ queue.db เก่าอยู่แล้ว)
	db.Exec("ALTER TABLE push_queue ADD COLUMN retry_count INTEGER DEFAULT 0")
}

// API Handler สำหรับรับ Push จาก CI4
func enqueuePushHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// อ่าน Body ทั้งหมดที่ CI4 ส่งมาแบบ Raw
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// ตรวจสอบว่า Body เป็น JSON ที่ถูกต้องหรือไม่
	if !json.Valid(bodyBytes) {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	payloadStr := string(bodyBytes)

	// ล็อก Mutex ป้องกัน Request อื่นมาอ่านค่า MAX() ชนกัน
	mu.Lock()
	defer mu.Unlock()

	// 1. หาค่า MAX(queue_index) ที่ is_sent = 0
	var maxIdx int
	err = db.QueryRow("SELECT COALESCE(MAX(queue_index), 0) FROM push_queue WHERE is_sent = 0").Scan(&maxIdx)
	if err != nil {
		log.Printf("Error getting max index: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	newIdx := maxIdx + 1

	// 2. ค้นหาแถวที่ส่งไปแล้ว (is_sent = 1) เพื่อนำกลับมาใช้ใหม่
	var reusableID int
	err = db.QueryRow("SELECT id FROM push_queue WHERE is_sent = 1 LIMIT 1").Scan(&reusableID)

	if err == sql.ErrNoRows {
		// ถ้าไม่มี is_sent = 1 ให้ INSERT ใหม่ (รีเซ็ต retry_count เป็น 0)
		_, err = db.Exec("INSERT INTO push_queue (queue_index, payload, is_sent, retry_count) VALUES (?, ?, 0, 0)", newIdx, payloadStr)
		if err != nil {
			log.Printf("Insert error: %v", err)
		}
	} else if err == nil {
		// ถ้ามี is_sent = 1 ให้ UPDATE ข้อมูลทับแถวเดิม (รีเซ็ต retry_count เป็น 0)
		_, err = db.Exec("UPDATE push_queue SET queue_index = ?, payload = ?, is_sent = 0, retry_count = 0 WHERE id = ?", newIdx, payloadStr, reusableID)
		if err != nil {
			log.Printf("Update error: %v", err)
		}
	} else {
		log.Printf("Select reusable row error: %v", err)
	}

	// ตอบกลับ CI4 ทันที ไม่ต้องรอส่ง
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprint(w, `{"status":"queued"}`)
}

// Worker ดึงคิวไปส่ง
func queueWorker() {
	for {
		var id, queueIndex, retryCount int
		var payloadStr string

		// ดึงคิวที่เก่าที่สุด (queue_index น้อยสุด) ที่ยังไม่ได้ส่ง พร้อมดึง retry_count
		err := db.QueryRow("SELECT id, queue_index, payload, retry_count FROM push_queue WHERE is_sent = 0 ORDER BY queue_index ASC LIMIT 1").Scan(&id, &queueIndex, &payloadStr, &retryCount)

		if err == sql.ErrNoRows {
			// ถ้าไม่มีคิว ให้พัก 1 วินาทีแล้วหาใหม่
			time.Sleep(1 * time.Second)
			continue
		} else if err != nil {
			log.Printf("Worker select error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// ทำการส่ง Raw JSON Payload ไปยัง Server ปลายทาง
		// คืนค่าเป็นตัวเลข: 1 = Success, 2 = Drop (4xx), 3 = Retry (5xx/Timeout)
		status := sendPushToRealServer(payloadStr)

		if status == 1 {
			// ส่งสำเร็จ อัปเดตเป็น is_sent = 1
			db.Exec("UPDATE push_queue SET is_sent = 1 WHERE id = ?", id)
			log.Printf("[SUCCESS] Sent push ID: %d", id)
			
		} else if status == 2 {
			// ส่งไม่ผ่านเพราะ Error ฝั่ง Client (เช่น 422 ข้อความซ้ำ) ให้ Drop ทิ้งทันที
			db.Exec("UPDATE push_queue SET is_sent = 1 WHERE id = ?", id)
			log.Printf("[DROPPED] Push ID: %d dropped due to client/validation error.", id)
			
		} else {
			// กรณี status == 3 (Server Error / Timeout) ให้นำไป Retry
			retryCount++
			if retryCount >= 3 {
				db.Exec("UPDATE push_queue SET is_sent = 1 WHERE id = ?", id)
				log.Printf("[DROPPED] Push ID: %d reached max retries (3) and was dropped.", id)
			} else {
				mu.Lock()
				var currentMaxIdx int
				db.QueryRow("SELECT COALESCE(MAX(queue_index), 0) FROM push_queue WHERE is_sent = 0").Scan(&currentMaxIdx)
				db.Exec("UPDATE push_queue SET queue_index = ?, retry_count = ? WHERE id = ?", currentMaxIdx+1, retryCount, id)
				mu.Unlock()
				log.Printf("[FAILED] Re-queued push ID: %d to index: %d (Retry %d/3)", id, currentMaxIdx+1, retryCount)
			}
		}

		// คุม TPS: 2 TPS = 1 งานใช้เวลาหน่วง 0.5 วินาที
		time.Sleep(500 * time.Millisecond)
	}
}

// ฟังก์ชันยิง HTTP POST Request ไปยัง NotiPush Server
// คืนค่า: 1 = Success, 2 = Drop, 3 = Retry
func sendPushToRealServer(payloadStr string) int {
	req, err := http.NewRequest("POST", targetURL, strings.NewReader(payloadStr))
	if err != nil {
		log.Printf("[ERROR] Failed to create request: %v", err)
		return 3 // สร้าง Request ไม่ได้ ให้ถือเป็น Network Error (Retry)
	}

	req.Header.Set("Content-Type", "application/json")
	
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	// ตั้ง Timeout 10 วิ ป้องกันคิวค้าง
	client := &http.Client{Timeout: 10 * time.Second}
	
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] Network error: %v", err)
		return 3 // ส่งไม่สำเร็จ (Timeout/ไม่มีเน็ต) ให้ Retry
	}
	defer resp.Body.Close()

	// HTTP Status 200-299 ถือว่าส่งสำเร็จ
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return 1 // Success
	}

	// HTTP Status 400-499 เป็น Error ฝั่ง Client (เช่น 422 ข้อความซ้ำ, 400 ข้อมูลผิด) ไม่ควร Retry
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		log.Printf("[WARNING] NotiPush rejected payload (Status: %d).", resp.StatusCode)
		return 2 // Drop ทิ้ง
	}

	// HTTP Status 500 ขึ้นไป เป็น Server Error (ปลายทางพังชั่วคราว) สมควร Retry
	log.Printf("[WARNING] NotiPush server error (Status: %d).", resp.StatusCode)
	return 3 // Retry
}