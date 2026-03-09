# Güvenli Go & n8n Mikroservis Entegrasyonu 🚀

Bu proje, **Go (Golang)** tabanlı güvenli bir backend servisi ile **n8n** otomasyon aracının Docker üzerinde birbirleriyle nasıl güvenli bir şekilde haberleşebileceğini gösteren bir mikroservis mimarisidir. 

Sistem; yetkilendirme, veri tabanı yönetimi, ters vekil sunucusu (reverse proxy) ve güvenli webhook tetikleme gibi temel bileşenleri barındırır.

## 🌟 Özellikler

* **Kullanıcı Yönetimi:** Bcrypt ile şifrelenmiş güvenli kullanıcı kaydı ve girişi.
* **JWT Kimlik Doğrulama:** Uç noktaları korumak için JSON Web Token (JWT) tabanlı oturum yönetimi.
* **Rol Tabanlı Erişim Kontrolü (RBAC):** Sadece `admin` yetkisine sahip kullanıcıların kritik işlemleri gerçekleştirebilmesi.
* **HMAC Güvenliği:** Go üzerinden n8n'e gönderilen webhook isteklerinin SHA-256 HMAC imzasıyla korunması. Sadece doğrulanmış istekler n8n tarafından işlenir.
* **Ters Vekil (Reverse Proxy):** Nginx üzerinden gelen trafiğin Go uygulamasına güvenli bir şekilde yönlendirilmesi.
* **Konteyner Mimarisi:** Tüm servislerin (Go, PostgreSQL, n8n, Nginx) Docker Compose ile tek tıkla ayağa kaldırılabilmesi.

## 🛠 Kullanılan Teknolojiler

* **Backend:** Go (Gin Framework, GORM)
* **Veritabanı:** PostgreSQL
* **Otomasyon:** n8n
* **Sunucu:** Nginx
* **Konteynerizasyon:** Docker & Docker Compose
* **Frontend:** HTML, CSS, JavaScript (Gömülü şablonlar ile)

## 📂 Proje Yapısı

```text
.
├── docker-compose.yml       # Tüm servislerin tanımlandığı dosya
├── nginx.conf               # Nginx reverse proxy yapılandırması
├── go-app/
│   ├── main.go              # Go uygulamasının ana kaynak kodu
│   ├── Dockerfile           # Go uygulaması için Multi-stage build dosyası
│   ├── go.mod / go.sum      # Bağımlılıklar
│   └── templates/
│       └── index.html       # Kullanıcı arayüzü
```

## 🚀 Kurulum ve Çalıştırma
Projeyi yerel ortamınızda çalıştırmak için sisteminizde Docker ve Docker Compose kurulu olmalıdır.

### 1️⃣ Repoyu Klonlayın
```bash
git clone https://github.com/Ahmet-KURKCU/Go-Nginx-JWT-Auth-Service.git
cd Go-Nginx-JWT-Auth-Service
```

### 2️⃣ Konteynerleri Başlatın
```bash
docker-compose up --build -d
```

### 3️⃣ Uygulamaya Erişin
Tarayıcınızı açın ve aşağıdaki adrese gidin:
👉 http://localhost


### 4️⃣ n8n Akışını (Workflow) İçe Aktarma

Bu projedeki Go servisi, işlemlerini n8n otomasyon aracı üzerinden yürütmektedir. Sistemin uçtan uca çalışabilmesi için repoda bulunan hazır akışı n8n'e yüklemeniz gerekir:

1. Konteynerleri ayağa kaldırdıktan sonra tarayıcınızdan `http://localhost:5678` adresine giderek n8n arayüzüne giriş yapın.
2. Sol menüden **Workflows** sekmesine tıklayın ve **Add Workflow** (Yeni Akış Ekle) butonuna basın.
3. Sağ üst köşedeki üç nokta (`...`) menüsüne tıklayıp **Import from File** (Dosyadan İçe Aktar) seçeneğini seçin.
4. Bu repoda bulunan `n8n-workflow.json` dosyasını seçip yükleyin.
5. Yüklenen akışta **Webhook** düğümüne çift tıklayın ve *Authentication* (Kimlik Doğrulama) kısmında yeni bir Basic Auth kimliği oluşturun (Kullanıcı adı: `admin`, Şifre: `admin123` olmalıdır).
6. Sağ üst köşedeki anahtarı kullanarak akışı **Active** (Aktif) duruma getirin.

Artık Go uygulamanız üzerinden güvenli bir şekilde n8n'i tetikleyebilir ve geri dönüş (callback) alabilirsiniz! 🚀

## 🔐 Güvenlik Akışı (HMAC & JWT)

1. Kullanıcı sisteme kayıt olur ve şifresi `bcrypt` ile veritabanına kaydedilir.
2. Giriş yapıldığında bir `jwt_token` üretilir ve çerez (cookie) olarak tarayıcıya bırakılır.
3. Yetkili kullanıcı n8n'i tetiklemek istediğinde, Go uygulaması veriyi hazırlar ve gizli bir anahtarla **HMAC-SHA256** imzası oluşturur.
4. Bu imza `X-Hmac-Signature` başlığı ile n8n webhook'una gönderilir.

5. n8n işlemleri bitirdikten sonra, yetkisiz erişimleri engellemek için sadece tanımlı `X-CALLBACK-TOKEN` ile Go uygulamasına geri dönüş (callback) yapar.

