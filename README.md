# Donation Bars

AI destekli OBS donation bar tasarımcısı. Kullanıcılar doğal dilde yazdıkları promptlar ile OpenAI GPT-4o-mini kullanarak otomatik donation bar tasarımları oluşturabilir.

## Proje Hakkında

Bu uygulama, streamerlerin OBS için donation bar'ları kolayca oluşturmasını sağlar. Kullanıcı basitçe "Cyberpunk temalı, neon mavi renkli donation bar istiyorum" gibi doğal bir cümle yazarak AI'dan istediği tasarımı üretebilir.

### Temel Özellikler

- **AI Entegrasyonu**: OpenAI GPT-4o-mini ile doğal dil işleme
- **Veritabanı**: MongoDB ile donation bar verileri saklama
- **Cache**: Redis ile rate limiting (opsiyonel)
- **Web Interface**: HTML template'leri ile kullanıcı arayüzü
- **REST API**: Programatik erişim için API endpoints
- **Güvenlik**: JavaScript engelleme, rate limiting, injection koruması

### Teknik Stack

- **Backend**: Go 1.21+ (Gin Framework)
- **Database**: MongoDB 6.0+
- **Cache**: Redis 7.0+ (opsiyonel)
- **AI**: OpenAI GPT-4o-mini API
- **Frontend**: HTML templates + CSS (Server-side rendering)

## Kurulum ve Çalıştırma

### Gereksinimler

`
Go 1.21+
MongoDB 6.0+
OpenAI API Key
`

### Kurulum

`ash
# Repository klonlama
git clone https://github.com/alperen20t/donationbars.git
cd donationbars

# Bağımlılık yükleme
go mod download

# Environment dosyası oluşturma
cp env.example .env
`

### Konfigürasyon

.env dosyasında gerekli ayarları yapın:

`env
# Database
MONGO_URI=mongodb://localhost:27017
DB_NAME=donationbars

# OpenAI API
OPENAI_API_KEY=your-openai-api-key

# Server
PORT=8080

# Business Rules
MAX_BARS_PER_USER=5
RATE_LIMIT_PER_DAY=5

# Redis (opsiyonel)
REDIS_ENABLED=false
REDIS_ADDR=localhost:6379
`

### Çalıştırma

`ash
# MongoDB başlatma (Docker)
docker run -d -p 27017:27017 --name mongodb mongo:latest

# Uygulama çalıştırma
go run cmd/main.go

# Alternatif: Build edip çalıştırma
go build -o donationbars cmd/main.go
./donationbars
`

Uygulama http://localhost:8080 adresinde çalışacaktır.

## Proje Yapısı

`
donationbars/
 cmd/main.go                    # Uygulama giriş noktası
 internal/
    config/                    # Konfigürasyon yönetimi
       config.go
       redis.go
    handlers/handlers.go       # HTTP handlers (Web + API)
    services/                  # Business logic
       bar_service.go
       ai_service.go
    repository/                # Database operations
       bar_repository.go
    models/bar.go              # Data models
    interfaces/services.go     # Service interfaces
    errors/errors.go           # Custom error types
 templates/                     # HTML templates
 static/                        # CSS files
 env.example                    # Environment variables örneği
`

## API Endpoints

### Health Check
`
GET /health
`

### Donation Bar İşlemleri
`
GET    /api/v1/bars          # Kullanıcının barlarını listele
GET    /api/v1/bars/:id      # Belirli bar detayı
POST   /api/v1/bars          # Manuel bar oluştur
POST   /api/v1/bars/generate # AI ile bar oluştur
PUT    /api/v1/bars/:id      # Bar güncelle
DELETE /api/v1/bars/:id      # Bar sil
`

### Örnek AI Bar Oluşturma

`ash
curl -X POST http://localhost:8080/api/v1/bars/generate \
  -H "Content-Type: application/json" \
  -H "X-User-ID: test-user" \
  -d '{
    "prompt": "Cyberpunk temalı neon mavi donation bar",
    "language": "tr",
    "theme": "cyberpunk",
    "initial_amount": 100,
    "goal_amount": 1000
  }'
`

## Teknik Detaylar

### Clean Architecture

Proje Clean Architecture prensiplerine uygun yapılandırılmıştır:
- **Handlers**: HTTP isteklerini karşılar
- **Services**: İş mantığını yönetir
- **Repository**: Veritabanı işlemlerini yürütür
- **Interfaces**: Loose coupling sağlar

### Database Schema

MongoDB'de donation_bars collection'ında her bar şu alanları içerir:

`json
{
  "_id": "ObjectId",
  "user_id": "string",
  "name": "string",
  "description": "string",
  "html": "string",
  "css": "string",
  "language": "string",
  "theme": "string",
  "is_active": "boolean",
  "created_at": "datetime",
  "updated_at": "datetime",
  "initial_amount": "float64",
  "goal_amount": "float64",
  "ai_generated": "boolean",
  "prompt": "string",
  "has_valid_injections": "boolean"
}
`

### Rate Limiting

İki seviyeli rate limiting uygulanır:
1. **Redis ile (hızlı)**: Rate limit sayacı Redis'de tutulur
2. **MongoDB ile (fallback)**: Redis yoksa veritabanından kontrol edilir

### Injection Fields

Donation bar'larda kullanılabilecek dinamik alanlar:
- {goal}: Hedef tutar
- {total}: Toplanan tutar
- {percentage}: Tamamlanma yüzdesi
- {remaining}: Kalan tutar
- {description}: Bar açıklaması

### Güvenlik Önlemleri

- JavaScript kodları tamamen engellenir
- Harici CDN/font yüklemeleri yasaklanır
- Rate limiting (günlük 5 bar/kullanıcı)
- User ID bazlı yetkilendirme
- HTML injection field validasyonu

## Test

`ash
# Unit testleri çalıştırma
go test ./...

# Coverage report
go test -cover ./...

# Specific package test
go test ./internal/services/
`

## Deployment

### Docker

`dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o donationbars cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/donationbars .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
EXPOSE 8080
CMD ["./donationbars"]
`

### Docker Compose

`yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - MONGO_URI=mongodb://mongodb:27017
      - REDIS_ENABLED=true
      - REDIS_ADDR=redis:6379
    depends_on:
      - mongodb
      - redis

  mongodb:
    image: mongo:6.0
    ports:
      - "27017:27017"
    volumes:
      - mongo_data:/data/db

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  mongo_data:
`

## Performans

- API response time: ~150ms
- AI generation time: ~3-5 saniye
- Database query time: ~20ms
- Memory usage: ~60MB

### Optimizasyonlar

- MongoDB connection pooling (max: 10, min: 2)
- Context-based timeout yönetimi
- Redis cache ile rate limiting optimizasyonu
- Structured logging ile slog

## Sorun Giderme

### Yaygın Hatalar

| Hata | Çözüm |
|------|-------|
| connection refused | MongoDB servisini başlatın |
| invalid API key | OpenAI API key'inizi kontrol edin |
| 
ate limit exceeded | 24 saat bekleyin veya limiti artırın |
| injection field missing | HTML'de 5 injection field'ın da olduğundan emin olun |

### Debug

`ash
# Debug mode
export LOG_LEVEL=debug
go run cmd/main.go

# Health check
curl http://localhost:8080/health
`