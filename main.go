package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User model
type User struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Email       string `json:"email" gorm:"uniqueIndex;not null"`
	Password    string `json:"-"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Phone       string `json:"phone"`
	Birthday    string `json:"birthday"` // keep simple as YYYY-MM-DD
	MemberID    string `json:"member_id" gorm:"uniqueIndex;not null"` // LBK member ID
	MemberTier  string `json:"member_tier" gorm:"default:'Gold'"`     // Gold, Silver, etc.
	Points      int64  `json:"points" gorm:"default:0"`               // Available points
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Transaction model for transfer history
type Transaction struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	FromUserID  uint      `json:"from_user_id"`
	ToUserID    uint      `json:"to_user_id"`
	FromUser    User      `json:"from_user" gorm:"foreignKey:FromUserID"`
	ToUser      User      `json:"to_user" gorm:"foreignKey:ToUserID"`
	Amount      int64     `json:"amount"`
	Type        string    `json:"type"` // "transfer", "receive"
	Status      string    `json:"status" gorm:"default:'completed'"` // completed, pending, failed
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time
}

var db *gorm.DB

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&User{}, &Transaction{}); err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func checkPasswordHash(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func generateJWT(userID uint) (string, error) {
	secret := jwtSecret()
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprint(userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func jwtSecret() string {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s
	}
	return "secret" // default (override in production)
}

// Middleware to protect routes
func jwtMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization header"})
		}
		tokStr := parts[1]
		var claims jwt.RegisteredClaims
		tok, err := jwt.ParseWithClaims(tokStr, &claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(jwtSecret()), nil
		})
		if err != nil || !tok.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		// load user
		userID := claims.Subject
		var user User
		if err := db.First(&user, userID).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
		}
		c.Locals("user", user)
		return c.Next()
	}
}

func registerHandler(c *fiber.Ctx) error {
	var payload struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Phone     string `json:"phone"`
		Birthday  string `json:"birthday"`
		MemberID  string `json:"member_id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	if payload.Email == "" || payload.Password == "" || payload.MemberID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email, password and member_id required"})
	}
	// check existing email
	var existing User
	if err := db.Where("email = ?", payload.Email).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email already registered"})
	}
	// check existing member_id
	if err := db.Where("member_id = ?", payload.MemberID).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "member_id already registered"})
	}
	hash, err := hashPassword(payload.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
	}
	user := User{
		Email:      payload.Email,
		Password:   hash,
		FirstName:  payload.FirstName,
		LastName:   payload.LastName,
		Phone:      payload.Phone,
		Birthday:   payload.Birthday,
		MemberID:   payload.MemberID,
		MemberTier: "Gold", // default tier
		Points:     15420,  // default points like in screenshot
	}
	if err := db.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create user"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": user.ID, "email": user.Email, "member_id": user.MemberID})
}

func loginHandler(c *fiber.Ctx) error {
	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	var user User
	if err := db.Where("email = ?", payload.Email).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}
	if err := checkPasswordHash(payload.Password, user.Password); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}
	token, err := generateJWT(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
	}
	return c.JSON(fiber.Map{"token": token})
}

func meHandler(c *fiber.Ctx) error {
	u := c.Locals("user")
	if u == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	user := u.(User)
	// don't return password
	user.Password = ""
	return c.JSON(user)
}

// Transfer points handler
func transferHandler(c *fiber.Ctx) error {
	u := c.Locals("user")
	if u == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	fromUser := u.(User)

	var payload struct {
		ToMemberID string `json:"to_member_id"`
		Amount     int64  `json:"amount"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}

	if payload.ToMemberID == "" || payload.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "to_member_id and positive amount required"})
	}

	if payload.ToMemberID == fromUser.MemberID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot transfer to yourself"})
	}

	// Check if sender has enough points
	if fromUser.Points < payload.Amount {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "insufficient points"})
	}

	// Find recipient by member ID
	var toUser User
	if err := db.Where("member_id = ?", payload.ToMemberID).First(&toUser).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "recipient not found"})
	}

	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Deduct points from sender
	if err := tx.Model(&fromUser).Update("points", fromUser.Points-payload.Amount).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to deduct points"})
	}

	// Add points to recipient
	if err := tx.Model(&toUser).Update("points", toUser.Points+payload.Amount).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add points"})
	}

	// Create transaction record
	transaction := Transaction{
		FromUserID:  fromUser.ID,
		ToUserID:    toUser.ID,
		Amount:      payload.Amount,
		Type:        "transfer",
		Status:      "completed",
		Description: fmt.Sprintf("Transfer to %s %s", toUser.FirstName, toUser.LastName),
	}
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create transaction record"})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to complete transfer"})
	}

	// Return success response with updated balance
	updatedFromUser := fromUser
	updatedFromUser.Points -= payload.Amount
	
	return c.JSON(fiber.Map{
		"message":           "Transfer successful",
		"transaction_id":    transaction.ID,
		"remaining_points":  updatedFromUser.Points,
		"transferred_amount": payload.Amount,
		"recipient":         fiber.Map{
			"member_id":  toUser.MemberID,
			"first_name": toUser.FirstName,
			"last_name":  toUser.LastName,
		},
	})
}

// Get recent transactions for current user
func recentTransactionsHandler(c *fiber.Ctx) error {
	u := c.Locals("user")
	if u == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	user := u.(User)

	var transactions []Transaction
	if err := db.Where("from_user_id = ? OR to_user_id = ?", user.ID, user.ID).
		Preload("FromUser").
		Preload("ToUser").
		Order("created_at DESC").
		Limit(10).
		Find(&transactions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch transactions"})
	}

	// Format transactions for response
	var formattedTx []fiber.Map
	for _, tx := range transactions {
		var contactName, contactMemberID, txType string
		var amount int64

		if tx.FromUserID == user.ID {
			// User sent money
			contactName = fmt.Sprintf("%s %s", tx.ToUser.FirstName, tx.ToUser.LastName)
			contactMemberID = tx.ToUser.MemberID
			txType = "sent"
			amount = -tx.Amount // negative for sent
		} else {
			// User received money
			contactName = fmt.Sprintf("%s %s", tx.FromUser.FirstName, tx.FromUser.LastName)
			contactMemberID = tx.FromUser.MemberID
			txType = "received"
			amount = tx.Amount // positive for received
		}

		formattedTx = append(formattedTx, fiber.Map{
			"id":               tx.ID,
			"contact_name":     contactName,
			"contact_member_id": contactMemberID,
			"amount":           amount,
			"type":             txType,
			"status":           tx.Status,
			"date":             tx.CreatedAt.Format("2006-01-02"),
			"time":             tx.CreatedAt.Format("15:04"),
		})
	}

	return c.JSON(fiber.Map{
		"transactions": formattedTx,
	})
}

// Search user by member ID for transfer
func searchUserHandler(c *fiber.Ctx) error {
	memberID := c.Query("member_id")
	if memberID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "member_id query parameter required"})
	}

	u := c.Locals("user")
	if u == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	currentUser := u.(User)

	if memberID == currentUser.MemberID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot search for yourself"})
	}

	var user User
	if err := db.Where("member_id = ?", memberID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	return c.JSON(fiber.Map{
		"member_id":   user.MemberID,
		"first_name":  user.FirstName,
		"last_name":   user.LastName,
		"member_tier": user.MemberTier,
	})
}

// Serve minimal OpenAPI JSON and Swagger UI
func swaggerJSON(c *fiber.Ctx) error {
	op := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   "LBK Points Transfer API",
			"version": "1.0.0",
			"description": "API for LBK member points transfer system",
		},
		"paths": map[string]interface{}{
			"/register": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Register user",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"required": []string{"email", "password", "member_id"},
									"properties": map[string]interface{}{
										"email":      map[string]interface{}{"type": "string"},
										"password":   map[string]interface{}{"type": "string"},
										"first_name": map[string]interface{}{"type": "string"},
										"last_name":  map[string]interface{}{"type": "string"},
										"phone":      map[string]interface{}{"type": "string"},
										"birthday":   map[string]interface{}{"type": "string"},
										"member_id":  map[string]interface{}{"type": "string"},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{"description": "User created successfully"},
						"400": map[string]interface{}{"description": "Bad request"},
					},
				},
			},
			"/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Login user",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"required": []string{"email", "password"},
									"properties": map[string]interface{}{
										"email":    map[string]interface{}{"type": "string"},
										"password": map[string]interface{}{"type": "string"},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Login successful"},
						"401": map[string]interface{}{"description": "Invalid credentials"},
					},
				},
			},
			"/me": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":   "Get current user profile",
					"security":  []map[string][]string{{"bearerAuth": {}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "User profile"},
						"401": map[string]interface{}{"description": "Unauthorized"},
					},
				},
			},
			"/transfer": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Transfer points to another user",
					"security": []map[string][]string{{"bearerAuth": {}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"required": []string{"to_member_id", "amount"},
									"properties": map[string]interface{}{
										"to_member_id": map[string]interface{}{"type": "string"},
										"amount":       map[string]interface{}{"type": "integer"},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Transfer successful"},
						"400": map[string]interface{}{"description": "Bad request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
					},
				},
			},
			"/transactions/recent": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":   "Get recent transactions",
					"security":  []map[string][]string{{"bearerAuth": {}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Recent transactions"},
						"401": map[string]interface{}{"description": "Unauthorized"},
					},
				},
			},
			"/search/user": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":   "Search user by member ID",
					"security":  []map[string][]string{{"bearerAuth": {}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "member_id",
							"in":          "query",
							"required":    true,
							"description": "LBK Member ID to search for",
							"schema":      map[string]interface{}{"type": "string"},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "User found"},
						"404": map[string]interface{}{"description": "User not found"},
						"401": map[string]interface{}{"description": "Unauthorized"},
					},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
	}
	return c.JSON(op)
}

func swaggerUI(c *fiber.Ctx) error {
	html := `<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>Swagger UI</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/4.18.3/swagger-ui.min.css" />
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/4.18.3/swagger-ui-bundle.min.js"></script>
    <script>
      window.ui = SwaggerUIBundle({
        url: '/swagger/doc.json',
        dom_id: '#swagger-ui'
      })
    </script>
  </body>
</html>`
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

func main() {
	initDB()
	app := fiber.New()

	// basic endpoints
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Hello World"})
	})

	api := app.Group("/")
	api.Post("/register", registerHandler)
	api.Post("/login", loginHandler)
	api.Get("/me", jwtMiddleware(), meHandler)

	// Transfer and transaction endpoints
	api.Post("/transfer", jwtMiddleware(), transferHandler)
	api.Get("/transactions/recent", jwtMiddleware(), recentTransactionsHandler)
	api.Get("/search/user", jwtMiddleware(), searchUserHandler)

	// swagger
	app.Get("/swagger/doc.json", swaggerJSON)
	app.Get("/swagger", swaggerUI)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Fatal(app.Listen(":" + port))
}

/*
Example API calls:

# Health check
curl http://localhost:3000/

# Register a new user
curl -X POST -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","first_name":"สมชาย","last_name":"ใจดี","phone":"081-234-5678","birthday":"1990-01-01","member_id":"LBK001234"}' \
  http://localhost:3000/register

# Register another user for testing transfers
curl -X POST -H "Content-Type: application/json" \
  -d '{"email":"test2@example.com","password":"password123","first_name":"นางสาว","last_name":"สวยงาม","phone":"081-234-5679","birthday":"1992-05-15","member_id":"LBK002345"}' \
  http://localhost:3000/register

# Login
curl -X POST -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' \
  http://localhost:3000/login

# Get current user profile
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  http://localhost:3000/me

# Search for user by member ID (for transfer)
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  "http://localhost:3000/search/user?member_id=LBK002345"

# Transfer points
curl -X POST -H "Content-Type: application/json" -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{"to_member_id":"LBK002345","amount":1000}' \
  http://localhost:3000/transfer

# Get recent transactions
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  http://localhost:3000/transactions/recent
*/
