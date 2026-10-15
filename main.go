package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ==========================================
// 1. DATABASE MODELS
// ==========================================

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Email      string    `gorm:"unique;not null" json:"email"`
	Password   string    `gorm:"not null" json:"-"`
	Role       Role      `gorm:"default:'user'" json:"role"`
	FullName   string    `json:"full_name"`  // ฟอร์ม 2: Profile Info
	Department string    `json:"department"` // ฟอร์ม 2: Profile Info
	CreatedAt  time.Time `json:"created_at"`
	Tickets    []Ticket  `gorm:"foreignKey:UserID"`
}

type Ticket struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	DeviceType  string    `json:"device_type"`
	Status      string    `gorm:"default:'Pending'" json:"status"` // Pending, In-Progress, Resolved
	UserID      uint      `gorm:"not null" json:"user_id"`         // Data Ownership Check
	User        User      `gorm:"foreignKey:UserID"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var db *gorm.DB
var sessions = make(map[string]uint)

// ==========================================
// 2. TEMPLATES (Tailwind CSS Dark Theme)
// ==========================================

var tpl = template.Must(template.New("layout").Parse(`
<!DOCTYPE html>
<html lang="th">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>IT Helpdesk Support System</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-slate-950 text-slate-100 min-h-screen font-sans">
    <!-- Navbar -->
    <nav class="bg-slate-900 border-b border-slate-800 px-6 py-4 flex flex-wrap justify-between items-center shadow-xl">
        <div class="flex items-center space-x-3">
            <div class="w-10 h-10 bg-cyan-500/10 rounded-xl flex items-center justify-center border border-cyan-500/30 text-cyan-400 font-bold text-xl">🛠️</div>
            <div>
                <span class="text-xl font-extrabold text-cyan-400 tracking-wider block leading-none">IT HELPDESK</span>
                <span class="text-xs text-slate-400">Support & Ticket Management</span>
            </div>
        </div>
        <div class="flex items-center space-x-6 mt-3 sm:mt-0 font-medium text-sm">
            <a href="/tickets" class="hover:text-cyan-400 transition">ตั๋วแจ้งซ่อมของฉัน</a>
            <a href="/tickets/new" class="text-cyan-400 hover:text-cyan-300 font-semibold bg-cyan-500/10 px-3 py-1.5 rounded-lg border border-cyan-500/20 transition">+ แจ้งซ่อมใหม่ (ฟอร์ม 1)</a>
            <a href="/profile" class="hover:text-cyan-400 transition">ข้อมูลส่วนตัว (ฟอร์ม 2)</a>
            <a href="/admin" class="text-amber-400 hover:text-amber-300 font-bold transition">Admin Panel</a>
            <a href="/logout" class="text-rose-400 hover:text-rose-300 bg-rose-500/10 px-3 py-1.5 rounded-lg border border-rose-500/20 transition">ออกจากระบบ</a>
        </div>
    </nav>

    <!-- Main Container -->
    <div class="max-w-6xl mx-auto px-4 py-8">
        {{.}}
    </div>
</body>
</html>
`))

// ==========================================
// 3. MIDDLEWARES & AUTHENTICATION
// ==========================================

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		userID, exists := sessions[cookie.Value]
		if !exists {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		var user User
		if err := db.First(&user, userID).Error; err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value("user").(User)
		if user.Role != RoleAdmin {
			http.Error(w, "Forbidden: เฉพาะ Admin เท่านั้นที่เข้าถึงหน้านี้ได้", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ==========================================
// 4. HANDLERS
// ==========================================

func RegisterPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html lang="th"><head><meta charset="UTF-8"><script src="https://cdn.tailwindcss.com"></script></head>
		<body class="bg-slate-950 text-slate-100 flex items-center justify-center min-h-screen">
			<div class="bg-slate-900 border border-slate-800 p-8 rounded-2xl shadow-2xl w-full max-w-md">
				<h2 class="text-3xl font-extrabold text-center text-cyan-400 mb-6">สมัครสมาชิก IT Support</h2>
				<form method="POST" class="space-y-4">
					<div><label class="block text-sm mb-1 text-slate-400">อีเมล</label><input type="email" name="email" required class="w-full bg-slate-800 border border-slate-700 rounded-lg px-4 py-2.5 focus:outline-none focus:border-cyan-500"></div>
					<div><label class="block text-sm mb-1 text-slate-400">รหัสผ่าน</label><input type="password" name="password" required class="w-full bg-slate-800 border border-slate-700 rounded-lg px-4 py-2.5 focus:outline-none focus:border-cyan-500"></div>
					<div><label class="block text-sm mb-1 text-slate-400">ชื่อ-นามสกุล</label><input type="text" name="full_name" required class="w-full bg-slate-800 border border-slate-700 rounded-lg px-4 py-2.5 focus:outline-none focus:border-cyan-500"></div>
					<button type="submit" class="w-full bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-bold py-3 rounded-lg shadow-lg shadow-cyan-500/20 transition">ลงทะเบียน</button>
				</form>
				<p class="text-center text-sm text-slate-400 mt-4">มีบัญชีอยู่แล้ว? <a href="/login" class="text-cyan-400 hover:underline">เข้าสู่ระบบ</a></p>
			</div>
		</body></html>`)
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(r.FormValue("password")), 10)
	user := User{
		Email:    r.FormValue("email"),
		Password: string(hashedPassword),
		FullName: r.FormValue("full_name"),
		Role:     RoleUser, // กำหนดบทบาทเป็น User ทันที
	}

	if err := db.Create(&user).Error; err != nil {
		http.Error(w, "Email นี้ถูกใช้งานในระบบแล้ว", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func LoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html lang="th"><head><meta charset="UTF-8"><script src="https://cdn.tailwindcss.com"></script></head>
		<body class="bg-slate-950 text-slate-100 flex items-center justify-center min-h-screen">
			<div class="bg-slate-900 border border-slate-800 p-8 rounded-2xl shadow-2xl w-full max-w-md">
				<div class="flex justify-center mb-4"><div class="w-16 h-16 bg-cyan-500/10 rounded-full flex items-center justify-center border border-cyan-500/30 text-cyan-400 text-3xl">💻</div></div>
				<h2 class="text-3xl font-extrabold text-center text-cyan-400 mb-6">เข้าสู่ระบบ IT Helpdesk</h2>
				<form method="POST" class="space-y-4">
					<div><label class="block text-sm mb-1 text-slate-400">อีเมล</label><input type="email" name="email" required class="w-full bg-slate-800 border border-slate-700 rounded-lg px-4 py-2.5 focus:outline-none focus:border-cyan-500"></div>
					<div><label class="block text-sm mb-1 text-slate-400">รหัสผ่าน</label><input type="password" name="password" required class="w-full bg-slate-800 border border-slate-700 rounded-lg px-4 py-2.5 focus:outline-none focus:border-cyan-500"></div>
					<button type="submit" class="w-full bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-bold py-3 rounded-lg shadow-lg shadow-cyan-500/20 transition">เข้าสู่ระบบ</button>
				</form>
				<p class="text-center text-sm text-slate-400 mt-4">ยังไม่มีบัญชี? <a href="/register" class="text-cyan-400 hover:underline">สมัครสมาชิกที่นี่</a></p>
			</div>
		</body></html>`)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	var user User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		http.Error(w, "อีเมลหรือรหัสผ่านไม่ถูกต้อง", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		http.Error(w, "อีเมลหรือรหัสผ่านไม่ถูกต้อง", http.StatusUnauthorized)
		return
	}

	sessionToken := fmt.Sprintf("token_%d_%d", user.ID, time.Now().UnixNano())
	sessions[sessionToken] = user.ID
	http.SetCookie(w, &http.Cookie{Name: "session_token", Value: sessionToken, Path: "/"})
	http.Redirectif user.Role == RoleAdmin {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/tickets", http.StatusSeeOther)
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		delete(sessions, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "session_token", Value: "", MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// --- CRUD READ: แสดงเฉพาะตั๋วที่เป็นของตนเอง (Data Ownership) ---

func UserTicketsPage(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(User)

	var tickets []Ticket
	db.Where("user_id = ?", user.ID).Order("created_at desc").Find(&tickets)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// ส่วนที่เพิ่ม: แถบแจ้งเตือนสีทองสำหรับ Admin
	adminNotice := ""
	if user.Role == RoleAdmin {
		adminNotice = `
		<div class="bg-amber-500/10 border border-amber-500/30 p-4 rounded-2xl mb-6 flex flex-wrap justify-between items-center text-amber-400">
			<div class="flex items-center space-x-2">
				<span class="text-xl">👑</span>
				<span class="font-semibold text-sm">คุณกำลังเข้าใช้งานในสิทธิ์ Admin (เจ้าหน้าที่ IT)</span>
			</div>
			<a href="/admin" class="bg-amber-500 hover:bg-amber-400 text-slate-950 font-bold px-4 py-2 rounded-lg text-sm shadow-lg shadow-amber-500/20 transition">
				ไปที่หน้าจัดการ Admin Panel ➔
			</a>
		</div>`
	}

	html := fmt.Sprintf(`
	%s <!-- แสดงแถบ Admin Notice ถ้าเป็น Admin -->

	<div class="bg-gradient-to-r from-slate-900 to-slate-800 border border-slate-800 p-6 rounded-2xl mb-8 flex justify-between items-center shadow-xl">
		<div>
			<h1 class="text-2xl font-bold">สวัสดี, <span class="text-cyan-400">%s</span></h1>
			<p class="text-slate-400 text-sm mt-1">แผนก: <span class="text-slate-200 font-semibold">%s</span> | สิทธิ์: <span class="uppercase text-cyan-400 font-bold">%s</span></p>
		</div>
		<a href="/tickets/new" class="bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-bold px-4 py-2.5 rounded-lg shadow-lg shadow-cyan-500/20 transition">+ สร้างตั๋วแจ้งซ่อม</a>
	</div>

	<div class="bg-slate-900 border border-slate-800 p-6 rounded-2xl shadow-xl">
		<h2 class="text-xl font-bold text-slate-100 mb-4">📋 ตั๋วแจ้งซ่อมทั้งหมดของคุณ</h2>
		<div class="overflow-x-auto">
			<table class="w-full text-left border-collapse">
				<thead>
					<tr class="border-b border-slate-800 text-slate-400 text-sm">
						<th class="p-3">ID</th>
						<th class="p-3">หัวข้อแจ้งซ่อม</th>
						<th class="p-3">ประเภทอุปกรณ์</th>
						<th class="p-3">รายละเอียดอาการ</th>
						<th class="p-3">สถานะ</th>
						<th class="p-3">จัดการ</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-800/50 text-sm">`, adminNotice, user.FullName, user.Department, user.Role)

	for _, t := range tickets {
		statusBadge := fmt.Sprintf(`<span class="px-2.5 py-1 rounded-full text-xs font-bold bg-amber-500/10 text-amber-400 border border-amber-500/20">%s</span>`, t.Status)
		if t.Status == "In-Progress" {
			statusBadge = fmt.Sprintf(`<span class="px-2.5 py-1 rounded-full text-xs font-bold bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">%s</span>`, t.Status)
		} else if t.Status == "Resolved" {
			statusBadge = fmt.Sprintf(`<span class="px-2.5 py-1 rounded-full text-xs font-bold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">%s</span>`, t.Status)
		}

		html += fmt.Sprintf(`<tr>
			<td class="p-3">#%d</td>
			<td class="p-3 font-semibold text-cyan-400">%s</td>
			<td class="p-3">%s</td>
			<td class="p-3 text-slate-400">%s</td>
			<td class="p-3">%s</td>
			<td class="p-3"><a href="/tickets/delete?id=%d" class="text-rose-400 hover:underline">ยกเลิก/ลบ</a></td>
		</tr>`, t.ID, t.Title, t.DeviceType, t.Description, statusBadge, t.ID)
	}

	html += `</tbody></table></div></div>`
	tpl.Execute(w, template.HTML(html))
}
// --- Form 1: Create Ticket ---

func CreateTicketPage(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(User)

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `
		<div class="max-w-xl mx-auto bg-slate-900 border border-slate-800 p-6 rounded-2xl shadow-xl">
			<h2 class="text-2xl font-bold text-cyan-400 mb-6 flex items-center gap-2"><span>🛠️</span> ฟอร์มแจ้งซ่อมอุปกรณ์ IT (ฟอร์มที่ 1)</h2>
			<form method="POST" class="space-y-4">
				<div>
					<label class="block text-sm text-slate-400 mb-1">หัวข้อปัญหา</label>
					<input type="text" name="title" required placeholder="เช่น จอภาพไม่ติด, ออกอินเทอร์เน็ตไม่ได้" class="w-full bg-slate-800 border border-slate-700 rounded-lg p-3 focus:border-cyan-500 focus:outline-none">
				</div>
				<div>
					<label class="block text-sm text-slate-400 mb-1">ประเภทหมวดหมู่อุปกรณ์</label>
					<select name="device_type" class="w-full bg-slate-800 border border-slate-700 rounded-lg p-3 focus:border-cyan-500 focus:outline-none">
						<option value="Computer/Laptop">Computer / Laptop</option>
						<option value="Printer/Scanner">Printer / Scanner</option>
						<option value="Network/Wi-Fi">Network / Wi-Fi</option>
						<option value="Software/OS">Software / Operating System</option>
					</select>
				</div>
				<div>
					<label class="block text-sm text-slate-400 mb-1">รายละเอียดอาการชำรุดเพิ่มเติม</label>
					<textarea name="description" rows="4" required placeholder="อธิบายอาการเบื้องต้น หรือรหัสข้อผิดพลาดที่พบ..." class="w-full bg-slate-800 border border-slate-700 rounded-lg p-3 focus:border-cyan-500 focus:outline-none"></textarea>
				</div>
				<button type="submit" class="w-full bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-bold py-3 rounded-lg shadow-lg shadow-cyan-500/20 transition">ส่งเรื่องแจ้งซ่อม</button>
			</form>
		</div>`
		tpl.Execute(w, template.HTML(html))
		return
	}

	ticket := Ticket{
		Title:       r.FormValue("title"),
		DeviceType:  r.FormValue("device_type"),
		Description: r.FormValue("description"),
		UserID:      user.ID, // ผูก ID เจ้าของตั๋ว
	}
	db.Create(&ticket)
	http.Redirect(w, r, "/tickets", http.StatusSeeOther)
}

func DeleteTicketHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(User)
	ticketID := r.URL.Query().Get("id")

	var ticket Ticket
	if err := db.First(&ticket, ticketID).Error; err != nil {
		http.Error(w, "ไม่พบรายการ", http.StatusNotFound)
		return
	}

	// DELETE DATA OWNERSHIP CHECK: ผู้ใช้ A ต้องลบตั๋วของผู้ใช้ B ไม่ได้
	if ticket.UserID != user.ID && user.Role != RoleAdmin {
		http.Error(w, "Forbidden: คุณไม่มีสิทธิ์ลบรายการของผู้อื่น", http.StatusForbidden)
		return
	}

	db.Delete(&ticket)
	http.Redirect(w, r, "/tickets", http.StatusSeeOther)
}

// --- Form 2: Profile Update ---

func ProfilePage(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(User)

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := fmt.Sprintf(`
		<div class="max-w-md mx-auto bg-slate-900 border border-slate-800 p-6 rounded-2xl shadow-xl">
			<h2 class="text-2xl font-bold text-cyan-400 mb-6">👤 แก้ไขข้อมูลส่วนตัว (ฟอร์มที่ 2)</h2>
			<form method="POST" class="space-y-4">
				<div>
					<label class="block text-sm text-slate-400 mb-1">อีเมล (อ่านอย่างเดียว)</label>
					<input type="text" value="%s" disabled class="w-full bg-slate-800/50 border border-slate-700 text-slate-500 rounded-lg p-3 cursor-not-allowed">
				</div>
				<div>
					<label class="block text-sm text-slate-400 mb-1">ชื่อ-นามสกุล</label>
					<input type="text" name="full_name" value="%s" required class="w-full bg-slate-800 border border-slate-700 rounded-lg p-3 focus:border-cyan-500 focus:outline-none">
				</div>
				<div>
					<label class="block text-sm text-slate-400 mb-1">แผนกงาน / ฝ่ายงาน</label>
					<input type="text" name="department" value="%s" placeholder="เช่น ฝ่ายบัญชี, การตลาด" class="w-full bg-slate-800 border border-slate-700 rounded-lg p-3 focus:border-cyan-500 focus:outline-none">
				</div>
				<button type="submit" class="w-full bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-bold py-3 rounded-lg shadow-lg shadow-cyan-500/20 transition">อัปเดตข้อมูลส่วนตัว</button>
			</form>
		</div>`, user.Email, user.FullName, user.Department)
		tpl.Execute(w, template.HTML(html))
		return
	}

	user.FullName = r.FormValue("full_name")
	user.Department = r.FormValue("department")
	db.Save(&user)

	http.Redirect(w, r, "/tickets", http.StatusSeeOther)
}

// --- Feature 5: Admin Dashboard ---

func AdminDashboardPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		tID, _ := strconv.Atoi(r.FormValue("ticket_id"))
		status := r.FormValue("status")
		db.Model(&Ticket{}).Where("id = ?", tID).Update("status", status)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	var allTickets []Ticket
	db.Preload("User").Order("created_at desc").Find(&allTickets)

	var totalUsers int64
	db.Model(&User{}).Count(&totalUsers)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := fmt.Sprintf(`
	<div class="bg-slate-900 border border-slate-800 p-6 rounded-2xl shadow-xl mb-8">
		<h1 class="text-2xl font-bold text-amber-400 mb-2">👑 IT Support Admin Dashboard</h1>
		<p class="text-slate-400 text-sm">ผู้ใช้งานในระบบ: <span class="text-cyan-400 font-bold">%d คน</span> | คำขอแจ้งซ่อมทั้งหมด: <span class="text-cyan-400 font-bold">%d รายการ</span></p>
	</div>

	<div class="bg-slate-900 border border-slate-800 p-6 rounded-2xl shadow-xl">
		<h2 class="text-xl font-bold mb-4">รายการแจ้งซ่อมของพนักงานทั้งหมด</h2>
		<div class="overflow-x-auto">
			<table class="w-full text-left border-collapse">
				<thead>
					<tr class="border-b border-slate-800 text-slate-400 text-sm">
						<th class="p-3">ID</th>
						<th class="p-3">ผู้แจ้งเรื่อง</th>
						<th class="p-3">หัวข้อปัญหา</th>
						<th class="p-3">ประเภทอุปกรณ์</th>
						<th class="p-3">สถานะงาน</th>
						<th class="p-3">อัปเดตสถานะงาน</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-800/50 text-sm">`, totalUsers, len(allTickets))

	for _, t := range allTickets {
		html += fmt.Sprintf(`<tr>
			<td class="p-3">#%d</td>
			<td class="p-3 font-semibold">%s <span class="text-xs font-normal text-slate-500">(%s)</span></td>
			<td class="p-3 text-cyan-400">%s</td>
			<td class="p-3">%s</td>
			<td class="p-3"><span class="px-2.5 py-1 rounded-full text-xs font-bold bg-slate-800 text-slate-300 border border-slate-700">%s</span></td>
			<td class="p-3">
				<form method="POST" class="flex gap-2">
					<input type="hidden" name="ticket_id" value="%d">
					<select name="status" class="bg-slate-800 border border-slate-700 rounded px-2 py-1 text-xs">
						<option value="Pending" %s>Pending</option>
						<option value="In-Progress" %s>In-Progress</option>
						<option value="Resolved" %s>Resolved</option>
					</select>
					<button type="submit" class="bg-amber-500/20 text-amber-400 border border-amber-500/30 px-2 py-1 rounded text-xs hover:bg-amber-500/40">บันทึก</button>
				</form>
			</td>
		</tr>`, t.ID, t.User.FullName, t.User.Email, t.Title, t.DeviceType, t.Status, t.ID,
			selectAttr(t.Status, "Pending"), selectAttr(t.Status, "In-Progress"), selectAttr(t.Status, "Resolved"))
	}

	html += `</tbody></table></div></div>`
	tpl.Execute(w, template.HTML(html))
}

func selectAttr(currentStatus, target string) string {
	if currentStatus == target {
		return "selected"
	}
	return ""
}

// ==========================================
// 5. MAIN EXECUTION
// ==========================================

func main() {
	var err error
	db, err = gorm.Open(sqlite.Open("helpdesk.db"), &gorm.Config{})
	if err != nil {
		panic("เชื่อมต่อฐานข้อมูลล้มเหลว: " + err.Error())
	}

	// Auto Migration
	db.AutoMigrate(&User{}, &Ticket{})

	// Auto-Create Default Admin Account
	var adminCount int64
	db.Model(&User{}).Where("role = ?", RoleAdmin).Count(&adminCount)
	if adminCount == 0 {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin1234"), 10)
		defaultAdmin := User{
			Email:      "admin@helpdesk.com",
			Password:   string(hashedPassword),
			Role:       RoleAdmin,
			FullName:   "System Administrator",
			Department: "IT Support",
		}
		db.Create(&defaultAdmin)
		fmt.Println("🔑 สร้างบัญชี Admin เรียบร้อย: admin@helpdesk.com / admin1234")
	}

	http.HandleFunc("/register", RegisterPage)
	http.HandleFunc("/login", LoginPage)
	http.HandleFunc("/logout", LogoutHandler)

	http.HandleFunc("/tickets", AuthMiddleware(UserTicketsPage))
	http.HandleFunc("/tickets/new", AuthMiddleware(CreateTicketPage))
	http.HandleFunc("/tickets/delete", AuthMiddleware(DeleteTicketHandler))
	http.HandleFunc("/profile", AuthMiddleware(ProfilePage))

	http.HandleFunc("/admin", AdminMiddleware(AdminDashboardPage))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	// อ่าน Port จาก Environment Variable ที่ Render กำหนด
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 IT Helpdesk System รันเรียบร้อยที่พอร์ต: %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
