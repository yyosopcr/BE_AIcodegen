# LBK Points Transfer API

Backend API สำหรับระบบโอนแต้ม LBK Membership ที่ให้สมาชิกสามารถโอนแต้มระหว่างกันได้ผ่าน Member ID

## Features

- 🔐 User Authentication (JWT)
- 👤 User Profile Management
- 💰 Points Balance System
- 🔄 Points Transfer between Members
- 📊 Transaction History
- 🔍 User Search by Member ID

## Tech Stack

- **Backend**: Go (Golang) with Fiber framework
- **Database**: SQLite with GORM
- **Authentication**: JWT (JSON Web Tokens)
- **API Documentation**: Swagger UI

## Database Schema

### User Model
```go
type User struct {
    ID          uint      `json:"id"`
    Email       string    `json:"email"`
    FirstName   string    `json:"first_name"`
    LastName    string    `json:"last_name"`
    Phone       string    `json:"phone"`
    Birthday    string    `json:"birthday"`
    MemberID    string    `json:"member_id"`    // LBK Member ID (e.g., LBK001234)
    MemberTier  string    `json:"member_tier"`  // Gold, Silver, etc.
    Points      int64     `json:"points"`       // Available points balance
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### Transaction Model
```go
type Transaction struct {
    ID          uint      `json:"id"`
    FromUserID  uint      `json:"from_user_id"`
    ToUserID    uint      `json:"to_user_id"`
    Amount      int64     `json:"amount"`
    Type        string    `json:"type"`         // "transfer", "receive"
    Status      string    `json:"status"`       // "completed", "pending", "failed"
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
}
```

## API Endpoints

### Authentication Endpoints

#### POST `/register`
สมัครสมาชิกใหม่
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "first_name": "สมชาย",
    "last_name": "ใจดี",
    "phone": "081-234-5678",
    "birthday": "1990-01-01",
    "member_id": "LBK001234"
  }' \
  http://localhost:3000/register
```

**Response:**
```json
{
  "id": 1,
  "email": "user@example.com",
  "member_id": "LBK001234"
}
```

#### POST `/login`
เข้าสู่ระบบ
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }' \
  http://localhost:3000/login
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### User Profile Endpoints

#### GET `/me`
ดูข้อมูลโปรไฟล์และยอดแต้มปัจจุบัน
```bash
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  http://localhost:3000/me
```

**Response:**
```json
{
  "id": 1,
  "email": "user@example.com",
  "first_name": "สมชาย",
  "last_name": "ใจดี",
  "phone": "081-234-5678",
  "birthday": "1990-01-01",
  "member_id": "LBK001234",
  "member_tier": "Gold",
  "points": 15420
}
```

### Points Transfer Endpoints

#### GET `/search/user`
ค้นหาสมาชิกด้วย Member ID (สำหรับโอนแต้ม)
```bash
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  "http://localhost:3000/search/user?member_id=LBK002345"
```

**Response:**
```json
{
  "member_id": "LBK002345",
  "first_name": "นาง",
  "last_name": "สวยงาม",
  "member_tier": "Gold"
}
```

#### POST `/transfer`
โอนแต้มให้สมาชิกคนอื่น
```bash
curl -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "to_member_id": "LBK002345",
    "amount": 1000
  }' \
  http://localhost:3000/transfer
```

**Response:**
```json
{
  "message": "Transfer successful",
  "transaction_id": 1,
  "remaining_points": 14420,
  "transferred_amount": 1000,
  "recipient": {
    "member_id": "LBK002345",
    "first_name": "นาง",
    "last_name": "สวยงาม"
  }
}
```

#### GET `/transactions/recent`
ดูประวัติการทำธุรกรรมล่าสุด
```bash
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  http://localhost:3000/transactions/recent
```

**Response:**
```json
{
  "transactions": [
    {
      "id": 1,
      "contact_name": "นาง สวยงาม",
      "contact_member_id": "LBK002345",
      "amount": -1000,
      "type": "sent",
      "status": "completed",
      "date": "2025-08-27",
      "time": "15:40"
    }
  ]
}
```

### System Endpoints

#### GET `/`
Health check
```bash
curl http://localhost:3000/
```

#### GET `/swagger`
API Documentation (Swagger UI)
```
http://localhost:3000/swagger
```

## Installation & Running

### Prerequisites
- Go 1.19+ with CGO enabled
- SQLite support

### Setup
```bash
# Clone repository
git clone <repository-url>
cd tmp-kbtg-be

# Install dependencies
go mod tidy

# Run server
CGO_ENABLED=1 go run main.go
```

Server จะรันที่ `http://localhost:3000`

## Error Handling

API จะส่งกลับ error ในรูปแบบ:
```json
{
  "error": "error message description"
}
```

### Common Error Codes
- `400` - Bad Request (ข้อมูลไม่ถูกต้อง)
- `401` - Unauthorized (ไม่มีสิทธิ์เข้าถึง)
- `404` - Not Found (ไม่พบข้อมูล)
- `500` - Internal Server Error (ข้อผิดพลาดระบบ)

## Security Features

- 🔒 Password hashing with bcrypt
- 🎫 JWT token authentication
- 🛡️ Protected routes with middleware
- 💸 Balance validation for transfers
- 🔄 Database transactions for consistency

## Points System

- สมาชิกใหม่เริ่มต้นด้วย **15,420 แต้ม**
- แต้มสามารถโอนระหว่างสมาชิกได้
- ระบบตรวจสอบยอดคงเหลือก่อนการโอน
- บันทึกประวัติการทำธุรกรรมทั้งหมด
