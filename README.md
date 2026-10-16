# 🛠️ IT Helpdesk & Issue Tracking Web Application

ระบบเว็บแอปพลิเคชันแจ้งซ่อมและติดตามปัญหาอุปกรณ์ IT พัฒนาด้วยภาษา **Go (Golang)** ออกแบบด้วยสไตล์ **Tailwind CSS (Dark Mode UI)** พร้อมติดตั้งและปรับใช้งานจริงบน **Render.com** ด้วย **Docker**

---

## 🌟 ฟีเจอร์หลักของระบบ (Core Features)

1. **Authentication & Security:**
   * ระบบสมัครสมาชิกและล็อกอินเข้าใช้งาน
   * ป้องกันรหัสผ่านด้วยการเข้ารหัสแบบ `bcrypt`
   * การจัดการ Session ด้วย Secure Cookies (`HttpOnly`, `SameSite`)
2. **Role-Based Access Control (RBAC):**
   * **User (พนักงานทั่วไป):** แจ้งเรื่องซ่อม, ดูตั๋วของตนเอง, แก้ไขข้อมูลส่วนตัว
   * **Admin (เจ้าหน้าที่ IT):** ดูรายการตั๋วทั้งหมด, กรองสถานะตั๋ว, อัปเดตสถานะงาน, ลบตั๋วที่ไม่ถูกต้อง
3. **Data Ownership & Management:**
   * ผู้ใช้งานทั่วไปจะเห็นและจัดการได้เฉพาะตั๋วที่เป็นของตนเองเท่านั้น
   * มีฟอร์มป้อนข้อมูล 2 ฟอร์ม ได้แก่ **ฟอร์มแจ้งซ่อม (Ticket Creation)** และ **ฟอร์มปรับปรุงโปรไฟล์ (Profile Update)**
4. **Auto-Seed Default Admin:**
   * ระบบสร้างบัญชี Admin เริ่มต้นให้อัตโนมัติเมื่อเริ่มระบบครั้งแรก

---

## 🛠️ Tech Stack

* **Backend:** Go (Standard Library `net/http`)
* **ORM:** GORM
* **Database:** SQLite (CGO-Free via `glebarez/sqlite`)
* **Frontend:** HTML5, Tailwind CSS (CDN Integration)
* **Container & Deployment:** Docker, Render.com (Web Service)

---

## 🔑 บัญชีเข้าใช้งานระบบเริ่มต้น (Default Credentials)

* **Admin Account:**
  * **Email:** `admin@helpdesk.com`
  * **Password:** `admin1234`
* **User Account:** สามารถสมัครสมาชิกใหม่ได้ทันทีจากหน้าเว็บ

---

## 🚀 การติดตั้งและรันโปรเจกต์ภายในเครื่อง (Local Setup)

```bash
# 1. Clone repository
git clone [https://github.com/USERNAME/it-helpdesk.git](https://github.com/USERNAME/it-helpdesk.git)
cd it-helpdesk

# 2. Download Dependencies
go mod download

# 3. Run Application
go run main.go
