package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- YAPILAR (STRUCTS) ---

// GORM modeline uygun User yapısı
type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Email    string `gorm:"uniqueIndex;not null" json:"email"`
	Password string `gorm:"not null" json:"password"`
	Role     string `gorm:"default:'user'" json:"role"`
}

type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// --- GLOBAL DEĞİŞKENLER ---
var DB *gorm.DB
var jwtSecret []byte

// Veritabanı Başlatma Fonksiyonu
func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Docker dışında lokalde çalıştırırsan diye varsayılan değer
		dsn = "host=localhost user=admin password=secretpassword dbname=go_app_db port=5432 sslmode=disable"
	}

	var err error
	// Veritabanına bağlanmayı dener (Docker ayağa kalkarken biraz bekleyebilir, o yüzden retry eklenebilir ama basit tuttuk)
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("❌ Veritabanına bağlanılamadı: " + err.Error())
	}

	fmt.Println("✅ PostgreSQL veritabanına başarıyla bağlanıldı!")

	// Tabloları otomatik oluşturur/günceller
	DB.AutoMigrate(&User{})
}

func main() {
	// Veritabanını Başlat
	initDB()

	// JWT Secret
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "cok-gizli-anahtar-123"
	}
	jwtSecret = []byte(secret)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// 1️⃣ REGISTER (Veritabanına Kayıt)
	r.POST("/register", func(c *gin.Context) {
		var newUser User
		if err := c.BindJSON(&newUser); err != nil {
			c.JSON(400, gin.H{"error": "Veri bozuk"})
			return
		}

		// Şifreyi Bcrypt ile hashle (Güvenlik!)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(500, gin.H{"error": "Şifre şifrelenemedi"})
			return
		}
		newUser.Password = string(hashedPassword)

		// Veritabanına kaydet
		if err := DB.Create(&newUser).Error; err != nil {
			c.JSON(400, gin.H{"error": "Bu e-posta zaten kayıtlı olabilir"})
			return
		}

		c.JSON(200, gin.H{"message": "Kayıt başarılı", "user": newUser.Email, "id": newUser.ID})
	})

	// 2️⃣ LOGIN (Veritabanından Kontrol)
	r.POST("/login", func(c *gin.Context) {
		var loginReq User
		if err := c.BindJSON(&loginReq); err != nil {
			c.JSON(400, gin.H{"error": "Veri bozuk"})
			return
		}

		var user User
		// Veritabanından email'e göre kullanıcıyı bul
		if err := DB.Where("email = ?", loginReq.Email).First(&user).Error; err != nil {
			c.JSON(401, gin.H{"error": "Hatalı e-posta veya şifre"})
			return
		}

		// Hashlenmiş şifre ile girilen şifreyi karşılaştır
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password)); err != nil {
			c.JSON(401, gin.H{"error": "Hatalı e-posta veya şifre"})
			return
		}

		// Token Üretimi
		expirationTime := time.Now().Add(1 * time.Hour)
		claims := &Claims{
			Email: user.Email,
			Role:  user.Role,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationTime),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(jwtSecret)
		if err != nil {
			c.JSON(500, gin.H{"error": "Token üretilemedi"})
			return
		}

		c.SetCookie("jwt_token", tokenString, 3600, "/", "", false, true)
		c.JSON(200, gin.H{"message": "Giriş başarılı, cookie set edildi!"})
	})

	// 3️⃣ KORUMALI ALAN (n8n'i Tetikleyen Endpoint)
	protected := r.Group("/")
	protected.Use(AuthMiddleware("admin"))
	{
		protected.POST("/send-to-n8n", func(c *gin.Context) {
			callbackURL := "http://go-app:8080/callback"

			message := map[string]string{
				"message":      "hello from go securely",
				"author":       "admin",
				"timestamp":    time.Now().String(),
				"callback_url": callbackURL,
			}
			jsonData, _ := json.Marshal(message)

			n8nHMACSecret := []byte("cok-gizli-n8n-sifresi")
			h := hmac.New(sha256.New, n8nHMACSecret)
			h.Write(jsonData)
			signature := hex.EncodeToString(h.Sum(nil))

			n8nURL := "http://n8n:5678/webhook/test-webhook"
			req, _ := http.NewRequest("POST", n8nURL, bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			req.SetBasicAuth("admin", "admin123")
			req.Header.Set("X-Hmac-Signature", signature)

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				c.JSON(500, gin.H{"error": "n8n'e ulaşılamadı: " + err.Error()})
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				c.JSON(resp.StatusCode, gin.H{"error": "n8n hata döndü (Auth veya Signature hatası olabilir)"})
				return
			}

			c.JSON(200, gin.H{"status": "sent securely with HMAC 🚀"})
		})
	}

	// 4️⃣ PUBLIC CALLBACK (n8n'den gelen cevap)
	r.POST("/callback", func(c *gin.Context) {
		secretToken := c.GetHeader("X-CALLBACK-TOKEN")
		if secretToken != "internal-secret" && secretToken != os.Getenv("N8N_CALLBACK_TOKEN") {
			c.JSON(403, gin.H{"error": "Yetkisiz erişim! Yanlış Secret."})
			return
		}

		var body map[string]interface{}
		c.BindJSON(&body)

		fmt.Println("🚀 CALLBACK ÇALIŞIYOR 🚀")
		fmt.Println("Data:", body)

		c.JSON(200, gin.H{"status": "securely processed"})
	})

	r.Run(":8080")
}

// --- MIDDLEWARE ---
func AuthMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("jwt_token")

		if err != nil {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(401, gin.H{"error": "Token bulunamadı (Lütfen giriş yapın)"})
				c.Abort()
				return
			}
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "Geçersiz Token"})
			c.Abort()
			return
		}

		if claims.Role != requiredRole {
			c.JSON(403, gin.H{"error": "Yetkiniz yok (Admin only)"})
			c.Abort()
			return
		}

		c.Next()
	}
}
