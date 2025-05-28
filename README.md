# Course GO Modular Monolith

## Create Simple Web server

- สร้างโปรเจคใหม่

  ```bash
  mkdir go-mma
  cd go-mma
  go mod init go-mma
  touch main.go
  ```

- Hello, World เปิดไฟล์ `main.go` แล้วใส่โค้ดนี้

  ```go
  package main

  import "github.com/gofiber/fiber/v3"

  func main() {
      app := fiber.New()

      app.Get("/", func(c fiber.Ctx) error {
          return c.SendString("Hello, World!")
      })

      app.Listen(":3000")
  }
  ```

- รัน `go mod tidy` เพื่อโหลด fiber

- รัน `go run main.go`

## ทดสอบเรียก API

- ติดตั้ง VS Code Extensions [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client)

- สร้างไฟล์ tests/hello.http

  ```text
  ### Hello World
  GET http://localhost:3000
  ```

- กดที่คำว่า Send Request

## รันด้วย Makefile

- สร้างไฟล์ `Makefile`

  ```make
  .PHONY: run

  run:
    go run main.go
  ```

- รันด้วยคำสั่ง `make run` แทน

## Configurations

จะเห็นว่ามีการ hard code หมายเลข port เอาไว้ ซึ่งถ้าต้องการแก้ไขค่านี้ จะต้อง build code ใหม่เสมอ ดังนั้นควรออกแบบระบบที่ยืดหยุ่น และสามารถรองรับได้หลาย environment

- สร้างไฟล์ `.env`

  ```text
  HTTP_PORT=8090
  ```

- สร้างไฟล์ `.gitignore` เพื่อไม่เอาไฟล์ `.env` เข้า git

  ```text
  .env
  ```

- แก้ไขไฟล์ `Makefile` เพื่อโหลดไฟล์ `.env`

  ```make
  include .env
  export

  .PHONY: run
  run:
    go run main.go
  ```

- สร้างไฟล์ `util/env/env.go` เพื่อโหลดค่า environment

  ```go
  package env

  import (
    "os"
    "strconv"
    "time"
  )

  func Get(key string) string {
    v, ok := os.LookupEnv(key)
    if !ok {
      return ""
    }
    return v
  }

  func GetDefault(key string, defaultValue string) string {
    v, ok := os.LookupEnv(key)
    if !ok {
      return defaultValue
    }
    return v
  }

  func GetInt(key string) int {
    v, err := strconv.Atoi(Get(key))
    if err != nil {
      return 0
    }
    return v
  }

  func GetIntDefault(key string, defaultValue int) int {
    v, err := strconv.Atoi(Get(key))
    if err != nil {
      return defaultValue
    }
    return v
  }
  func GetFloat(key string) float64 {
    v, err := strconv.ParseFloat(Get(key), 64)
    if err != nil {
      return 0.0
    }
    return v
  }

  func GetFloatDefault(key string, defaultValue float64) float64 {
    v, err := strconv.ParseFloat(Get(key), 64)
    if err != nil {
      return defaultValue
    }
    return v
  }

  func GetBool(key string) bool {
    v := Get(key)
    switch v {
    case "true", "yes":
      return true
    case "false", "no":
      return false
    default:
      return false
    }
  }
  func GetBoolDefault(key string, defaultValue bool) bool {
    v := Get(key)
    switch v {
    case "true", "yes":
      return true
    case "false", "no":
      return false
    default:
      return defaultValue
    }
  }

  func GetDuration(key string) time.Duration {
    v := Get(key)
    if len(v) == 0 {
      return 0
    }
    d, err := time.ParseDuration(v)
    if err != nil {
      return 0
    }
    return d
  }

  func GetDurationDefault(key string, defaultValue time.Duration) time.Duration {
    v := Get(key)
    if len(v) == 0 {
      return defaultValue
    }
    d, err := time.ParseDuration(v)
    if err != nil {
      return defaultValue
    }
    return d
  }

  ```

- แก้ไขไฟล์ `main.go`
  
  ```go
  // เปลี่ยนมาใช้งาน port จาก env
  app.Listen(fmt.Sprintf(":%d", env.GetInt("HTTP_PORT")))
  ```

- ทดสอบรัน `make run`

- แต่ปัญหาคือถ้าลืมระบุค่า port โปรแกรมจะทำงานไม่ถูกต้อง หรือไม่สามารถทำงานได้ ดังนั้นควรเพิ่มการตรวจสอบค่าที่จำเป็นต้องระบุ ก่อนเริ่มต้นโปรแกรม โดยเริ่มจากสร้างไฟล์ `config/config.go`

  ```go
  package config

  import (
    "errors"
    "go-mma/util/env"
  )

  var (
    ErrInvalidHTTPPort = errors.New("HTTP_PORT must be a positive integer")
  )

  type Config struct {
    HTTPPort int
  }

  func Load() (*Config, error) {
    config := &Config{
      HTTPPort: env.GetIntDefault("HTTP_PORT", 8090),
    }
    err := config.Validate()
    if err != nil {
      return nil, err
    }
    return config, err
  }

  func (c *Config) Validate() error {
    if c.HTTPPort <= 0 {
      return ErrInvalidHTTPPort
    }
    return nil
  }
  ```

- แก้ไขไฟล์ `main.go` เพื่อโหลด environment ถ้าไม่ค่าที่ต้องระบุ ให้จบการทำงานทันที

  ```go
  package main

  import (
    "fmt"
    "go-mma/config"
    "log"

    "github.com/gofiber/fiber/v3"
  )

  func main() {
    // เพิ่มโหลด config
    config, err := config.Load()
    // ถ้ามี error ให้จบการทำงาน
    if err != nil {
      log.Panic(err)
    }

    app := fiber.New()

    app.Get("/", func(c fiber.Ctx) error {
      return c.SendString("Hello, World!")
    })

    // เปลี่ยนมาใช้งาน port จาก env
    app.Listen(fmt.Sprintf(":%d", config.HTTPPort))
  }

  ```

## Graceful Shutdown

เรื่องถัดมาที่ควรทำ คือ การทำ Graceful Shutdown คือ รอให้ request ปัจจุบันทำงานเสร็จก่อนปิด และปิดการเชื่อมต่อฐานข้อมูลอย่างเหมาะสม

- แก้ไขไฟล์ main.go โดยลบบรรทัดนี้ออก

  ```go
  app.Listen(fmt.Sprintf(":%d", config.HTTPPort))
  ```

- และแทนที่ด้วย
  
  ```go
  package main

  import (
    "context"
    "fmt"
    "go-mma/config"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gofiber/fiber/v3"
  )

  func main() {
    // ...

    // Run server in goroutine
    go func() {
      if err := app.Listen(fmt.Sprintf(":%d", config.HTTPPort)); err != nil && err != http.ErrServerClosed {
        log.Fatalf("Error starting server: %v", err)
      }
    }()

    // Wait for shutdown signal
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    <-stop

    log.Println("Shutting down...")

    // Gracefully close fiber server
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := app.ShutdownWithContext(ctx); err != nil {
      log.Fatalf("Error shutting down server: %v", err)
    }

    // Optionally: close DB, cleanup, etc.

    log.Println("Shutdown complete.")
  }
  ```

- จะเห็นว่ามีการ hard code เวลา timeout ตรงนี้ให้แก้รับค่าจาก config แทน เช่น เพิ่ม env ชื่อ `GRACEFUL_TIMEOUT` ในไฟล์ `.env`

  ```env
  HTTP_PORT=3000
  GRACEFUL_TIMEOUT=5s
  ```

- แก้ไชไฟล์ `config/config.go` ดังนี้

  ```go
  package config

  import (
    "errors"
    "go-mma/util/env"
    "time"
  )

  var (
    ErrInvalidHTTPPort = errors.New("HTTP_PORT must be a positive integer")
    ErrGracefulTimeout = errors.New("GRACEFUL_TIMEOUT must be a positive duration")
  )

  type Config struct {
    HTTPPort        int
    GracefulTimeout time.Duration
  }

  func Load() (*Config, error) {
    config := &Config{
      HTTPPort:        env.GetIntDefault("HTTP_PORT", 8090),
      GracefulTimeout: parseDuration(env.GetDefault("GRACEFUL_TIMEOUT", "5s")),
    }
    err := config.Validate()
    if err != nil {
      return nil, err
    }
    return config, err
  }

  func (c *Config) Validate() error {
    if c.HTTPPort <= 0 {
      return ErrInvalidHTTPPort
    }
    if c.GracefulTimeout <= 0 {
      return ErrGracefulTimeout
    }

    return nil
  }

  func parseDuration(t string) time.Duration {
    d, _ := time.ParseDuration(t)
    return d
  }
  ```

- แก้ไฟล์ `main.go` ให้ใช้ค่าจาก config

  ```go
  ctx, cancel := context.WithTimeout(context.Background(), app.config.GracefulTimeout)
  ```

## Refactor Code

ตอนนี้ไฟล์ `main.go` เริ่มใหญ่ เราควรแยกส่วนออกไป

- เริ่มจากย้ายไฟล์ `main.go` ไปไว้ที่ `cmd/api/main.go`
- แก้ไขไฟล์ `Makefile` เพื่อแก้ไขตำแหน่งของ `main.go`

    ```makefile
    include .env
    export
    
    .PHONY: run
    run:
     go run cmd/api/main.go
    ```

- สร้างไฟล์ `application/http.go` เพื่อจัดการเกี่ยวกับ HTTP Server

    ```go
    package application
    
    import (
     "context"
     "fmt"
     "go-mma/config"
     "log"
     "net/http"
     "time"
    
     "github.com/gofiber/fiber/v3"
     "github.com/gofiber/fiber/v3/middleware/cors"
     "github.com/gofiber/fiber/v3/middleware/logger"
     "github.com/gofiber/fiber/v3/middleware/recover"
    )
    
    type HTTPServer interface {
     Start()
     Shutdown() error
     RegisterRoutes()
    }
    
    type httpServer struct {
     config config.Config
     app    *fiber.App
    }
    
    func newHTTPServer(config config.Config) HTTPServer {
     return &httpServer{
      config: config,
      app:    newFiber(config),
     }
    }
    
    func newFiber(config config.Config) *fiber.App {
     app := fiber.New(fiber.Config{
      AppName: fmt.Sprintf("Go MMA v%s", config.AppVersion),
     })
    
     // global middleware
     app.Use(logger.New())  // logs HTTP request/response details
     app.Use(recover.New()) // recovers from any panics
     app.Use(cors.New())    // allows all origins
    
     return app
    }
    
    func (s *httpServer) Start() {
     go func() {
      log.Printf("Starting server on port %d", s.config.HTTPPort)
      if err := s.app.Listen(fmt.Sprintf(":%d", s.config.HTTPPort)); err != nil && err != http.ErrServerClosed {
       log.Fatalf("Error starting server: %v", err)
      }
     }()
    }
    
    func (s *httpServer) Shutdown() error {
     ctx, cancel := context.WithTimeout(context.Background(), s.config.GracefulTimeout)
     defer cancel()
     return s.app.ShutdownWithContext(ctx)
    }
    
    func (s *httpServer) Router() *fiber.App {
     return s.app
    }
    
    func (s *httpServer) RegisterRoutes() {
     v1 := s.app.Group("/api/v1")
    
     customers := v1.Group("/customers")
     {
      customers.Post("", func(c fiber.Ctx) error {
       time.Sleep(3 * time.Second)
       return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": "c1"})
      })
     }
    
     orders := v1.Group("/orders")
     {
      orders.Post("", func(c fiber.Ctx) error {
       return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": "o1"})
      })
    
      orders.Delete("/:orderID", func(c fiber.Ctx) error {
       return c.SendStatus(fiber.StatusNoContent)
      })
     }
    }
    
    ```

- สร้างไฟล์ `application/application.go` เพื่อจัดการส่วนของการ start/stop โปรแกรม

    ```go
    package application
    
    import (
     "go-mma/config"
     "log"
    )
    
    type Application struct {
     config     config.Config
     httpServer HTTPServer
    }
    
    func New(config config.Config) *Application {
     return &Application{
      config:     config,
      httpServer: newHTTPServer(config),
     }
    }
    
    func (app *Application) Run() error {
     app.httpServer.Start()
    
     return nil
    }
    
    func (app *Application) Shutdown() error {
     // Gracefully close fiber server
     log.Println("Shutting down server")
     if err := app.httpServer.Shutdown(); err != nil {
      log.Fatalf("Error shutting down server: %v", err)
     }
     log.Println("Server stopped")
    
     return nil
    }
    
    func (app *Application) RegisterRoutes() {
     app.httpServer.RegisterRoutes()
    }
    ```

- แก้ไข `cmd/api/main.go` เพื่อเรียกใช้งาน application

    ```go
    package main
    
    import (
     "go-mma/application"
     "go-mma/config"
     "log"
     "os"
     "os/signal"
     "syscall"
    )
    
    func main() {
     config, err := config.Load()
     if err != nil {
      log.Panic(err)
     }
    
     app := application.New(*config)
     app.RegisterRoutes()
     app.Run()
    
     // Wait for shutdown signal
     stop := make(chan os.Signal, 1)
     signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
     <-stop
    
     app.Shutdown()
    
     // Optionally: close DB, cleanup, etc.
    
     log.Println("Shutdown complete.")
    }
    ```

    จะเห็นว่าไฟล์ `main.go` ดู clean ขึ้น
