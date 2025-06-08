# Course GO Modular Monolith

ในบทความนี้ จะพาไปดูว่าเราจะออกแบบโครงสร้างโปรเจคเป็นแบบ Modular Monolith ได้อย่างไร โดยจะพาทำไปทีละขั้นตอน และเปลี่ยนแปลงไปทีละนิด จนออกมาเป็น Modular Monolith

## เนื้อหาในบทความนี้

- สิ่งที่จะทำ
- Web Server
- Implement Handler
- เชื่อมต่อฐานข้อมูล
- จัดวางโครงสร้างแบบ Layered Architectue
- การจัดการ Error
- Database Transaction
- Unit of Work
- Dependency Inversion
- จัดวางโครงสร้างแบบ Modular
- ซ่อนรายละเอียดภายในของ subdomain
- Service Registry
- จัดวางโครงสร้างแบบ Mono-Repository
- Public API contract
- จัดโครงสร้างโมดูลแยกตาม feature
- Event-Driven Architecture

## สิ่งที่จะทำ

```markdown
+------------+        +----------------------+        +-----------+
|   Client   | <----> |    Monolith App      | <----> | Database  |
+------------+        |----------------------|        +-----------+
                      |  Modules:            |
                      |    - customer        |
                      |    - order           |
                      |    - email           |
                      +----------------------+

1. สร้างลูกค้าใหม่ (POST /customers)
---------------------------------------
Client ----> Monolith: POST /customers {email, credit}
Monolith.customer --> Database: ตรวจสอบ email ซ้ำ?
  └─ ซ้ำ --> Monolith.customer --> Client: 409 Conflict (email already exists)
  └─ ไม่ซ้ำ:
      Monolith.customer --> Database: INSERT INTO customers
      Monolith.email --> ส่งอีเมลต้อนรับ
      Monolith.customer --> Client: 201 Created

2. สั่งออเดอร์ (POST /orders)
-------------------------------
Client ----> Monolith: POST /orders {customer_id, order_total}
Monolith.order --> Database: ตรวจสอบ customer_id
  └─ ไม่พบ --> Monolith.order --> Client: 404 Not Found (customer not found)
  └─ พบ:
      Monolith.order --> Database: ตรวจสอบ credit เพียงพอ?
          └─ ไม่พอ --> Monolith.order --> Client: 422 Unprocessable Entity (insufficient credit)
          └─ พอ:
              Monolith.order --> Database: INSERT INTO orders, UPDATE credit (หักยอด)
              Monolith.email --> ส่งอีเมลยืนยันออเดอร์
              Monolith.order --> Client: 201 Created

3. ยกเลิกออเดอร์ (DELETE /orders/:orderID)
---------------------------------------------
Client ----> Monolith: DELETE /orders/:orderID
Monolith.order --> Database: ตรวจสอบ orderID
  └─ ไม่พบ --> Monolith.order --> Client: 404 Not Found (order not found)
  └─ พบ:
      Monolith.order --> Database: DELETE order, UPDATE credit (คืนยอด)
      Monolith.order --> Client: 204 No Content
```

## Web Server

เนื้อหาในส่วนนี้ประกอบด้วย

- สร้าง Web Server
- การจัดการ route
- การทดสอบ Rest API
- การทำ Graceful Shutdown
- สร้าง Logger ที่ใช้ได้ทั้งแอป
- การจัดการ Configurations
- รันด้วย Makefile
- Refactor Code

### สร้าง Web Server

เริ่มจากสร้าง Web Server ขึ้นมาก่อน โดยในบทความนี้จะภาษา Go และใช้ Fiber v3

- สร้างโปรเจคใหม่

    ```bash
    mkdir go-mma
    cd go-mma
    go mod init go-mma
    touch main.go
    ```

- จะได้แบบนี้

    ```bash
    tree
    .
    ├── go.mod
    └── main.go
    ```

- สร้าง Web Server ด้วย Fiber v3 ในไฟล์ `main.go`

    ```go
    package main
    
    import "github.com/gofiber/fiber/v3"
    
    func main() {
        app := fiber.New(fiber.Config{
       AppName: "Go MMA v0.0.1",
      })
    
        app.Get("/", func(c fiber.Ctx) error {
            return c.SendString("Hello, World!")
        })
    
        app.Listen(":8090")
    }
    ```

- รันคำสั่ง `go mod tidy` เพื่อติดตั้ง package
- รันคำสั่ง `go run main.go` รันโปรแกรม

    ```bash
    go run main.go
    
        _______ __             
       / ____(_) /_  ___  _____
      / /_  / / __ \/ _ \/ ___/
     / __/ / / /_/ /  __/ /    
    /_/   /_/_.___/\___/_/          v3.0.0-beta.4
    --------------------------------------------------
    INFO Server started on:         http://127.0.0.1:8090 (bound on host 0.0.0.0 and port 3000)
    INFO Application name:          Go MMA v0.0.1
    INFO Total handlers count:      1
    INFO Prefork:                   Disabled
    INFO PID:                       47664
    INFO Total process count:       1
    ```

- ทดสอบเปิด <http://127.0.0.1:8090> ผ่านเบราว์เซอร์

### การจัดการ route

ถัดมาเราจะมาเพิ่ม routes ตามที่ออกแบบไว้ทั้งหมด และจะมีการใช้งาน middlewares ที่จำเป็นด้วย

- แก้ไขไฟล์ `main.go`

    ```go
    package main
    
    import (
     "github.com/gofiber/fiber/v3"
     "github.com/gofiber/fiber/v3/middleware/cors"
     "github.com/gofiber/fiber/v3/middleware/logger"
     "github.com/gofiber/fiber/v3/middleware/recover"
     "github.com/gofiber/fiber/v3/middleware/requestid"
    )
    
    func main() {
     app := fiber.New(fiber.Config{
      AppName: "Go MMA v0.0.1",
     })
    
     // global middleware
     app.Use(cors.New())      // CORS ลำดับแรก เพื่อให้ OPTIONS request ผ่านได้เสมอ
     app.Use(requestid.New()) // สร้าง request id ใน request header สำหรับการ debug
     app.Use(recover.New())   // auto-recovers from panic (internal only)
     app.Use(logger.New())    // logs HTTP request
    
     v1 := app.Group("/api/v1")
    
     customers := v1.Group("/customers")
     {
      customers.Post("", func(c fiber.Ctx) error {
       return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": 1})
      })
     }
    
     orders := v1.Group("/orders")
     {
      orders.Post("", func(c fiber.Ctx) error {
       return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": 1})
      })
    
      orders.Delete("/:orderID", func(c fiber.Ctx) error {
       return c.SendStatus(fiber.StatusNoContent)
      })
     }
    
     app.Listen(":8090")
    }
    ```

- รันโปรแกรมใหม่ `go run main.go`

### การทดสอบ Rest API

ในบทความนี้จะใช้ VS Code Extensions ชื่อ [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client)

- ติดตั้ง [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client)
- สร้างไฟล์ใหม่ 2 ไฟล์ `tests/customers.http` กับ `test/orders.http`
- จะได้แบบนี้

    ```bash
    tree
    .
    ├── go.mod
    ├── main.go
    └── tests
      ├── customers.http
        └── orders.http
    ```

- แก้ไขไฟล์ `tests/customers.http`

    ```bash
    @host = http://localhost:8090
    @base_url = api/v1/customers
    ### Create Customer
    POST {{host}}/{{base_url}} HTTP/1.1
    content-type: application/json
    
    {
      "email": "cust@example.com",
      "credit": 1000
    }
    ```

- แก้ไขไฟล์ `tests/orders.http`

    ```bash
    @host = http://localhost:8090
    @base_url = api/v1/orders
    @order_id = o1
    ### Create Order
    POST {{host}}/{{base_url}} HTTP/1.1
    content-type: application/json
    
    {
      "customer_id": 1,
      "order_total": 10
    }
    
    ### Cancel Order
    DELETE {{host}}/{{base_url}}/{{order_id}} HTTP/1.1
    ```

- ทดสอบเรียก API โดยการกดที่คำว่า `Send Request`

    ```bash
    @host = http://localhost:8090
    @base_url = api/v1/customers
    ### Create Customer
    Send Request
    POST {{host}}/{{base_url}} HTTP/1.1
    content-type: application/json
    
    {
      "email": "cust@example.com",
      "credit": 1000
    }
    ```

- จะได้ผลลัพธ์แบบนี้

    ```bash
    HTTP/1.1 201 Created
    Date: Thu, 29 May 2025 04:07:11 GMT
    Content-Type: application/json
    Content-Length: 11
    Connection: close
    
    {
      "id": 1
    }
    ```

### **การทำ Graceful Shutdown**

สิ่งหนึ่งที่มักจะมองข้าม แต่สำคัญมาก คือ การทำ Graceful Shutdown หรือ การรอให้ request ปัจจุบันทำงานให้เสร็จก่อนปิด และปิดการเชื่อมต่อต่างๆ เช่น ฐานข้อมูล อย่างเหมาะสม

ในภาษา Go ทำได้ง่ายๆ ดังนี้

- เริ่มจากย้ายจุดที่ start server ไปทำงานใน goroutines แทน

    ```go
    // Run server in goroutine
    go func() {
      if err := app.Listen(":8090"); err != nil && err != http.ErrServerClosed {
        log.Fatalf("Error starting server: %v", err)
      }
    }()
    ```

- ถัดมา ให้หยุดรอสัญญาณการหยุดระบบ โดยเพิ่มโค้ดนี้ ในบรรทัดถัดไป

    ```go
    // Wait for shutdown signal
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    <-stop
    ```

- เมื่อได้รับสัญญาณมาแล้ว ให้เริ่มจากหยุดรับ Request ใหม่ แล้วรอจนกว่า request เดิมจะทำงานจนเสร็จทั้งหมด หรือกำหนดระยะรอคอยก็ได้ เช่น ต้องทำให้เสร็จภายใน 5 วินาที ถ้าไม่เสร็จก็ปิด server ไปเลย ถัดมาจึงค่อยมาปิดการเชื่อมต่ออื่นๆ โดยเพิ่มโค้ดนี้ ในบรรทัดถัดไป

    ```go
    log.Println("Shutting down...")
    
    // Gracefully close fiber server
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := app.ShutdownWithContext(ctx); err != nil {
      log.Fatalf("Error shutting down server: %v", err)
    }
    
    // Optionally: close DB, cleanup, etc.
    
    log.Println("Shutdown complete.")
    ```

- ทดสอบแก้ไขโค้ดของการสร้าง Customer เพื่อหน่วงเวลา 3 วินาที

    ```go
    customers.Post("", func(c fiber.Ctx) error {
     time.Sleep(3 * time.Second)
     return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": 1})
    })
    ```

- เมื่อลองรันใหม่ แล้วเรียก API สร้าง customer จากนั้นก็กด Ctrl+C ทันที จะเห็นว่าระบบจะรอให้ request เดิมทำงานให้เสร็จก่อน จึงค่อยหยุดระบบไป

    ```go
    2025/05/29 12:00:07 Shutting down...
    12:00:06 | 201 |  3.001456625s |       127.0.0.1 | POST    | /api/v1/customers       
    2025/05/29 12:00:09 Shutdown complete.
    ```

### สร้าง Logger ที่ใช้ได้ทั้งแอป

จากหัวที่แล้ว จะสังเกตได้ว่าการแสดงผลของ log มีรูปแบบที่แตกต่างกัน ซึ่งจุดนี้เราควรกำหนดรูปแบบของ log ให้เหมือนกันทั้งระบบ โดยการสร้าง logger ขค้นมาจัดการ ดังนี้

- สร้างไฟล์ `util/logger/logger.go`

    ```go
    package logger
    
    import (
     "go.elastic.co/ecszap"
     "go.uber.org/zap"
    )
    
    type closeLog func() error
    
    var Log *zap.Logger
    
    func Init() (closeLog, error) {
     config := zap.NewDevelopmentConfig()
     config.EncoderConfig = ecszap.ECSCompatibleEncoderConfig(config.EncoderConfig)
    
     var err error
     Log, err = config.Build(ecszap.WrapCoreOption())
    
     if err != nil {
      return nil, err
     }
    
     return func() error {
      return Log.Sync()
     }, nil
    }
    
    func With(fields ...zap.Field) *zap.Logger {
     return Log.With(fields...)
    }
    ```

- รันคำสั่ง `go mod tidy` เพื่อติดตั้ง package
- ทำการ initialize logger ที่ `main.go`

    ```go
    func main() {
     closeLog, err := logger.Init()
     if err != nil {
      panic(err.Error())
     }
     defer closeLog()
    
     // ...
    }
    ```

- แก้ทุกที่ ที่ใช้ `log` มาเป็น `logger` แทน ตัวอย่าง เช่น

    ```go
     func main() {
     // ...
     
     // Run server in goroutine
     go func() {
       if err := app.Listen(":8090"); err != nil && err != http.ErrServerClosed {
         logger.Log.Fatal(fmt.Sprintf("Error starting server: %v", err))
       }
     }()
     
     // Wait for shutdown signal
     stop := make(chan os.Signal, 1)
     signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
     <-stop
     
     logger.Log.Info("Shutting down...")
     
     // Gracefully close fiber server
     ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
     defer cancel()
     if err := app.ShutdownWithContext(ctx); err != nil {
       logger.Log.Fatal(fmt.Sprintf("Error shutting down server: %v", err))
     }
     
     // Optionally: close DB, cleanup, etc.
    
     logger.Log.Info("Shutdown complete.")
    }
    ```

- ทำการสร้าง `RequestLogger` middleware ไว้ที่ไฟล์ `application/middleware/request_logger.go` เพื่อใช้ logger ในการแสดงผล

    ```go
    package middleware
    
    import (
     "fmt"
     "go-mma/util/logger"
     "runtime/debug"
     "time"
    
     "github.com/gofiber/fiber/v3"
     "go.uber.org/zap"
    )
    
    func RequestLogger() fiber.Handler {
     return func(c fiber.Ctx) error {
      start := time.Now()
    
      log := logger.With(
       zap.String("requestId", c.GetRespHeader("X-Request-ID")),
       zap.String("method", c.Method()),
       zap.String("path", c.Path()),
      )
    
        // catch panic
      defer func() {
       if r := recover(); r != nil {
        printAccessLog(log, c.Method(), c.Path(), start, fiber.StatusInternalServerError, r)
        panic(r) // throw panic to recover middleware
       }
      }()
    
      err := c.Next()
    
      status := c.Response().StatusCode()
      if err != nil {
       switch e := err.(type) {
       case *fiber.Error:
        status = e.Code
       default: // case error
        status = fiber.StatusInternalServerError
       }
      }
    
      printAccessLog(log, c.Method(), c.Path(), start, status, err)
    
      return err
     }
    }
    
    func printAccessLog(log *zap.Logger, method string, uri string, start time.Time, status int, err any) {
     if err != nil {
      // log unhandle error
      log.Error("an error occurred",
       zap.Any("error", err),
       zap.ByteString("stack", debug.Stack()),
      )
     }
    
     msg := fmt.Sprintf("%d - %s %s", status, method, uri)
    
     log.Info(msg,
      zap.Int("status", status),
      zap.Duration("latency", time.Since(start)))
    }
    
    ```

- เปลี่ยนมาใช้ `RequestLogger` middleware แทน ในไฟล์ `main.go`

    ```go
    // global middleware
    app.Use(cors.New())                 // CORS ลำดับแรก เพื่อให้ OPTIONS request ผ่านได้เสมอ
    app.Use(requestid.New())            // สร้าง request id ใน request header สำหรับการ debug
    app.Use(recover.New())              // auto-recovers from panic (internal only)
    app.Use(middleware.RequestLogger()) // logs HTTP request
    ```

- เมื่อลองรันใหม่อีกครั้ง จะเห็นว่ารูปแบบการแสดง log จะเหมือนกันทั้งหมดแล้ว

### การจัดการ Configurations

จากโค้ดข้างบน มีค่าบางอย่าง เช่น HTTP Port และระยะเวลารอคอยการปิด server นั้น ควรสามารถปรับเปลี่ยนได้ตามแต่ละ environments ที่นำระบบไปรัน แต่ปัจจุบันนจะต้องมาแก้ไขโค้ด แล้วทำการ build ใหม่ทุกครั้ง

ซึ่งควรออกแบบให้ระบบยืดหยุ่น โดยสามารถกำหนดค่าต่างๆ ได้จาก system environments แทน

- สร้างไฟล์ `util/env/env.go` เพื่อเป็นตัวช่วยอ่านค่า environment

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

- แก้โค้ดในไฟล์ `main.go` ให้ใช้ค่าจาก environment แทน

    ```go
    func main() {
     // ...
     go func() {
       // ถ้าไม่กำหนด env มาให้ default 8090
      if err := app.Listen(fmt.Sprintf(":%d", env.GetIntDefault("HTTP_PORT", 8090))); err != nil && err != http.ErrServerClosed {
       // ...
      }
     }()
     // ...
     
     // ถ้าไม่กำหนด env มาให้ default 5 วินาที
     ctx, cancel := context.WithTimeout(context.Background(), env.GetDurationDefault("GRACEFUL_TIMEOUT", 5*time.Second))
     // ...
    }
    ```

- รันโปรแกรมใหม่ พร้อมกำหนดค่า env

    ```bash
    HTTP_PORT=8091 GRACEFUL_TIMEOUT=10s go run main.go
    ```

- เพื่อรองรับ config ที่อาจเกิดขึ้นในอนาคต จะทำการสร้าง package `config` ขึ้นมาสำหรับจัดการโหลดค่า env ทั้งหมด โดยให้สร้างไฟล์ `config/config.go`

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
      GracefulTimeout: env.GetDurationDefault("GRACEFUL_TIMEOUT", 5*time.Second),
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
    ```

- เรียกใช้งานใน `main.go`

    ```go
    func main() {
      // logger
     config, err := config.Load()
     if err != nil {
      log.Panic(err)
     }
     
      // ...
      
     go func() {
      if err := app.Listen(fmt.Sprintf(":%d", config.HTTPPort)); err != nil && err != http.ErrServerClosed {
        // ...
      }
     }()
     
     // ...
     
     ctx, cancel := context.WithTimeout(context.Background(), config.GracefulTimeout)
     
     // ...
    }
    ```

### **รันด้วย Makefile**

เนื่องจากถ้ารันแบบปกติจะต้องมีการกำหนด env ลงไปด้วยทุกครั้ง ทำให้ไม่สะดวกในการพิมพ์คำสั่ง ดังนั้น จะแนะนำให้รันผ่าน Makefile แทน ซึ่งมีขั้นตอนดังนี้

- สร้างไฟล์ `.env`

    ```
    HTTP_PORT=8090
    GRACEFUL_TIMEOUT=5s
    ```

- สร้างไฟล์ `.gitignore` เพื่อไม่เอาไฟล์ `.env` เข้า git

    ```
    .env
    ```

    <aside>
    💡

    ถ้าหากต้องการให้มีตัวอย่างการ config ให้คัดลอกไฟล์ไปเป็น `.env.example` แทน และต้องไม่ใส่ค่าที่เป็นความลับเอาไว้

    </aside>

- สร้างไฟล์ `Makefile`

    ```makefile
    include .env
    export
    
    .PHONY: run
    run:
     go run main.go
    ```

- รันโปรแกรมด้วยคำสั่ง `make run`

    ```bash
    make run
    
        _______ __             
       / ____(_) /_  ___  _____
      / /_  / / __ \/ _ \/ ___/
     / __/ / / /_/ /  __/ /    
    /_/   /_/_.___/\___/_/          v3.0.0-beta.4
    --------------------------------------------------
    INFO Server started on:         http://127.0.0.1:8090 (bound on host 0.0.0.0 and port 8090)
    INFO Application name:          Go MMA v0.0.1
    INFO Total handlers count:      6
    INFO Prefork:                   Disabled
    INFO PID:                       31427
    INFO Total process count:       1
    ```

### Refactor Code

ตอนนี้ไฟล์ `main.go` เริ่มมีขนาดใหญ่ จึงควรที่จะทำการแยกส่วนออกไป

- เริ่มจากย้ายไฟล์ `main.go` ไปไว้ที่ `cmd/api/main.go`
- แก้ไขไฟล์ `Makefile` เพื่อแก้ไขตำแหน่งของ `main.go` ใหม่

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
     "github.com/gofiber/fiber/v3/middleware/requestid"
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
      app:    newFiber(),
     }
    }
    
    func newFiber() *fiber.App {
     app := fiber.New(fiber.Config{
      AppName: "Go MMA v0.0.1",
     })
    
     // global middleware
     app.Use(cors.New())      // CORS ลำดับแรก เพื่อให้ OPTIONS request ผ่านได้เสมอ
     app.Use(requestid.New()) // สร้าง request id ใน request header สำหรับการ debug
     app.Use(recover.New())   // auto-recovers from panic (internal only)
     app.Use(logger.New())    // logs HTTP request
    
     return app
    }
    
    func (s *httpServer) Start() {
     go func() {
      logger.Log.Info(fmt.Sprintf("Starting server on port %d", s.config.HTTPPort))
      if err := s.app.Listen(fmt.Sprintf(":%d", s.config.HTTPPort)); err != nil && err != http.ErrServerClosed {
       logger.Log.Fatal(fmt.Sprintf("Error starting server: %v", err))
      }
     }()
    }
    
    func (s *httpServer) Shutdown() error {
     ctx, cancel := context.WithTimeout(context.Background(), s.config.GracefulTimeout)
     defer cancel()
     return s.app.ShutdownWithContext(ctx)
    }
    
    func (s *httpServer) RegisterRoutes() {
     v1 := s.app.Group("/api/v1")
    
     customers := v1.Group("/customers")
     {
      customers.Post("", func(c fiber.Ctx) error {
       time.Sleep(3 * time.Second)
       return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": 1})
      })
     }
    
     orders := v1.Group("/orders")
     {
      orders.Post("", func(c fiber.Ctx) error {
       return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": 1})
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
     logger.Log.Info("Shutting down server")
     if err := app.httpServer.Shutdown(); err != nil {
      logger.Log.Fatal(fmt.Sprintf("Error shutting down server: %v", err))
     }
     logger.Log.Info("Server stopped")
    
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
     // logger
     // config
    
     app := application.New(*config)
     app.RegisterRoutes()
     app.Run()
    
     // Wait for shutdown signal
     // stop
     
     app.Shutdown()
    
     // Optionally: close DB, cleanup, etc.
    
     logger.Log.Info("Shutdown complete.")
    }
    ```

    จะเห็นว่าไฟล์ `main.go` ดู clean ขึ้น

## Implement Handler

จากโจทย์ที่จะทำสามารถออกแบบ API ได้ 3 เส้น ดังนี้

- `POST /customers` – สร้างลูกค้าใหม่

    | JSON Field | Type | Required | Description |
    | --- | --- | --- | --- |
    | `email` | string | ✅ | อีเมลลูกค้า |
    | `credit` | number | ✅ | เครดิตเริ่มต้นของลูกค้า |

    **Response**

    | Status Code | Description |
    | --- | --- |
    | `201` | สร้างลูกค้าเรียบร้อย |
    | `400` | ไม่ส่ง `email`, `email` ผิดรูปแบบ หรือ `credit` ≤ 0 |
    | `409` | อีเมลนี้มีอยู่แล้วในระบบ (Conflict) |

- `POST /orders` – สร้างออเดอร์

    | JSON Field | Type | Required | Description |
    | --- | --- | --- | --- |
    | `customer_id` | integer | ✅ | ID ของลูกค้าที่จะสั่งออเดอร์ |
    | `order_total` | number | ✅ | ยอดรวมของออเดอร์ |

    **Response**

    | Status Code | Description |
    | --- | --- |
    | `201` | สร้างออเดอร์เรียบร้อย |
    | `400` | ไม่ส่ง `customer_id` หรือ `order_total` ≤ 0 |
    | `404` | ไม่พบลูกค้า (`customer_id` ไม่ถูกต้อง) |
    | `422` | เครดิตไม่เพียงพอในการสั่งออเดอร์ |

- `DELETE /orders/:orderID` – ยกเลิกออเดอร์

    | Path Param | Type | Required | Description |
    | --- | --- | --- | --- |
    | `orderID` | integer | ✅ | ID ของออเดอร์ที่จะยกเลิก |

    **Response**

    | Status Code | Description |
    | --- | --- |
    | `204` | ลบออเดอร์สำเร็จ (No Content) |
    | `404` | ไม่พบออเดอร์นี้ในระบบ |

โดยให้ทำการสร้างไฟล์ 2 ไฟล์

- `handlers/customer.go` สำหรับสร้างลูกค้ารายใหมา
- `handler/order.go` สำหรับสั่ง และยกเลิกออเดอร์

### Customer Handler

- แก้ไขไฟล์ `handlers/customer.go` ตามนี้

    ```go
    package handlers
    
    import (
     "fmt"
     "net/mail"
    
     "github.com/gofiber/fiber/v3"
    )
    
    type CustomerHandler struct {
    }
    
    func NewCustomerHandler() *CustomerHandler {
     return &CustomerHandler{}
    }
    
    func (h *CustomerHandler) CreateCustomer(c fiber.Ctx) error {
     type CreateCustomerRequest struct {
      Email  string `json:"email"`
      Credit int    `json:"credit"`
     }
     var req CreateCustomerRequest
     if err := c.Bind().Body(&req); err != nil {
      return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
     }
    
     logger.Log.Info("Received customer:", req)
    
     // Validate payload fields
     if req.Email == "" {
      return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "email is required"})
     }
     if _, err := mail.ParseAddress(req.Email); err != nil{
      return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "email is invalid"})
     }
     if req.Credit <= 0 {
      return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "credit must be greater than 0"})
     }
    
     // TODO: save new customer to the database
     var id int
    
     // Return a created response
     type CreateCustomerResponse struct {
      ID int `json:"id"`
     }
     resp := &CreateCustomerResponse{ID: id}
     return c.Status(fiber.StatusCreated).JSON(resp})
    }
    ```

- แก้ไขไฟล์ `application/http.go` เพื่อเรียกใช้ CustomerHandler

    ```go
    customers := v1.Group("/customers")
    {
     hdlr := handlers.NewCustomerHandler()
     customers.Post("", hdlr.CreateCustomer)
    }
    ```

### Order Handler

- แก้ไขไฟล์ `handlers/order.go` ตามนี้

    ```go
    package handlers
    
    import (
     "fmt"
    
     "github.com/gofiber/fiber/v3"
    )
    
    type OrderHandler struct {
    }
    
    func NewOrderHandler() *OrderHandler {
     return &OrderHandler{}
    }
    
    func (h *OrderHandler) CreateOrder(c fiber.Ctx) error {
     type CreateOrderRequest struct {
      CustomerID string `json:"customer_id"`
      OrderTotal int    `json:"order_total"`
     }
     var req CreateOrderRequest
     if err := c.Bind().Body(&req); err != nil {
      return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
     }
    
     logger.Log.Info("Received Order:", req)
    
     // Validate payload fields
     if req.CustomerID == "" {
      return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "customer_id is required"})
     }
     if req.OrderTotal <= 0 {
      return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "order_total must be greater than 0"})
     }
    
     // TODO: get customer by ID from the database
     // customer := getCustomer(order.CustomerID)
     // if customer == nil {
     //  return return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "the customer with given id was not found"})
     // }
    
     // TODO: check credit balance of the customer
     // if credit < payload.OrderTotal {
     //  return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "insufficient credit"})
     // }
    
     // TODO: reserve credit for the customer
    
     // TODO: update customer's credit balance in the database
    
     // TODO: save new Order to the database
     var id int
    
     // Return a created response
     type CreateOrderResponse struct {
      ID int `json:"id"`
     }
     resp := &CreateCustomerResponse{ID: id}
     return c.Status(fiber.StatusCreated).JSON(resp)
    }
    
    func (h *OrderHandler) CancelOrder(c fiber.Ctx) error {
     // Implement the logic to cancel an order
     orderID, err := strconv.Atoi(c.Params("orderID"))
     if err != nil {
      return errs.InputValidationError("invalid order id")
     }
    
     fmt.Println("Cancelling order:", orderID)
    
     // TODO: get order details from the database
     // order := getOrder(orderID)
     // if order == nil {
     //  return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "the order with given id was not found"})
     // }
    
     // TODO: get cutomer details from the database
     // customer := getCustomer(order.CustomerID)
     // if customer == nil {
     //  return return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "the customer with given id was not found"})
     // }
    
     // TODO: release credit limit for the customer
     // creditLimit += CreateOrderRequest.OrderTotal
     // TODO: save the customer details to the database
    
     // TODO: update the order status in the database
    
     return c.SendStatus(fiber.StatusNoContent)
    }
    
    ```

- แก้ไขไฟล์ `application/http.go` เพื่อเรียกใช้ OrderHandler

    ```go
    orders := v1.Group("/orders")
     {
      hdlr := handlers.NewOrderHandler()
      orders.Post("", hdlr.CreateOrder)
      orders.Delete("/:orderID", hdlr.CancelOrder)
     }
    ```

## เชื่อมต่อฐานข้อมูล

จากโค้ดด้านบนจะเห็นว่าไม่สามารถเขียนต่อให้เสร็จได้ เนื่องจากจำเป็นต้องมีการบันทึกลงฐานข้อมูลก่อน โดยในส่วนนี้จะประกอบด้วย

- ติดตั้ง PostgreSQL
- ออกแบบฐานข้อมูล
- การทำ Database migration
- เชื่อมต่อฐานข้อมูล
- Dependency Injection
- บันทึก customer ลงฐานข้อมูล

### ติดตั้ง PostgreSQL

ในบทความนี้จะใช้ PostgreSQL โดยจะติดตั้งด้วย docker

- สร้างไฟล์ `docker-compose.yml`

    ```yaml
    services:
      db:
        image: postgres:17-alpine
        container_name: go-mma-db
        volumes:
          - pg_data:/var/lib/postgresql/data
    
    volumes:
      pg_data:
    ```

- สร้างไฟล์ `docker-compose.dev.yml`

    ```yaml
    services:
      db:
        environment:
          POSTGRES_DB: go-mma-db
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: postgres
        ports:
          - 5433:5432
    ```

- รัน PostgreSQL Server ด้วย `Makefile` โดยเพิ่มคำสั่ง ดังนี้

    ```bash
    
    .PHONY: devup
    devup:
     docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d
    
    .PHONY: devdown
    devdown:
     docker compose -f docker-compose.yml -f docker-compose.dev.yml down
    ```

- รันคำสั่ง `make devup`

### ออกแบบฐานข้อมูล

```sql
+------------------+           +------------------+
|   customers      |           |     orders       |
+------------------+           +------------------+
| id (PK)          |<--------+ | id (PK)          |
| email (UNIQUE)   |         | | customer_id (FK) |
| credit           |         | | order_total      |
| created_at       |         | | created_at       |
| updated_at       |         | | canceled_at       |
+------------------+         | +------------------+
                             |
      [1] ---------------- [*]
     1 customer      →   many orders
```

### การทำ Database migration

การทำ Database Migration คือ กระบวนการจัดการการเปลี่ยนแปลงโครงสร้างของฐานข้อมูล (เช่น ตาราง, คอลัมน์, ดัชนี ฯลฯ) อย่างเป็นระบบ ผ่านชุดของสคริปต์หรือโค้ดที่สามารถรันซ้ำได้อย่างปลอดภัยในทุก environment (เช่น dev, staging, prod)

- เพิ่ม env ชื่อ `DB_DSN` ในไฟล์ `.env`

    ```
    DB_DSN=postgres://postgres:postgres@localhost:5433/go-mma-db?sslmode=disable
    ```

- แก้ไขไฟล์ `Makefile` เพื่อรันคำสั่ง migration

    ```makefile
    ROOT_DIR := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))
    
    .PHONY: mgc
    # Example: make mgc filename=create_customer
    mgc:
     docker run --rm -v $(ROOT_DIR)migrations:/migrations migrate/migrate -verbose create -ext sql -dir /migrations $(filename)
    
    .PHONY: mgu
    mgu:
     docker run --rm --network host -v $(ROOT_DIR)migrations:/migrations migrate/migrate -verbose -path=/migrations/ -database "$(DB_DSN)" up
    
    .PHONY: mgd
    mgd:
     docker run --rm --network host -v $(ROOT_DIR)migrations:/migrations migrate/migrate -verbose -path=/migrations/ -database $(DB_DSN) down 1
    ```

    <aside>
    💡

    ถ้าใช้ Docker Desktop ต้องเปิด host networking ก่อน ไปที่ `Setting → Resources → Network` เลือก Enable host networking แล้ว Apply & restart

    </aside>

- สร้างไฟล์ migration สำหรับสร้าง customer

    ```bash
    make mgc filename=create_customer
    ```

    จะไฟล์ออกมา 2 ไฟล์

    ```bash
    ./migrations/20250529103238_create_customer.up.sql
    ./migrations/20250529103238_create_customer.down.sql
    ```

    แก้ไขไฟล์ `create_customer.up.sql`

    ```sql
    CREATE TABLE public.customers (
     id BIGINT NOT NULL,
     email text NOT NULL,
     credit int4 NOT NULL,
     created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
     updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
     CONSTRAINT customers_pkey PRIMARY KEY (id),
     CONSTRAINT customers_unique UNIQUE (email)
    );
    ```

    แก้ไขไฟล์ `create_customer.down.sql`

    ```sql
    drop table public.customers;
    ```

- สร้างไฟล์ migration สำหรับสร้าง order

    ```bash
    make mgc filename=create_order
    ```

    จะไฟล์ออกมา 2 ไฟล์

    ```bash
    ./migrations/20250529103715_create_order.up.sql
    ./migrations/20250529103715_create_order.down.sql
    ```

    แก้ไขไฟล์ `create_order.up.sql`

    ```sql
    CREATE TABLE public.orders (
     id BIGINT NOT NULL,
     customer_id BIGINT NOT NULL,
     order_total int4 NOT NULL,
     created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
     canceled_at timestamp NULL,
     CONSTRAINT orders_pkey PRIMARY KEY (id),
     CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES public.customers(id)
    );
    ```

    แก้ไขไฟล์ `create_order.down.sql`

    ```sql
    drop table public.orders;
    ```

- รันคำสั่ง migration เพื่อสร้างตารางทั้งหมด

    ```bash
    make mgu
    
    2025/05/29 10:39:20 Start buffering 20250529103238/u create_customer
    2025/05/29 10:39:20 Start buffering 20250529103715/u create_order
    2025/05/29 10:39:20 Read and execute 20250529103238/u create_customer
    2025/05/29 10:39:20 Finished 20250529103238/u create_customer (read 906.667µs, ran 2.125583ms)
    2025/05/29 10:39:20 Read and execute 20250529103715/u create_order
    2025/05/29 10:39:20 Finished 20250529103715/u create_order (read 3.458625ms, ran 1.860583ms)
    2025/05/29 10:39:20 Finished after 7.190625ms
    2025/05/29 10:39:20 Closing source and database
    ```

### เชื่อมต่อฐานข้อมูล

- เพิ่ม config สำหรับ `DB_DSN` โดยแก้ไขไฟล์ `config/config.go`

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
     ErrDSN             = errors.New("DB_DSN must be set")
    )
    
    type Config struct {
     HTTPPort        int
     GracefulTimeout time.Duration
     DSN             string
    }
    
    func Load() (*Config, error) {
     config := &Config{
      HTTPPort:        env.GetIntDefault("HTTP_PORT", 8090),
      GracefulTimeout: env.GetDurationDefault("GRACEFUL_TIMEOUT", 5*time.Second),
      DSN:             env.Get("DB_DSN"),
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
     if len(c.DSN) == 0 {
      return ErrDSN
     }
    
     return nil
    }
    ```

- สร้างไฟล์ `util/storage/sqldb/sqldb.go` สำหรับสร้าง database connection

    ```go
    package sqldb
    
    import (
     "github.com/jmoiron/sqlx"
     _ "github.com/lib/pq"
    )
    
    type closeDB func() error
    
    type DBContext interface {
     DB() *sqlx.DB
    }
    
    type dbContext struct {
     db *sqlx.DB
    }
    
    var _ DBContext = (*dbContext)(nil)
    
    func New(dsn string) (DBContext, closeDB, error) {
     // this Pings the database trying to connect
     db, err := sqlx.Connect("postgres", dsn)
     if err != nil {
      return nil, nil, err
     }
     return &dbContext{db: db},
      func() error {
       return db.Close()
      },
      nil
    }
    
    func (c *dbContext) DB() *sqlx.DB {
     return c.db
    }
    ```

- รันคำสั่ง `go mod tidy` เพื่อติดตั้ง package
- แก้ไขไฟล์ `application/application.go` เพื่อเก็บ database connection

    ```go
    type Application struct {
     config     config.Config
     httpServer HTTPServer
     db         sqldb.DBContext
    }
    
    func New(config config.Config, db sqldb.DBContext) *Application {
     return &Application{
      config:     config,
      httpServer: newHTTPServer(config),
      db:         db,
     }
    }
    ```

- สร้าง database connection ใน `cmd/api/main.go` และส่งไปให้ `application/application.go`

    ```go
    func main() {
     // config
     
     db, closeDB, err := sqldb.New(config.DSN)
     if err != nil {
      logger.Log.Panic(err)
     }
     defer func() {
      if err := closeDB(); err != nil {
       logger.Log.Info("Error closing database:", err)
      }
     }()
    
     app := application.New(*config, db)
     // ...
    }
    ```

### Generate ID

เพิ่มฟังก์ชันสำหรับสร้าง ID ของตารางต่างๆ

- สร้างไฟล์ `util/genid/genid.go`

    ```go
    package idgen
    
    import (
     "fmt"
     "math/rand"
     "strconv"
     "strings"
     "time"
    )
    
    // ใช้แค่ตัวนี้ ที่เหลือแถม
    // GenerateTimeRandomID สร้าง ID แบบ int64
    func GenerateTimeRandomID() int64 {
     timestamp := time.Now().UnixNano() >> 32
     randomPart := rand.Int63() & 0xFFFFFFFF
     return (timestamp << 32) | randomPart
    }
    
    // GenerateTimeID สร้าง ID แบบ int (ใช้ timestamp เป็นหลัก)
    func GenerateTimeID() int {
     // ใช้ timestamp Unix วินาที (int64) แปลงเป็น int (int32/64 ขึ้นกับระบบ)
     return int(time.Now().Unix())
    }
    
    // GenerateTimeRandomIDBase36 คืนค่า ID เป็น base36 string
    func GenerateTimeRandomIDBase36() string {
     id := GenerateTimeRandomID()
     return strconv.FormatInt(id, 36) // แปลงเลขฐาน 10 -> 36
    }
    
    // GenerateUUIDLikeID คืนค่าเป็น string แบบ UUID-like (แต่ไม่ใช่ UUID จริง)
    func GenerateUUIDLikeID() string {
     id := GenerateTimeRandomID()
    
     // แปลง int64 เป็น hex string ยาว 16 ตัว (64 bit)
     hex := fmt.Sprintf("%016x", uint64(id))
    
     // สร้าง UUID-like string รูปแบบ 8-4-4-4-12
     // แต่มีแค่ 16 hex chars แบ่งคร่าวๆ: 8-4-4 (เหลือไม่พอจริงๆ)
     // ดังนั้นเราจะเติม random เพิ่มเพื่อครบ 32 hex (128 bit) เหมือน UUID
    
     randPart := fmt.Sprintf("%016x", rand.Uint64())
    
     uuidLike := strings.Join([]string{
      hex[0:8],
      hex[8:12],
      hex[12:16],
      randPart[0:4],
      randPart[4:16],
     }, "-")
    
     return uuidLike
    }
    
    // ก่อน Go 1.20 ต้องเรียก เพื่อให้ได้เลขสุ่มไม่ซ้ำ
    // func init() {
    //  rand.Seed(time.Now().UnixNano())
    // }
    ```

### Dependency Injection

โจทย์ถัดมา คือ เราจะเรียกใช้ database connection ใน handlers ได้อย่างไร ซึ่งในบทความนี้จะใช้วิธี Dependency Injection คือ ส่ง db เข้าไปในตอนสร้าง handler

- แก้ไขไฟล์ `handlers/customer.go`

    ```go
    type CustomerHandler struct {
     dbCtx sqldb.DBContext
    }
    
    func NewCustomerHandler(db sqldb.DBContext) *CustomerHandler {
     return &CustomerHandler{dbCtx: db}
    }
    ```

- แก้ไขไฟล์ `application/http.go` เพื่อส่ง db ไปให้ handler

    ```go
    type HTTPServer interface {
     Start()
     Shutdown() error
     RegisterRoutes(db sqldb.DBContext)
    }
    
    // ...
    
    func (s *httpServer) RegisterRoutes(db sqldb.DBContext) {
     v1 := s.app.Group("/api/v1")
    
     customers := v1.Group("/customers")
     {
      hdlr := handlers.NewCustomerHandler(db)
      customers.Post("", hdlr.CreateCustomer)
     }
    
     // orders
    }
    ```

- แก้ไขไฟล์ `application/application.go` เพื่อส่ง db เข้าไป

    ```go
    func (app *Application) RegisterRoutes() {
     app.httpServer.RegisterRoutes(app.db)
    }
    ```

### บันทึก customer ลงฐานข้อมูล

- แก้ไขไฟล์ `handlers/customer.go` เพื่อบันทึกข้อมูลลงตาราง customers

    ```go
    func (h *CustomerHandler) CreateCustomer(c fiber.Ctx) error {
      // รับ request body มาเป็น DTO
     type CreateCustomerRequest struct {
      Email  string `json:"email"`
      Credit int    `json:"credit"`
     }
     var req CreateCustomerRequest
     if err := c.Bind().Body(&req); err != nil {
      return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
     }
    
     logger.Log.Info("Received customer:", req)
    
     // ตรวจสอบความถูกต้อง (validate)
     if req.Email == "" {
      return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "email is required"})
     }
     if _, err := mail.ParseAddress(req.Email); err != nil{
      return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "email is invalid"})
     }
     if req.Credit <= 0 {
      return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "credit must be greater than 0"})
     }
     
     // ตรวจสอบว่ามี email นี้รึยัง
     var id int64
     sql := "SELECT id FROM public.customers where email = $1"
     ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
     defer cancel()
     
     if err := h.dbCtx.DB().QueryRowContext(ctx, sql, req.Email).Scan(&id); err != nil && err == sql.ErrNoRows{
      return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already exists"})
     }
    
     // บันทึกลงฐานข้อมูล
     sqlIns := "INSERT INTO customers (id, email, credit) VALUES ($1, $2, $3) RETURNING id"
     ctxIns, cancelIns := context.WithTimeout(c.Context(), 10*time.Second)
     defer cancelIns()
     
     if err := h.dbCtx.DB().QueryRowContext(ctxIns, sqlIns, idgen.GenerateTimeRandomID(), req.Email, req.Credit).Scan(&id); err != nil {
      return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
     }
    
     // ตอบกลับ client
     type CreateCustomerResponse struct {
      ID int `json:"id"`
     }
     resp := &CreateCustomerResponse{ID: id}
     return c.Status(fiber.StatusCreated).JSON(resp})
    }
    ```

## จัดวางโครงสร้างแบบ Layered Architecture

จากโค้ดการสร้าง customer จะเห็นได้ว่า โค้ดทุกอย่างจะรวมอยู่ที่เดียวกัน ทำให้ยากต่อการดูแลรักษา และทดสอบ ดังนั้นควรแยกความรับผิดชอบชัดเจน โดยใช้ Layered Architecture

เนื้อหาในส่วนนี้ประกอบด้วย

- ทำความรู้จัก Layered Architecture
- Repository Layer
- Service Layer
- Presentation Layer (HTTP Handlers)
- ประกอบร่าง

### ทำความรู้จัก Layered Architecture

**Layered Architecture** (หรือ **Multi-layer Architecture**) คือรูปแบบการออกแบบซอฟต์แวร์ที่แยกความรับผิดชอบของแต่ละส่วนออกเป็น “เลเยอร์” (ชั้น) อย่างชัดเจน โดยแต่ละเลเยอร์ทำหน้าที่เฉพาะของตัวเอง และเรียกใช้งานกันตามลำดับจากบนลงล่าง

**โครงสร้าง Layered Architecture โดยทั่วไป**

```
Client/UI Layer        ← ผู้ใช้โต้ตอบกับระบบ
↓
Presentation Layer     ← Controller, API (จัดการคำขอจากผู้ใช้)
↓ ← DTO
Service Layer          ← Business Logic (กฎทางธุรกิจ)
↓ ← Model
Repository/Data Layer  ← จัดการกับฐานข้อมูล, external APIs
↓
Database/External APIs
```

**เมื่อนำมา implement ในโค้ดของเรา**

```
project/
│
├── handler/          ← Presentation Layer (HTTP handlers)
├── dto/              ← รับ/ส่งข้อมูลระหว่าง handler ↔ service
├── service/          ← Business Logic (core logic)
├── model/            ← ใช้จัดการข้อมูลภายในระบบ service ↔ repository
├── repository/       ← Data Access (DB queries)
└── main.go           ← Entry point (setup DI, server, etc)
```

### Repository Layer

**Repository Layer** มีหน้าที่หลักในการ ติดต่อกับฐานข้อมูลหรือแหล่งเก็บข้อมูล โดยรับคำสั่งจาก Service Layer แล้วทำหน้าที่ CRUD (Create, Read, Update, Delete) โดยไม่ให้เลเยอร์อื่นรู้ว่าข้อมูลมาจากที่ใด (Postgres, MySQL, Redis หรือแม้แต่ API)

- เริ่มจากการสร้าง Model โดยให้ไฟล์ชื่อ `model/customer.go` เพื่อใช้แทนโครงสร้างข้อมูลที่ตรงกับฐานข้อมูล

    ```go
    package model
    
    import (
     "go-mma/util/idgen"
     "time"
    )
    
    type Customer struct {
     ID          int64     `db:"id"` // tag db ใช้สำหรับ StructScan() ของ sqlx
     Email       string    `db:"name"`
     Credit      int       `db:"credit"`
     CreatedAt   time.Time `db:"created_at"`
     UpdatedAt   time.Time `db:"updated_at"`
    }
    
    func NewCustomer(email string, credit int) *Customer {
     return &Customer{
      ID:     idgen.GenerateTimeRandomID(),
      Email:  email,
      Credit: creditLimit,
     }
    }
    ```

    <aside>
    💡

    ในบทความนี้จะสร้างเป็น [Rich Model](https://somprasongd.work/blog/architecture/anemic-vs-rich-model-ddd)

    </aside>

- สร้าง Repository สำหรับ บันทึกลูกค้าใหม่ลงฐานข้อมูล ไว้ที่ไฟล์ `repository/customer.go`

    ```go
    package repository
    
    import (
     "context"
     "fmt"
     "go-mma/util/storage/sqldb"
     "go-mma/model"
     "time"
    )
    
    type CustomerRepository struct {
     dbCtx sqldb.DBContext // dbCtx is an instance of DBContext interface for interacting with the database
    }
    
    func NewCustomerRepository(dbCtx sqldb.DBContext) *CustomerRepository {
     return &CustomerRepository{
      dbCtx: dbCtx, // Initialize the dbCtx field with the passed DBContext parameter
     }
    }
    
    func (r *CustomerRepository) Create(ctx context.Context, customer *model.Customer) error {
     query := `
    INSERT INTO public.customers (id, email, credit)
    VALUES ($1, $2, $3)
    RETURNING *
    `
    
     // Use context.WithTimeout to set a timeout for the database query execution
     ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
     defer cancel()
    
     err := r.dbCtx.DB().
      QueryRowxContext(ctx, query, customer.ID, customer.Email, customer.Credit).
      StructScan(customer)
     if err != nil {
      return fmt.Errorf("failed to create customer: %w", err) // Return an error if the query execution fails
     }
     return nil // Return nil if the operation is successful
    }
    
    func (r *customerRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
     query := `SELECT 1 FROM public.customers WHERE email = $1 LIMIT 1`
    
     ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
     defer cancel()
    
     var exists int
     err := r.dbCtx(ctx).
      QueryRowxContext(ctx, query, email).
      Scan(&exists)
     if err != nil {
      if err == sql.ErrNoRows {
       return false, nil
      }
      return false, errs.HandleDBError(fmt.Errorf("failed to select customer: %w", err))
     }
     return true, nil
    }
    ```

    <aside>
    💡

    การใช้ `context.WithTimeout` เป็นแนวปฏิบัติมาตรฐานสำหรับระบบงานที่เกี่ยวข้องกับฐานข้อมูลหรือ external service

    </aside>

### Service Layer

**Service Layer** คือเลเยอร์ที่อยู่ตรงกลางระหว่าง Controller (หรือ Handler) กับ Repository

หน้าที่หลักของ Service Layer คือ รวมและควบคุม Business Logic ของแอปพลิเคชันไว้ในที่เดียว ดังนี้

- **รับ DTO**: รับ DTO จาก Handler เข้ามาเพื่อประมวลผล
- **ตรวจสอบ**: ตรวจสอบความถูกต้องตาม business logic rule
- **แปลงข้อมูล**: แปลง DTO → Model
- **เรียก Repository**: เพื่อทำ CRUD (Create, Read, Update, Delete) ตามเงื่อนไข
- **ส่งผลลัพธ์**: รับผลลัพธ์จาก Repository แล้วแปลงกลับเป็น DTO Response
- **จัดการ error**: แสดง error log แล้วส่งกลับไปให้ Controller (หรือ Handler) จัดการต่อ

ขั้นตอนการสร้าง Service Layer

- สร้าง DTO (Data Transfer Object) ไว้เป็นตัวกลางสำหรับรับ–ส่งข้อมูล ระหว่างชั้น Handler ↔ Service

    สร้างไฟล์ `dto/customer_request.go`

    ```go
    package dto
    
    import "errors"
    
    type CreateCustomerRequest struct {
     Email  string `json:"email"`
     Credit int    `json:"credit"`
    }
    ```

    สร้างไฟล์ `dto/customer_response.go`

    ```go
    package dto
    
    type CreateCustomerResponse struct {
     ID int64 `json:"id"`
    }
    
    func NewCreateCustomerResponse(id int64) *CreateCustomerResponse {
     return &CreateCustomerResponse{ID: id}
    }
    ```

- สร้าง Service สำหรับควบคุม Business Logic ทั้งหมดในการสร้างลูกค้าใหม่

    สร้างไฟล์ `service/customer.go`

    ```go
    package service
    
    import (
     "context"
     "go-mma/dto"
     "go-mma/model"
     "go-mma/repository"
     "log"
    )
    
    var (
     ErrEmailExists = errors.New("email already exists")
    )
    
    type CustomerService struct {
     custRepo *repository.CustomerRepository
    }
    
    func NewCustomerService(custRepo *repository.CustomerRepository) *CustomerService {
     return &CustomerService{
      custRepo: custRepo,
     }
    }
    
    func (s *CustomerService) CreateCustomer(ctx context.Context, req *dto.CreateCustomerRequest) (*dto.CreateCustomerResponse, error) {
     // Business Logic Rule: ตรวจสอบ email ซ้ำ
     exists, err := h.custRepo.ExistsByEmail(ctx, cmd.Email)
     if err != nil {
      // error logging
      logger.Log.Error(err.Error())
      return nil, err
     }
    
     if exists {
      return nil, ErrEmailExists
     }
    
     // แปลง DTO → Model
     customer = model.NewCustomer(req.Email, req.Credit)
    
     // ส่งไปที่ Repository Layer เพื่อบันทึกข้อมูลลงฐานข้อมูล
     if err := s.custRepo.Create(ctx, customer); err != nil {
      // error logging
      logger.Log.Error(err.Error())
      return nil, err
     }
    
     // สร้าง DTO Response
     resp := dto.NewCreateCustomerResponse(customer.ID)
     return resp, nil
    }
    ```

### Presentation Layer (HTTP Handlers)

**Presentation Layer (HTTP Handlers)** คือชั้นที่อยู่บนสุดของระบบในสถาปัตยกรรมแบบ Layered Architecture โดยทำหน้าที่เป็น “จุดเชื่อมต่อระหว่างผู้ใช้ (Client) กับระบบ” ผ่านโปรโตคอล เช่น HTTP หรือ WebSocket

หน้าที่หลักของ Presentation Layer (หรือ HTTP Handler)

- รับคำขอ: รับ HTTP Request จาก Client
- แปลงข้อมูล: แปลง JSON → DTO (ใช้ `BodyParser`, `Bind`, หรือ Unmarshal)
- ตรวจสอบ: ตรวจสอบความถูกต้องของข้อมูล (validation)
- เรียก Service: ส่ง DTO เข้า Service Layer เพื่อประมวลผล
- ส่งผลลัพธ์: รับผลลัพธ์จาก Service แล้วแปลงกลับเป็น JSON Response
- จัดการ error: แปลง error จากชั้นล่างเป็น HTTP response code เช่น 400, 500

ขั้นตอนการสร้าง Presentation Layer (HTTP Handlers)

- แก้ไขไฟล์ `dto/customer_request.go` เพื่อเพิ่ม validation เช่น credit ต้อง ≥ 0 ก่อนส่งให้ Service

    ```go
    package dto
    
    import (
     "errors"
     "net/mail"
    )
    
    // struct
    
    func (r *CreateCustomerRequest) Validate() error {
     var errs error
     if r.Email == "" {
      errs = errors.Join(errs, errors.New("email is required"))
     }
     if _, err := mail.ParseAddress(r.Email); err != nil {
      errs = errors.Join(errs, errors.New("email is invalid"))
     }
     if r.Credit <= 0 {
      errs = errors.Join(errs, errors.New("credit must be greater than 0"))
     }
     return errs
    }
    ```

- แก้ไขไฟล์ `handler/customer.go` เพื่อให้ทำงานตามหน้าที่ของ Presentation Layer

    ```go
    package handlers
    
    import (
     "go-mma/dto"
     "go-mma/service"
    
     "github.com/gofiber/fiber/v3"
    )
    
    type CustomerHandler struct {
     custService *service.CustomerService
    }
    
    func NewCustomerHandler(custService *service.CustomerService) *CustomerHandler {
     return &CustomerHandler{
      custService: custService,
     }
    }
    
    func (h *CustomerHandler) CreateCustomer(c fiber.Ctx) error {
     // รับ request body มาเป็น DTO
     var req dto.CreateCustomerRequest
     if err := c.Bind().Body(&req); err != nil {
      return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
     }
    
     // ตรวจสอบความถูกต้อง (validate)
     if err := req.Validate(); err != nil {
      return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": strings.Join(strings.Split(err.Error(), "\n"), ", ")})
     }
    
     // ส่งไปที่ Service Layer
     resp, err := h.custService.CreateCustomer(c.Context(), &req)
    
     // จัดการ error จาก Service Layer หากเกิดขึ้น
     if err != nil {
      return c.Status(500).JSON(fiber.Map{"error": err.Error()})
     }
    
     // ตอบกลับ client
     return c.Status(fiber.StatusCreated).JSON(resp)
    }
    ```

### ประกอบร่าง

เมื่อเราทำครบทุกเลเยอร์แล้ว ถึงเวลาประกอบร่าง โดยจะใช้การทำ Dependency Injection แต่ละเลเยอร์เข้าไป ไปที่ไฟล์ `application/http.go` แล้วแก้ไขตามนี้

```go
func (s *httpServer) RegisterRoutes(db sqldb.DBContext) {
 v1 := s.app.Group("/api/v1")

 customers := v1.Group("/customers")
 {
  repo := repository.NewCustomerRepository(db)
  svc := service.NewCustomerService(repo)
  hdlr := handler.NewCustomerHandler(svc)
  customers.Post("", hdlr.CreateCustomer)
 }

 // orders
}
```

## การจัดการ Error

ตอนนี้ทุกๆ error ที่ส่งออกมาจาก Service Layer นั้น จะตอบกลับมายัง client ด้วย status code 500 ทั้งหมด ซึ่งยังไม่ถูกต้อง ดังนั้นในเนื้อหาส่วนนี้จะมาทำการจัดการ Error ให้ตอบกลับ status code ที่ถูกต้องกัน

- ออกแบบ status code ทั้งหมดของระบบ
- สร้าง Custome Error
- การจัดการ Error ใน Repository Layer
- การจัดการ Error ใน Service Layer
- การจัดการ Error ใน Presentation Layer
- สร้าง ErrorHandler Middleware

### ออกแบบ status code ทั้งหมดของระบบ

เริ่มเรามาดูก่อนว่าในระบบของเราจะมี error อะไรเกิดขึ้นได้บ้าง

| ประเภท | สถานะ | ใช้เมื่อ | หมายเหตุ |
| --- | --- | --- | --- |
| Input Validation | 400 Bad Request | ข้อมูลไม่ครบ, รูปแบบผิด | ตรวจจับได้ที่ Handler / DTO |
| Authorization | 401 Unauthorized | ยังไม่ login / token ผิด | ตรวจจับได้ที่ Middleware |
|  | 403 Forbidden | login แล้ว แต่ไม่มีสิทธิ์ | ตรวจจับได้ที่ Middleware |
| Business Rule | 404 Not Found | ไม่พบข้อมูล | ตรวจจับได้ที่ Service |
|  | 409 Conflict | ข้อมูลซ้ำกัน, ขัดแย้ง เช่น email ซ้ำ, order ถูก cancel ไปแล้ว | ตรวจจับได้ที่ Service |
|  | 422 Unprocessable Entity |  ข้อมูลมีรูปแบบถูก แต่ logic ผิด เช่น เครดิตไม่พอ, วันที่ย้อนหลัง | ตรวจจับได้ที่ Service |
| Database | 500 Internal Server Error | เกิด database connection error | ตรวจจับได้ที่ Repository |
| Exception | 500 Internal Server Error | เกิด exception หรือ panic ใน server code | เกิดได้ทุกที่ |

### สร้าง Custome Error

เมื่อได้ error ทั้งหมดที่จะเกิดขึ้นได้แล้วนั้น ก็มาสร้าง custome error เพื่อใช้จัดการ error ทั้งหมดที่จะเกิดขึ้นในระบบ

- สร้างไฟล์ `util/errs/types.go` ไว้สำหรับกำหนดประเภท error ทั้งหมดก่อน

    ```go
    package errs
    
    type ErrorType string
    
    const (
     ErrInputValidation   ErrorType = "input_validation_error"   // Invalid input (e.g., missing fields, format issues)
     ErrAuthentication    ErrorType = "authentication_error"     // Wrong credentials, not logged in
     ErrAuthorization     ErrorType = "authorization_error"      // No permission to access resource
     ErrResourceNotFound  ErrorType = "resource_not_found"       // Entity does not exist
     ErrConflict          ErrorType = "conflict"                 // Conflict, already exists
     ErrBusinessRule      ErrorType = "business_rule_error"      // Business rule violation
     ErrDataIntegrity     ErrorType = "data_integrity_error"     // Foreign key, constraint violations
     ErrDatabaseFailure   ErrorType = "database_failure"         // Generic DB error
     ErrOperationFailed   ErrorType = "operation_failed"         // General failure case
     ErrServiceDependency ErrorType = "service_dependency_error" // External service unavailable
    )
    ```

- สร้าง Custome Error ให้สร้างไฟล์ `util/errs/errs.go`

    ```go
    package errs
    
    import "fmt"
    
    type AppError struct {
     Type    ErrorType `json:"type"`    // สำหรับ client
     Message string    `json:"message"` // สำหรับ client
     Err     error     `json:"-"`       // สำหรับ log ภายใน
    }
    
    func (e *AppError) Error() string {
     if e.Err != nil {
      return fmt.Sprintf("[%s] %s - %v", e.Type, e.Message, e.Err)
     }
     return fmt.Sprintf("[%s] %s", e.Type, e.Message)
    }
    
    // Unwrap allows for errors.Is and errors.As compatibility
    func (e *AppError) Unwrap() error {
     return e.Err
    }
    
    func New(errorType ErrorType, message string, err ...error) *AppError {
     var underlyingErr error
     if len(err) > 0 {
      underlyingErr = err[0]
     }
     return &AppError{
      Type:    errorType,
      Message: message,
      Err:     underlyingErr,
     }
    }
    
    // Helper functions for each error type
    func InputValidationError(message string, err ...error) *AppError {
     return New(ErrInputValidation, message, err...)
    }
    
    func AuthenticationError(message string, err ...error) *AppError {
     return New(ErrAuthentication, message, err...)
    }
    
    func NewAuthorizationError(message string, err ...error) *AppError {
     return New(ErrAuthorization, message, err...)
    }
    
    func ResourceNotFoundError(message string, err ...error) *AppError {
     return New(ErrResourceNotFound, message, err...)
    }
    
    func ConflictError(message string, err ...error) *AppError {
     return New(ErrConflict, message, err...)
    }
    
    func BusinessRuleError(message string, err ...error) *AppError {
     return New(ErrBusinessRule, message, err...)
    }
    
    func DataIntegrityError(message string, err ...error) *AppError {
     return New(ErrDataIntegrity, message, err...)
    }
    
    func DatabaseFailureError(message string, err ...error) *AppError {
     return New(ErrDatabaseFailure, message, err...)
    }
    
    func OperationFailedError(message string, err ...error) *AppError {
     return New(ErrOperationFailed, message, err...)
    }
    
    func ServiceDependencyError(message string, err ...error) *AppError {
     return New(ErrServiceDependency, message, err...)
    }
    ```

### การจัดการ Error ใน Repository Layer

ในชั้นของ repository เมื่อเชื่อมต่อกับฐานข้อมูล PostgreSQL จะเกิด error ได้ ดังนี้

- 23502: Not null violation → **ErrConflict**
- 23503: Foreign key violation → **ErrDataIntegrity**
- 23505: Unique constraint violation → **ErrDataIntegrity**
- อื่นๆ → **ErrDatabaseFailure**

การ implement

- สร้างไฟล์ `util/errs/helpers.go` สำหรับเป็นตัวช่วย Map error code กับ error type

    ```go
    package errs
    
    import (
     "github.com/lib/pq"
    )
    
    // HandleDBError maps PostgreSQL errors to custom application errors
    func HandleDBError(err error) error {
     if pgErr, ok := err.(*pq.Error); ok {
      switch pgErr.Code {
      case "23505": // Unique constraint violation
       return New(ErrConflict, "duplicate entry detected: "+pgErr.Message)
      case "23503": // Foreign key violation
       return New(ErrDataIntegrity, "foreign key constraint violation: "+pgErr.Message)
      case "23502": // Not null violation
       return New(ErrDataIntegrity, "not null constraint violation: "+pgErr.Message)
      default:
       return New(ErrDatabaseFailure, "database error: "+pgErr.Message)
      }
     }
     // Fallback for unknown DB errors
     return New(ErrDatabaseFailure, err.Error())
    }
    ```

- แก้ไฟล์ `repository/customer.go` เพื่อมาเรียกใช้งาน `HandleDBError`

    ```go
    func (r *CustomerRepository) Create(ctx context.Context, customer *model.Customer) error {
     // ...
     if err != nil {
      return errs.HandleDBError(fmt.Errorf("failed to create customer: %w", err))
     }
     return nil // Return nil if the operation is successful
    }
    
    func (r *CustomerRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
     // ...
     if err != nil {
      if err == sql.ErrNoRows {
       return false, nil
      }
      return false, errs.HandleDBError(fmt.Errorf("failed to select customer: %w", err))
     }
     return true, nil
    }
    ```

### การจัดการ Error ใน Service Layer

ใน service layer ถ้าเป็น error ที่ได้มาจาก repository layer เราจะคืนกลับ error นั้นๆ ได้เลย เพราะถูกจัดการมาแล้ว ดังนั้น แค่เปลี่ยน error อื่นๆ มาเป็น AppError แทน

ในไฟล์ `service/customer.go` มีแค่ error การตรวจสอบ email ซ้ำเท่านั้น

```go
var (
 ErrEmailExists = errs.ConflictError("email already exists")
)
```

ใน handler จะมี error ดังนี้

- การแปลง JSON → DTO: ต้องเปลี่ยนมาใช้ AppError
- การตรวจสอบ DTO: ต้องเปลี่ยนมาใช้ AppError
- Error ที่ได้รับมาจาก Service Layer: สามารถใช้ได้เลย

ขั้นตอนการ implement

- แก้ไขไฟล์ `handler/customer.go` เพื่อเปลี่ยนมาใช้ AppError

    ```go
    func (h *CustomerHandler) CreateCustomer(c fiber.Ctx) error {
     // 1. รับ request body มาเป็น DTO
     var req dto.CreateCustomerRequest
     if err := c.Bind().Body(&req); err != nil {
      errResp := errs.InputValidationError(err.Error())
      return c.Status(fiber.StatusBadRequest).JSON(errResp)
     }
    
     // 2. ตรวจสอบความถูกต้อง (validate)
     if err := req.Validate(); err != nil {
      errResp := errs.InputValidationError(strings.Join(strings.Split(err.Error(), "\n"), ", "))
      return c.Status(fiber.StatusBadRequest).JSON(errResp)
     }
    
     // 3. ส่งไปที่ Service Layer
     resp, err := h.custService.CreateCustomer(c.Context(), &req)
    
     // 4. จัดการ error จาก Service Layer หากเกิดขึ้น
     if err != nil {
      return c.Status(fiber.StatusInternalServerError).JSON(err)
     }
    
     // 5. ตอบกลับ client
     return c.Status(fiber.StatusCreated).JSON(resp)
    }
    ```

- แต่จะเห็นว่า status code ยังไม่ถูกต้อง ซึ่งจะต้องดึงมาจาก AppError ดังนั้น ให้สร้าง helper function สำหรับถอด status code มา โดยแก้ไขไฟล์ `util/errs/helpers.go` ให้เพิ่ม ตามนี้

    ```go
    // GetErrorType extracts the error type from an errorAdd commentMore actions
    func GetErrorType(err error) ErrorType {
     var appErr *AppError
     if errors.As(err, &appErr) {
      return appErr.Type
     }
     return ErrOperationFailed // Default error type if not recognized
    }
    
    // Map error type to HTTP status code
    func GetHTTPStatus(err error) int {
     switch GetErrorType(err) {
     case ErrInputValidation:
      return fiber.StatusBadRequest // 400
     case ErrAuthentication:
      return fiber.StatusUnauthorized // 401
     case ErrAuthorization:
      return fiber.StatusForbidden // 403
     case ErrResourceNotFound:
      return fiber.StatusNotFound // 404
     case ErrConflict:
      return fiber.StatusConflict // 409
     case ErrBusinessRule:
      return fiber.StatusBadRequest // 422
     case ErrDataIntegrity, ErrDatabaseFailure:
      return fiber.StatusInternalServerError // 500
     case ErrOperationFailed:
      return fiber.StatusInternalServerError // 500
     case ErrServiceDependency:
      return fiber.StatusServiceUnavailable // 503
     default: // Default: Unknown errors, fallback to internal server error
      return fiber.StatusInternalServerError // 500
     }
    }
    ```

- แก้ไขไฟล์ `handler/customer.go` เพื่อใช้ status code ที่ถูกต้อง

    ```go
    func (h *CustomerHandler) CreateCustomer(c fiber.Ctx) error {
     // 1. รับ request body มาเป็น DTO
     var req dto.CreateCustomerRequest
     if err := c.Bind().Body(&req); err != nil {
      errResp := errs.InputValidationError(err.Error())
      return c.Status(errs.GetHTTPStatus(errResp)).JSON(errResp)
     }
    
     // 2. ตรวจสอบความถูกต้อง (validate)
     if err := req.Validate(); err != nil {
      errResp := errs.InputValidationError(strings.Join(strings.Split(err.Error(), "\n"), ", "))
      return c.Status(errs.GetHTTPStatus(errResp)).JSON(errResp)
     }
    
     // 3. ส่งไปที่ Service Layer
     resp, err := h.custService.CreateCustomer(c.Context(), &req)
    
     // 4. จัดการ error จาก Service Layer หากเกิดขึ้น
     if err != nil {
      return c.Status(errs.GetHTTPStatus(err)).JSON(err)
     }
    
     // 5. ตอบกลับ client
     return c.Status(fiber.StatusCreated).JSON(resp)
    }
    ```

- สร้าง Standard Error Response เพื่อมาจัดการส่ง error response ให้สร้างไฟล์ `util/response/response.go`

    ```go
    package response
    
    import (
     "errors"
     "go-mma/util/errs"
    
     "github.com/gofiber/fiber/v3"
    )
    
    func JSONError(c fiber.Ctx, err error) error {
     // Convert non-AppError to AppError with type ErrOperationFailed
     appErr, ok := err.(*errs.AppError)
     if !ok {
      appErr = errs.New(
       errs.ErrOperationFailed,
       err.Error(),
       err,
      )
     }
    
     // Get the appropriate HTTP status code
     statusCode := errs.GetHTTPStatus(err)
     
     // Retrieve the custom status code if it's a *fiber.Error
     var e *fiber.Error
     if errors.As(err, &e) {
      statusCode = e.Code
     }
    
     // Return structured response with error type and message
     return c.Status(statusCode).JSON(appErr)
    }
    ```

- แก้ไขไฟล์ `handler/customer.go` เพื่อใช้เรียกใช้ `JSONError`

    ```go
    func (h *CustomerHandler) CreateCustomer(c fiber.Ctx) error {
     // 1. รับ request body มาเป็น DTO
     var req dto.CreateCustomerRequest
     if err := c.Bind().Body(&req); err != nil {
      return response.JSONError(c, errs.InputValidationError(err.Error()))
     }
    
     // 2. ตรวจสอบความถูกต้อง (validate)
     if err := req.Validate(); err != nil {
      return response.JSONError(c, errs.InputValidationError(strings.Join(strings.Split(err.Error(), "\n"), ", ")))
     }
    
     // 3. ส่งไปที่ Service Layer
     resp, err := h.custService.CreateCustomer(c.Context(), &req)
    
     // 4. จัดการ error จาก Service Layer หากเกิดขึ้น
     if err != nil {
      return response.JSONError(c, err)
     }
    
     // 5. ตอบกลับ client
     return c.Status(fiber.StatusCreated).JSON(resp)
    }
    ```

### สร้าง Middleware สำหรับจัดการตอบกลับ Error

อีกวิธีการหนึ่งในการตอบกลับ error แทนที่จะเรียก `response.JSONError` ในทุกๆ ที่ ที่เกิด error ขึ้นใน handler คือ ให้ `return error` กลับออกไปเลย แล้วสร้าง middleware ใหม่ ขึ้นมาจัดการแทน ดังนี้

- สร้างไฟล์ `application/middleware/response_error.go`

    ```go
    package middleware
    
    import (
     "errors"
     "go-mma/util/errs"
    
     "github.com/gofiber/fiber/v3"
    )
    
    func ResponseError() fiber.Handler {
     return func(c fiber.Ctx) error {
      err := c.Next()
      if err == nil {
       return nil
      }
    
      return jsonError(c, err)
     }
    }
    
    // ย้ายจาก util/response มาไว้ที่นี่แทน เพราะใช้งานเฉพาะในนี้
    func jsonError(c fiber.Ctx, err error) error {
     // Convert non-AppError to AppError with type ErrOperationFailed
     appErr, ok := err.(*errs.AppError)
     if !ok {
      appErr = errs.New(
       errs.ErrOperationFailed,
       err.Error(),
       err,
      )
     }
    
     // Get the appropriate HTTP status code
     statusCode := errs.GetHTTPStatus(err)
    
     // Retrieve the custom status code if it's a *fiber.Error
     var e *fiber.Error
     if errors.As(err, &e) {
      statusCode = e.Code
     }
    
     // Return structured response with error type and message
     return c.Status(statusCode).JSON(appErr)
    }
    ```

- แก้ไขไฟล์ `handler/customer.go` ให้ `return errror` กลับออกไป

    ```go
    func (h *CustomerHandler) CreateCustomer(c fiber.Ctx) error {
     // 1. รับ request body มาเป็น DTO
     var req dto.CreateCustomerRequest
     if err := c.Bind().Body(&req); err != nil {
      return errs.InputValidationError(err.Error()) // <-- ตรงนี้
     }
    
     // 2. ตรวจสอบความถูกต้อง (validate)
     if err := req.Validate(); err != nil {
      return errs.InputValidationError(strings.Join(strings.Split(err.Error(), "\n"), ", ")) // <-- ตรงนี้
     }
    
     // 3. ส่งไปที่ Service Layer
     resp, err := h.custService.CreateCustomer(c.Context(), &req)
    
     // 4. จัดการ error จาก Service Layer หากเกิดขึ้น
     if err != nil {
      return err // <-- ตรงนี้
     }
    
     // 5. ตอบกลับ client
     return c.Status(fiber.StatusCreated).JSON(resp)
    }
    ```

## ระบบส่งอีเมล

จากโจทย์ที่ตั้งไว้ในส่วนของการสร้างลูกค้าใหม่ยังขาดในเรื่อง ของการส่งอีเมลต้อนรับ ซึ่งการส่งอีเมลนั้น ยังใช้ในส่วนของการส่งยืนยันการสั่งออเดอร์ด้วย เราจึงควรสร้างเป็น service แยกออกมาเพื่อใช้งานร่วมกัน ซึ่งมีขั้นตอน ดังนี้

- สร้างไฟล์ `service/notification.go`

    ```go
    package service
    
    import (
     "fmt"
     "go-mma/util/logger"
    )
    
    type NotificationService struct {
    }
    
    func NewNotificationService() *NotificationService {
     return &NotificationService{}
    }
    
    func (s *NotificationService) SendEmail(to string, subject string, payload map[string]any) error {
     // implement email sending logic here
     logger.Log.Info(fmt.Sprintf("Sending email to %s with subject: %s and payload: %v", to, subject, payload))
     return nil
    }
    
    ```

- แก้ไขไฟล์ `service/customer.go` เพื่อรับ notification service มาใช้งาน

    ```go
    package service
    
    // ...
    
    type CustomerService struct {
     custRepo *repository.CustomerRepository
     notiSvc  *NotificationService // <-- เพิ่มตรงนี้
    }
    
    func NewCustomerService(custRepo *repository.CustomerRepository, 
     notiSvc *NotificationService, // <-- เพิ่มตรงนี้
     ) *CustomerService {
     return &CustomerService{
      custRepo: custRepo,
      notiSvc:  notiSvc, // <-- เพิ่มตรงนี้
     }
    }
    
    func (s *CustomerService) CreateCustomer(ctx context.Context, req *dto.CreateCustomerRequest) (*dto.CreateCustomerResponse, error) {
     // Business Logic Rule: ตรวจสอบ email ซ้ำ
     // แปลง DTO → Model
     // ส่งไปที่ Repository Layer เพื่อบันทึกข้อมูลลงฐานข้อมูล
    
     // ส่งอีเมลต้อนรับ // <-- เพิ่มตรงนี้
     if err := s.notiSvc.SendEmail(customer.Email, "Welcome to our service!", map[string]any{
      "message": "Thank you for joining us! We are excited to have you as a member.",
     }); err != nil {
      // error logging
      logger.Log.Error(err.Error())
      return nil, err
     }
    
     // สร้าง DTO Response
    }
    ```

- แก้ไขไฟล์ `application/http.go` เพื่อทำ depency injection

    ```go
    func (s *httpServer) RegisterRoutes(db sqldb.DBContext) {
     v1 := s.app.Group("/api/v1")
    
     customers := v1.Group("/customers")
     {
      repo := repository.NewCustomerRepository(db)
      svcNoti := service.NewNotificationService()      // <-- เพิ่มตรงนี้
      svc := service.NewCustomerService(repo, svcNoti) // <-- เพิ่มตรงนี้
      hdlr := handler.NewCustomerHandler(svc)
      customers.Post("", hdlr.CreateCustomer)
     }
     // orders
    }
    ```

## ระบบจัดการออเดอร์

เนื้อหาส่วนนี้จะทำการสร้างโค้ดสำหรับจัดการออเดอร์ โดยใช้ Layered Architecture มีขั้นตอน ดังนี้

### Repository Layer

- สร้างโมเดล:

    สร้างไฟล์ `model/order.go`

    ```go
    package model
    
    import (
     "go-mma/util/idgen"
     "time"
    )
    
    type Order struct {
     ID         int64      `db:"id"`
     CustomerID int64      `db:"customer_id"`
     OrderTotal int        `db:"order_total"`
     CreatedAt  time.Time  `db:"created_at"`
     CanceledAt *time.Time `db:"canceled_at"` // nullable
    }
    
    func NewOrder(customerID int64, orderTotal int) *Order {
     return &Order{
      ID:     idgen.GenerateTimeRandomID(),
      CustomerID: customerID,
      OrderTotal: orderTotal,
     }
    }
    ```

    แก้ไขไฟล์ `model/customer.go` เพื่อเพิ่มฟังก์ชัน การตัดยอด credit และคืนยอด credit ([Rich Model](https://somprasongd.work/blog/architecture/anemic-vs-rich-model))

    ```go
    func (c *Customer) ReserveCredit(v int) error {
     newCredit := c.Credit - v
     if newCredit < 0 {
      return errs.BusinessRuleError("insufficient credit limit")
     }
     c.Credit = newCredit
     return nil
    }
    
    func (c *Customer) ReleaseCredit(v int) {
     if c.Credit <= 0 {
      c.Credit = 0
     }
     c.Credit = c.Credit + v
     return
    }
    ```

- สร้าง Repository:

    สร้างไฟล์ `repository/order.go` สำหรับสร้างออเดอร์ใหม่, ค้นหาจาก id และยกเลิกออเดอร์

    ```go
    package repository
    
    import (
     "context"
     "database/sql"
     "fmt"
     "go-mma/util/storage/sqldb"
     "go-mma/model"
     "go-mma/util/errs"
     "time"
    )
    
    type OrderRepository struct {
     dbCtx sqldb.DBContext
    }
    
    func NewOrderRepository(dbCtx sqldb.DBContext) *OrderRepository {
     return &OrderRepository{
      dbCtx: dbCtx,
     }
    }
    
    func (r *OrderRepository) Create(ctx context.Context, m *model.Order) error {
     query := `
     INSERT INTO public.orders (
       id, customer_id, order_total
     )
     VALUES ($1, $2, $3)
     RETURNING *
     `
    
     ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
     defer cancel()
    
     err := r.dbCtx.DB().QueryRowxContext(ctx, query, m.ID, m.CustomerID, m.OrderTotal).StructScan(m)
     if err != nil {
      return errs.HandleDBError(fmt.Errorf("failed to create order: %w", err))
     }
     return nil
    }
    
    func (r *OrderRepository) FindByID(ctx context.Context, id int64) (*model.Order, error) {
     query := `
     SELECT *
     FROM public.orders
     WHERE id = $1
     AND canceled_at IS NULL
    `
     ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
     defer cancel()
    
     var order model.Order
     err := r.dbCtx.DB().QueryRowxContext(ctx, query, id).StructScan(&order)
     if err != nil {
      if err == sql.ErrNoRows {
       return nil, nil
      }
      return nil, errs.HandleDBError(fmt.Errorf("failed to get order by ID: %w", err))
     }
     return &order, nil
    }
    
    func (r *OrderRepository) Cancel(ctx context.Context, id int64) error {
     query := `
     UPDATE public.orders
     SET canceled_at = current_timestamp
     WHERE id = $1
    `
     ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
     defer cancel()
     _, err := r.dbCtx.DB().ExecContext(ctx, query, id)
     if err != nil {
      return errs.HandleDBError(fmt.Errorf("failed to cancel order: %w", err))
     }
     return nil
    }
    ```

    แก้ไขไฟล์ `repository/customer.go` เพิ่มฟังก์ชันสำหรับค้นหาจาก id และอัพเดท credit

    ```go
    func (r *CustomerRepository) FindByID(ctx context.Context, id int64) (*model.Customer, error) {
     query := `
     SELECT *
     FROM public.customers
     WHERE id = $1
    `
     ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
     defer cancel()
    
     var customer model.Customer
     err := r.dbCtx.DB().QueryRowxContext(ctx, query, id).StructScan(&customer)
     if err != nil {
      if err == sql.ErrNoRows {
       return nil, nil
      }
      return nil, errs.HandleDBError(fmt.Errorf("failed to get customer by ID: %w", err))
     }
    
     return &customer, nil
    }
    
    func (r *CustomerRepository) UpdateCredit(ctx context.Context, m *model.Customer) error {
     query := `
     UPDATE public.customers
     SET credit = $2
     WHERE id = $1
     RETURNING *
    `
     ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
     defer cancel()
    
     err := r.dbCtx.DB().QueryRowxContext(ctx, query, m.ID, m.Credit).StructScan(m)
     if err != nil {
      return errs.HandleDBError(fmt.Errorf("failed to update customer credit: %w", err))
     }
     return nil
    }
    ```

### Service Layer

- สร้าง DTO สำหรับไว้ รับ-ส่งข้อมูลระหว่าง Handler ↔ Service

    สร้างไฟล์ `dto/order_request.go`

    ```go
    package dto
    
    import "fmt"
    
    type CreateOrderRequest struct {
     CustomerID int64 `json:"customer_id"`
     OrderTotal int   `json:"order_total"`
    }
    
    func (r *CreateOrderRequest) Validate() error {
     if r.CustomerID <= 0 {
      return fmt.Errorf("customer_id is required")
     }
     if r.OrderTotal <= 0 {
      return fmt.Errorf("order_total must be greater than 0")
     }
     return nil
    }
    ```

    สร้างไฟล์ `dto/order_response.go`

    ```go
    package dto
    
    type CreateOrderResponse struct {
     ID int64 `json:"id"`
    }
    
    func NewCreateOrderResponse(id int64) *CreateOrderResponse {
     return &CreateOrderResponse{ID: id}
    }
    ```

- สร้าง Service: สร้างไฟล์ `service/order.go`

    ```go
    package service
    
    import (
     "context"
     "go-mma/dto"
     "go-mma/model"
     "go-mma/repository"
     "go-mma/util/errs"
     "go-mma/util/logger"
    )
    
    var (
     ErrNoCustomerID = errs.ResourceNotFoundError("the customer with given id was not found")
     ErrNoOrderID    = errs.ResourceNotFoundError("the order with given id was not found")
    )
    
    type OrderService struct {
     custRepo  *repository.CustomerRepository
     orderRepo *repository.OrderRepository
     notiSvc   *NotificationService
    }
    
    func NewOrderService(custRepo *repository.CustomerRepository, orderRepo *repository.OrderRepository, notiSvc *NotificationService) *OrderService {
     return &OrderService{
      custRepo:  custRepo,
      orderRepo: orderRepo,
      notiSvc:   notiSvc,
     }
    }
    
    func (s *OrderService) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) (*dto.CreateOrderResponse, error) {
     // Business Logic Rule: ตรวจสอบ customer id
     customer, err := s.custRepo.FindByID(ctx, req.CustomerID)
     if err != nil {
      logger.Log.Error(err.Error())
      return 0, err
     }
    
     if customer == nil {
      return 0, ErrNoCustomerID
     }
    
     // Business Logic Rule: ตัดยอด credit ถ้าไม่พอให้ error
     if err := customer.ReserveCredit(req.OrderTotal); err != nil {
      return 0, err
     }
    
     // ตัดยอด credit ในตาราง customer
     if err := s.custRepo.UpdateCredit(ctx, customer); err != nil {
      logger.Log.Error(err.Error())
      return 0, err
     }
    
     // สร้าง order ใหม่
     order := model.NewOrder(req.CustomerID, req.OrderTotal)
     err = s.orderRepo.Create(ctx, order)
     if err != nil {
      logger.Log.Error(err.Error())
      return 0, err
     }
    
     err = s.notiSvc.SendEmail(customer.Email, "Order Created", map[string]any{
      "order_id": order.ID,
      "total":    order.OrderTotal,
     })
     if err != nil {
      logger.Log.Error(err.Error())
      return 0, err
     }
    
     // สร้าง DTO Response
     resp := dto.NewCreateOrderResponse(order.ID)
     return resp, nil
    }
    
    func (s *OrderService) CancelOrder(ctx context.Context, id int64) error {
     // Business Logic Rule: ตรวจสอบ order id
     order, err := s.orderRepo.FindByID(ctx, id)
     if err != nil {
      logger.Log.Error(err.Error())
      return err
     }
    
     if order == nil {
      return ErrNoOrderID
     }
    
     err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
    
      // ยกเลิก order
      if err := s.orderRepo.Cancel(ctx, order.ID); err != nil {
       logger.Log.Error(err.Error())
       return err
      }
     
      // Business Logic Rule: ตรวจสอบ customer id
      customer, err := s.custRepo.FindByID(ctx, order.CustomerID)
      if err != nil {
       logger.Log.Error(err.Error())
       return err
      }
     
      if customer == nil {
       return ErrNoCustomerID
      }
     
      // Business Logic: คืนยอด credit
      customer.ReleaseCredit(order.OrderTotal)
     
      // บันทึกการคืนยอด credit
      if err := s.custRepo.UpdateCredit(ctx, customer); err != nil {
       logger.Log.Error(err.Error())
       return err
      }
     
      return nil
     })
     
    
     return err
    }
    ```

### Presentation Layer

- สร้าง Handler: สร้างไฟล์ `handler/order.go`

    ```go
    package handler
    
    import (
     "go-mma/dto"
     "go-mma/service"
     "go-mma/util/errs"
     "strconv"
     "strings"
    
     "github.com/gofiber/fiber/v3"
    )
    
    type OrderHandler struct {
     orderSvc *service.OrderService
    }
    
    func NewOrderHandler(orderSvc *service.OrderService) *OrderHandler {
     return &OrderHandler{orderSvc: orderSvc}
    }
    
    func (h *OrderHandler) CreateOrder(c fiber.Ctx) error {
     // 1. รับ request body มาเป็น DTO
     var req dto.CreateOrderRequest
     if err := c.Bind().Body(&req); err != nil {
      return errs.InputValidationError(err.Error())
     }
    
     // 2. ตรวจสอบความถูกต้อง (validate)
     if err := req.Validate(); err != nil {
      return errs.InputValidationError(strings.Join(strings.Split(err.Error(), "\n"), ", "))
     }
    
     // 3. ส่งไปที่ Service Layer
     resp, err := h.orderSvc.CreateOrder(c.Context(), &req)
    
     // 4. จัดการ error จาก Service Layer หากเกิดขึ้น
     if err != nil {
      return err
     }
    
     // 5. ตอบกลับ client
     return c.Status(fiber.StatusCreated).JSON(resp)
    }
    
    func (h *OrderHandler) CancelOrder(c fiber.Ctx) error {
     // 1. อ่านค่า id จาก path param
     id := c.Params("orderID")
    
     // 2. ตรวจสอบรูปแบบ order id
     orderID, err := strconv.Atoi(id)
     if err != nil {
      return errs.InputValidationError("invalid order id")
     }
    
     // 3. ส่งไปที่ Service Layer
     err = h.orderSvc.CancelOrder(c.Context(), int64(orderID))
    
     // 4. จัดการ error จาก Service Layer หากเกิดขึ้น
     if err != nil {
      return err
     }
    
     // 5. ตอบกลับ client
     return c.SendStatus(fiber.StatusNoContent)
    }
    ```

### ประกอบร่างด้วย Dependency Injection

- แก้ไขไฟล์ `application/http.go`

    ```go
    func (s *httpServer) RegisterRoutes(db sqldb.DBContext) {
     v1 := s.app.Group("/api/v1")
    
     // customers
    
     orders := v1.Group("/orders")
     {
      repoCust := repository.NewCustomerRepository(db)
      repoOrder := repository.NewOrderRepository(db)
      svcNoti := service.NewNotificationService()
      svcCust := service.NewOrderService(repoCust, repoOrder, svcNoti)
      hdlr := handler.NewOrderHandler(svcCust)
      orders.Post("", hdlr.CreateOrder)
      orders.Delete("/:orderID", hdlr.CancelOrder)
     }
    }
    ```

## Database Transaction

จากโค้ดล่าสุด ตัวอย่างการสร้างออเดอร์ใหม่ จะมีการเรียกใช้ repository หลายครั้ง เช่น

1. หักเครดิตจากลูกค้า
2. บันทึกคำสั่งซื้อ (order)

หากคำสั่งแรกสำเร็จ แต่คำสั่งที่สองล้มเหลว จะทำให้ข้อมูลจะไม่สมบูรณ์

การใช้ Transaction ช่วยให้คุณสามารถ `ROLLBACK` กลับทั้งหมดได้ทันที หากเกิด error ในคำสั่งใดคำสั่งหนึ่ง

### Transactor

เริ่มจากสร้างตัวช่วยสำหรับจัดการเรื่องควบคุม transaction

<aside>
💡

โค้ดส่วนนี้จะถูกดัดแปลงมาจาก <https://github.com/Thiht/transactor>

</aside>

- สร้าง custom type ทั้งหมดที่ต้องใช้งาน

    สร้างไฟล์ `util/storage/sqldb/transactor/types.go`

    ```go
    package transactor
    
    import (
     "context"
     "database/sql"
    
     "github.com/jmoiron/sqlx"
    )
    
    // DBTX is the common interface between *[sqlx.DB] and *[sqlx.Tx].
    type DBTX interface {
     // database/sql methods
    
     ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
     PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
     QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
     QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
    
     Exec(query string, args ...any) (sql.Result, error)
     Prepare(query string) (*sql.Stmt, error)
     Query(query string, args ...any) (*sql.Rows, error)
     QueryRow(query string, args ...any) *sql.Row
    
     // sqlx methods
    
     GetContext(ctx context.Context, dest any, query string, args ...any) error
     MustExecContext(ctx context.Context, query string, args ...any) sql.Result
     NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
     PrepareNamedContext(ctx context.Context, query string) (*sqlx.NamedStmt, error)
     PreparexContext(ctx context.Context, query string) (*sqlx.Stmt, error)
     QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row
     QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error)
     SelectContext(ctx context.Context, dest any, query string, args ...any) error
    
     Get(dest any, query string, args ...any) error
     MustExec(query string, args ...any) sql.Result
     NamedExec(query string, arg any) (sql.Result, error)
     NamedQuery(query string, arg any) (*sqlx.Rows, error)
     PrepareNamed(query string) (*sqlx.NamedStmt, error)
     Preparex(query string) (*sqlx.Stmt, error)
     QueryRowx(query string, args ...any) *sqlx.Row
     Queryx(query string, args ...any) (*sqlx.Rows, error)
     Select(dest any, query string, args ...any) error
    
     Rebind(query string) string
     BindNamed(query string, arg any) (string, []any, error)
     DriverName() string
    }
    
    type sqlxDB interface {
     DBTX
     BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
    }
    
    type sqlxTx interface {
     Commit() error
     Rollback() error
    }
    
    var (
     _ DBTX   = &sqlx.DB{}
     _ DBTX   = &sqlx.Tx{}
     _ sqlxDB = &sqlx.DB{}
     _ sqlxTx = &sqlx.Tx{}
    )
    
    type (
     transactorKey struct{}
     // DBContext is used to get the current DB handler from the context.
     // It returns the current transaction if there is one, otherwise it will return the original DB.
     DBContext func(context.Context) DBTX
    )
    
    func txToContext(ctx context.Context, tx sqlxDB) context.Context {
     return context.WithValue(ctx, transactorKey{}, tx)
    }
    
    func txFromContext(ctx context.Context) sqlxDB {
     if tx, ok := ctx.Value(transactorKey{}).(sqlxDB); ok {
      return tx
     }
     return nil
    }
    ```

- สร้างฟังก์ชันสำหรับจัดการเรื่อง nested transactions

    สร้างไฟล์ `util/storage/sqldb/transactor/nested_transactions_none.go` สำหรับไม่รองรับ nested transactions

    ```go
    package transactor
    
    import (
     "context"
     "database/sql"
     "errors"
    
     "github.com/jmoiron/sqlx"
    )
    
    // NestedTransactionsNone is an implementation that prevents using nested transactions.
    func NestedTransactionsNone(db sqlxDB, tx *sqlx.Tx) (sqlxDB, sqlxTx) {
     switch typedDB := db.(type) {
     case *sqlx.DB:
      return &nestedTransactionNone{tx}, tx
    
     case *nestedTransactionNone:
      return typedDB, typedDB
    
     default:
      panic("unsupported type")
     }
    }
    
    type nestedTransactionNone struct {
     *sqlx.Tx
    }
    
    func (t *nestedTransactionNone) BeginTxx(_ context.Context, _ *sql.TxOptions) (*sqlx.Tx, error) {
     return nil, errors.New("nested transactions are not supported")
    }
    
    func (t *nestedTransactionNone) Commit() error {
     return errors.New("nested transactions are not supported")
    }
    
    func (t *nestedTransactionNone) Rollback() error {
     return errors.New("nested transactions are not supported")
    }
    
    ```

    สร้างไฟล์ `util/storage/sqldb/transactor/nested_transactions_savepoint.go` สำหรับรองรับ nested transactions แบบ savepoint

    ```go
    package transactor
    
    import (
     "context"
     "database/sql"
     "fmt"
     "strconv"
     "sync/atomic"
    
     "github.com/jmoiron/sqlx"
    )
    
    // NestedTransactionsSavepoints is a nested transactions implementation using savepoints.
    // It's compatible with PostgreSQL, MySQL, MariaDB, and SQLite.
    func NestedTransactionsSavepoints(db sqlxDB, tx *sqlx.Tx) (sqlxDB, sqlxTx) {
     switch typedDB := db.(type) {
     case *sqlx.DB:
      return &nestedTransactionSavepoints{Tx: tx}, tx
    
     case *nestedTransactionSavepoints:
      nestedTransaction := &nestedTransactionSavepoints{
       Tx:    tx,
       depth: typedDB.depth + 1,
      }
      return nestedTransaction, nestedTransaction
    
     default:
      panic("unsupported type")
     }
    }
    
    type nestedTransactionSavepoints struct {
     *sqlx.Tx
     depth int64
     done  atomic.Bool
    }
    
    func (t *nestedTransactionSavepoints) BeginTxx(ctx context.Context, _ *sql.TxOptions) (*sqlx.Tx, error) {
     if _, err := t.ExecContext(ctx, "SAVEPOINT sp_"+strconv.FormatInt(t.depth+1, 10)); err != nil {
      return nil, fmt.Errorf("failed to create savepoint: %w", err)
     }
    
     return t.Tx, nil
    }
    
    func (t *nestedTransactionSavepoints) Commit() error {
     if !t.done.CompareAndSwap(false, true) {
      return sql.ErrTxDone
     }
    
     if _, err := t.Exec("RELEASE SAVEPOINT sp_" + strconv.FormatInt(t.depth, 10)); err != nil {
      return fmt.Errorf("failed to release savepoint: %w", err)
     }
    
     return nil
    }
    
    func (t *nestedTransactionSavepoints) Rollback() error {
     if !t.done.CompareAndSwap(false, true) {
      return sql.ErrTxDone
     }
    
     if _, err := t.Exec("ROLLBACK TO SAVEPOINT sp_" + strconv.FormatInt(t.depth, 10)); err != nil {
      return fmt.Errorf("failed to rollback to savepoint: %w", err)
     }
    
     return nil
    }
    ```

- ตัวจัดการ transaction

    สร้างไฟล์ `util/storage/sqldb/transactor/transactor.go`

    ```go
    // Ref: https://github.com/Thiht/transactor/blob/main/sqlx/transactor.go
    package transactor
    
    import (
     "context"
     "fmt"
    
     "github.com/jmoiron/sqlx"
    )
    
    type Transactor interface {
     WithinTransaction(ctx context.Context, txFunc func(ctxWithTx context.Context) error) error
    }
    
    type (
     sqlxDBGetter               func(context.Context) sqlxDB
     nestedTransactionsStrategy func(sqlxDB, *sqlx.Tx) (sqlxDB, sqlxTx)
    )
    
    type sqlTransactor struct {
     sqlxDBGetter
     nestedTransactionsStrategy
    }
    
    type Option func(*sqlTransactor)
    
    func New(db *sqlx.DB, opts ...Option) (Transactor, DBContext) {
     t := &sqlTransactor{
      sqlxDBGetter: func(ctx context.Context) sqlxDB {
       if tx := txFromContext(ctx); tx != nil {
        return tx
       }
       return db
      },
      nestedTransactionsStrategy: NestedTransactionsNone, // Default strategy
     }
    
     for _, opt := range opts {
      opt(t)
     }
    
     dbGetter := func(ctx context.Context) DBTX {
      if tx := txFromContext(ctx); tx != nil {
       return tx
      }
    
      return db
     }
    
     return t, dbGetter
    }
    
    func WithNestedTransactionStrategy(strategy nestedTransactionsStrategy) Option {
     return func(t *sqlTransactor) {
      t.nestedTransactionsStrategy = strategy
     }
    }
    
    func (t *sqlTransactor) WithinTransaction(ctx context.Context, txFunc func(ctxWithTx context.Context) error) error {
     currentDB := t.sqlxDBGetter(ctx)
    
     tx, err := currentDB.BeginTxx(ctx, nil)
     if err != nil {
      return fmt.Errorf("failed to begin transaction: %w", err)
     }
    
     newDB, currentTX := t.nestedTransactionsStrategy(currentDB, tx)
     defer func() {
      _ = currentTX.Rollback() // If rollback fails, there's nothing to do, the transaction will expire by itself
     }()
     ctxWithTx := txToContext(ctx, newDB)
    
     if err := txFunc(ctxWithTx); err != nil {
      return err
     }
    
     if err := currentTX.Commit(); err != nil {
      return fmt.Errorf("failed to commit transaction: %w", err)
     }
    
     return nil
    }
    
    func IsWithinTransaction(ctx context.Context) bool {
     return ctx.Value(transactorKey{}) != nil
    }
    ```

### Repository Layer

แก้ไขในส่วนของ repository ให้เปลี่ยนจากการใช้ `dbCtx` จาก `sqldb.DBContext` มาเป็น  `transactor.DBContext` แทน คือ ทุกครั้งที่จะใช้งาน `db` ให้ดึงมาจาก `context` เพื่อจะได้มี transaction เดียวกัน

- แก้ไขไฟล์ `repository/customer.go`

    ```go
    package repository
    
    import (
     "context"
     "database/sql"
     "fmt"
     "go-mma/model"
     "go-mma/util/errs"
     "go-mma/util/storage/sqldb/transactor"
     "time"
    )
    
    type CustomerRepository struct {
     dbCtx transactor.DBContext // <-- ตรงนี้
    }
    
    func NewCustomerRepository(dbCtx transactor.DBContext) // <-- ตรงนี้
    *CustomerRepository {
     // ...
    }
    
    func (r *CustomerRepository) Create(ctx context.Context, customer *model.Customer) error {
     // ...
    
     err := r.dbCtx(ctx). // <-- ตรงนี้
     
     // ...
    }
    
    func (r *CustomerRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
     // ...
     
     err := r.dbCtx(ctx). // <-- ตรงนี้
     
     // ...
    }
    
    func (r *CustomerRepository) FindByID(ctx context.Context, id int64) (*model.Customer, error) {
     // ...
     
     err := r.dbCtx(ctx). // <-- ตรงนี้
     
     // ...
    }
    
    func (r *CustomerRepository) UpdateCredit(ctx context.Context, m *model.Customer) error {
     // ...
    
     err := r.dbCtx(ctx). // <-- ตรงนี้
     
     // ...
    }
    
    ```

- แก้ไขไฟล์ `repository/order.go`

    ```go
    package repository
    
    import (
     "context"
     "database/sql"
     "fmt"
     "go-mma/model"
     "go-mma/util/errs"
     "go-mma/util/storage/sqldb/transactor"
     "time"
    )
    
    type OrderRepository struct {
     dbCtx transactor.DBContext  // <-- ตรงนี้
    }
    
    func NewOrderRepository(dbCtx transactor.DBContext) // <-- ตรงนี้
    *OrderRepository {
     // ...
    }
    
    func (r *OrderRepository) Create(ctx context.Context, m *model.Order) error {
     // ...
    
     err := r.dbCtx(ctx). // <-- ตรงนี้
     
     // ...
    }
    
    func (r *OrderRepository) FindByID(ctx context.Context, id int64) (*model.Order, error) {
     // ...
     
     err := r.dbCtx(ctx). // <-- ตรงนี้
     
     // ...
    }
    
    func (r *OrderRepository) Cancel(ctx context.Context, id int64) error {
     // ...
     
     _, err := r.dbCtx(ctx). // <-- ตรงนี้
     
     // ...
    }
    ```

### Service Layer

เราจะจัดการควบคุม transaction ใน service layer โดยจะรับ transactor เข้ามาตอนสร้าง service

- แก้ไขไฟล์ `service/customer.go` ย้ายส่วนการบันทึกลูกค้าใหม่ กับส่งอีเมล มาไว้ใน `WithinTransaction`

    ```go
    package service
    
    // ...
    
    type CustomerService struct {
     transactor transactor.Transactor // <-- ตรงนี้
     custRepo   *repository.CustomerRepository
     notiSvc    *NotificationService
    }
    
    func NewCustomerService(
     transactor transactor.Transactor, // <-- ตรงนี้
     custRepo *repository.CustomerRepository,
     notiSvc *NotificationService,
    ) *CustomerService {
     return &CustomerService{
      transactor: transactor, // <-- ตรงนี้
      custRepo:   custRepo,
      notiSvc:    notiSvc,
     }
    }
    
    func (s *CustomerService) CreateCustomer(ctx context.Context, req *dto.CreateCustomerRequest) (*dto.CreateCustomerResponse, error) {
     // ...
     // แปลง DTO → Model
     customer := model.NewCustomer(req.Email, req.Credit)
    
      // ย้ายส่วนที่ติดต่อฐานข้อมูล กับส่งอีเมลมาทำงานใน WithinTransaction // <-- ตรงนี้
     err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
    
      // ส่งไปที่ Repository Layer เพื่อบันทึกข้อมูลลงฐานข้อมูล
      if err := s.custRepo.Create(ctx, customer); err != nil {
       // error logging
       logger.Log.Error(err.Error())
       return err
      }
    
      // ส่งอีเมลต้อนรับ
      if err := s.notiSvc.SendEmail(customer.Email, "Welcome to our service!", map[string]any{
       "message": "Thank you for joining us! We are excited to have you as a member.",
      }); err != nil {
       // error logging
       logger.Log.Error(err.Error())
       return err
      }
    
      return nil
     })
    
     if err != nil {
      return nil, err
     }
    
     // สร้าง DTO Response
     // ...
    }
    ```

- แก้ไขไฟล์ `service/order.go` ย้ายส่วนที่ติดต่อฐานข้อมูล กับส่งอีเมล มาไว้ใน `WithinTransaction`

    ```go
    package service
    
    // ...
    
    type OrderService struct {
     transactor transactor.Transactor // <-- ตรงนี้
     custRepo   *repository.CustomerRepository
     orderRepo  *repository.OrderRepository
     notiSvc    *NotificationService
    }
    
    func NewOrderService(
     transactor transactor.Transactor, // <-- ตรงนี้
     custRepo *repository.CustomerRepository,
     orderRepo *repository.OrderRepository,
     notiSvc *NotificationService) *OrderService {
     return &OrderService{
      transactor: transactor, // <-- ตรงนี้
      custRepo:   custRepo,
      orderRepo:  orderRepo,
      notiSvc:    notiSvc,
     }
    }
    
    func (s *OrderService) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) (int, error) {
     // Business Logic Rule: ตรวจสอบ customer id
     // ...
    
     // Business Logic Rule: ตัดยอด credit ถ้าไม่พอให้ error
     // ...
    
     // ย้ายส่วนที่ติดต่อฐานข้อมูล กับส่งอีเมลมาทำงานใน WithinTransaction  // <-- ตรงนี้
     var order *model.Order
     err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
    
      // ตัดยอด credit ในตาราง customer
      if err := s.custRepo.UpdateCredit(ctx, customer); err != nil {
       logger.Log.Error(err.Error())
       return err
      }
    
      // สร้าง order ใหม่
      order = model.NewOrder(req.CustomerID, req.OrderTotal)
      err = s.orderRepo.Create(ctx, order)
      if err != nil {
       logger.Log.Error(err.Error())
       return err
      }
    
      err = s.notiSvc.SendEmail(customer.Email, "Order Created", map[string]any{
       "order_id": order.ID,
       "total":    order.OrderTotal,
      })
      if err != nil {
       logger.Log.Error(err.Error())
       return err
      }
    
      return nil
     })
    
     if err != nil {
      return 0, err
     }
    
     return order.ID, nil
    }
    ```

### Dependency Injection

เพิ่มการ inject transactor เข้าไปใน service layer

- แก้ไขไฟล์ `application/http.go`

    ```go
    func (s *httpServer) RegisterRoutes(db sqldb.DBContext) {
     v1 := s.app.Group("/api/v1")
    
     transactor, dbCtx := transactor.New(db.DB()) // <-- ตรงนี้
     customers := v1.Group("/customers")
     {
      repo := repository.NewCustomerRepository(dbCtx) // <-- ตรงนี้
      svcNoti := service.NewNotificationService()
      svc := service.NewCustomerService(transactor, repo, svcNoti) // <-- ตรงนี้
      hdlr := handler.NewCustomerHandler(svc)
      customers.Post("", hdlr.CreateCustomer)
     }
    
     orders := v1.Group("/orders")
     {
      repoCust := repository.NewCustomerRepository(dbCtx) // <-- ตรงนี้
      repoOrder := repository.NewOrderRepository(dbCtx) // <-- ตรงนี้
      svcNoti := service.NewNotificationService()
      svcCust := service.NewOrderService(transactor, repoCust, repoOrder, svcNoti) // <-- ตรงนี้
      hdlr := handler.NewOrderHandler(svcCust)
      orders.Post("", hdlr.CreateOrder)
      orders.Delete("/:orderID", hdlr.CancelOrder)
     }
    }
    ```

## Unit of Work (UoW)

Unit of Work (UoW) คือ design pattern ที่ใช้จัดการ การทำงานแบบกลุ่มของ operations ให้อยู่ใน transaction เดียวกัน เพื่อให้ระบบมีความสอดคล้อง (consistency) และไม่เกิด partial updates ที่อาจทำให้ข้อมูลเสียหาย

### องค์ประกอบของ Unit of Work

1. Start / Begin: เริ่มต้น transaction
2. Register Changes: เก็บรายการ operation ที่จะทำ (insert, update, delete)
3. Commit: ถ้าทุกอย่างผ่าน → commit DB
4. Rollback: ถ้า error เกิดขึ้น → ยกเลิกทั้งหมด (rollback)
5. Post-Commit Hook: รัน side effects (เช่น send email) **หลังจาก** commit สำเร็จ

### Refactor ตัวจัดการ database transaction

ถ้าเทียบกับ `transactor` ที่ทำเอาไว้แล้วนั้น มีองค์ประกอบเกือบครบแล้ว ยังขาดแค่ Post-Commit Hook โดยให้แก้ไข ดังนี้

- แก้ไขไฟล์ `util/storage/sqldb/transactor/transactor.go` เพื่อเพิ่มเรื่อง Post-Commit Hook

    ```go
    // Ref: https://github.com/Thiht/transactor/blob/main/sqlx/transactor.go
    package transactor
    
    import (
     "context"
     "fmt"
     "go-mma/shared/common/logger" // <-- เพิ่ม
    
     "github.com/jmoiron/sqlx"
    )
    
    type PostCommitHook func(ctx context.Context) error // <-- เพิ่ม
    
    type Transactor interface {
     WithinTransaction(ctx context.Context, txFunc func(ctxWithTx context.Context, registerPostCommitHook func(PostCommitHook) // <-- เพิ่ม
     ) error) error
    }
    
    // ...
    
    func (t *sqlTransactor) WithinTransaction(ctx context.Context, txFunc func(ctxWithTx context.Context, registerPostCommitHook func(PostCommitHook) // <-- เพิ่ม
    ) error) error {
     currentDB := t.sqlxDBGetter(ctx)
    
     tx, err := currentDB.BeginTxx(ctx, nil)
     if err != nil {
      return fmt.Errorf("failed to begin transaction: %w", err)
     }
    
     var hooks []PostCommitHook // <-- เพิ่ม
    
     registerPostCommitHook := func(hook PostCommitHook) { // <-- เพิ่ม
      hooks = append(hooks, hook)
     }
    
     newDB, currentTX := t.nestedTransactionsStrategy(currentDB, tx)
     defer func() {
      _ = currentTX.Rollback() // If rollback fails, there's nothing to do, the transaction will expire by itself
     }()
     ctxWithTx := txToContext(ctx, newDB)
    
     if err := txFunc(
      ctxWithTx, 
      registerPostCommitHook, // <-- เพิ่ม
     ); err != nil {
      return err
     }
    
     if err := currentTX.Commit(); err != nil {
      return fmt.Errorf("failed to commit transaction: %w", err)
     }
    
     // <-- เพิ่ม
     // หลังจาก commit แล้ว รัน hook แบบ isolated
     go func() {
      for _, hook := range hooks {
       func(h PostCommitHook) {
        defer func() {
         if r := recover(); r != nil {
          // Log panic ที่เกิดใน hook
          logger.Log.Error(fmt.Sprintf("post-commit hook panic: %v", r))
         }
        }()
        if err := h(ctx); err != nil {
         logger.Log.Error(fmt.Sprintf("post-commit hook error: %v", err))
        }
       }(hook)
      }
     }()
    
     return nil
    }
    ```

- การใช้งาน

    แก้ไขไฟล์ `service/customer.go`

    ```go
    func (s *CustomerService) CreateCustomer(ctx context.Context, req *dto.CreateCustomerRequest) (*dto.CreateCustomerResponse, error) {
     // ...
     
     err = s.transactor.WithinTransaction(ctx, func(ctx context.Context, 
     registerPostCommitHook func(transactor.PostCommitHook), // เพิ่มตรงนี้
     ) error {
    
      // ส่งไปที่ Repository Layer เพื่อบันทึกข้อมูลลงฐานข้อมูล
      if err := s.custRepo.Create(ctx, customer); err != nil {
       // error logging
       logger.Log.Error(err.Error())
       return err
      }
    
      // แก้ตรงนี้
      // เพิ่มส่งอีเมลต้อนรับ เข้าไปใน hook แทน การเรียกใช้งานทันที
      registerPostCommitHook(func(ctx context.Context) error {
       return h.notiSvc.SendEmail(customer.Email, "Welcome to our service!", map[string]any{
       "message": "Thank you for joining us! We are excited to have you as a member."})
      })
      
      return nil
     })
    
     // ...
    }
    ```

    แก้ไขไฟล์ `service/order.go`

    ```go
    func (s *OrderService) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) (int, error) {
     // ...
     err = s.transactor.WithinTransaction(ctx, func(ctx context.Context, 
     registerPostCommitHook func(transactor.PostCommitHook), // เพิ่มตรงนี้
     ) error {
    
      // ตัดยอด credit ในตาราง customer
      if err := s.custRepo.UpdateCredit(ctx, customer); err != nil {
       logger.Log.Error(err.Error())
       return err
      }
    
      // สร้าง order ใหม่
      order = model.NewOrder(req.CustomerID, req.OrderTotal)
      err = s.orderRepo.Create(ctx, order)
      if err != nil {
       logger.Log.Error(err.Error())
       return err
      }
    
        // แก้ตรงนี้
      registerPostCommitHook(func(ctx context.Context) error {
       return h.notiSvc.SendEmail(customer.Email, "Order Created", map[string]any{
        "order_id": order.ID,
        "total":    order.OrderTotal,
       })
      })
    
      return nil
     })
    
     // ...
    }
    ```

## Dependency Inversion

Dependency Inversion คือ โค้ดส่วนหลัก (เช่น Handler, Service) ไม่ควรขึ้นกับโค้ดส่วนล่าง (เช่น Repository แบบเฉพาะเจาะจง), แต่ควรขึ้นกับ Interface แทน

มีเป้าหมาย คือ

- ลดการผูกติดกันของโค้ด (loose coupling)
- เปลี่ยน implementation ได้ง่าย เช่นเปลี่ยนจาก PostgreSQL → MongoDB
- ทำ unit test ได้ง่าย เพราะ mock ได้จาก interface

เมื่อใช้ Dependency Inversion

```go
┌────────────┐
│  Handler   │ ← struct: CustomerHandler
└────┬───────┘
     │ depends on interface
     ▼
┌────────────┐
│  Service   │  ← interface: CustomerService
└────┬───────┘
     │ implemented by
     ▼
┌────────────────────┐
│ ServiceImp         │ ← struct: customerService
└────────────────────┘
     │ depends on interface
     ▼
┌────────────┐
│ Repository │  ← interface: CustomerRepository
└────┬───────┘
     │ implemented by
     ▼
┌────────────────────┐
│ PostgresRepository │ ← struct: customerRepository
└────────────────────┘
```

### Repository Layer

- แก้ไขไฟล์ `repository/customer.go`

    ```go
    package repository
    
    // ...
    
    // --> Step 1: สร้าง interface
    type CustomerRepository interface {
     Create(ctx context.Context, customer *model.Customer) error
     ExistsByEmail(ctx context.Context, email string) (bool, error)
     FindByID(ctx context.Context, id int) (*model.Customer, error)
     UpdateCredit(ctx context.Context, customer *model.Customer) error
    }
    
    type customerRepository struct { // --> Step 2: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
     dbCtx transactor.DBContext
    }
    
    // --> Step 3: return เป็น interface
    func NewCustomerRepository(dbCtx transactor.DBContext) CustomerRepository {
     return &customerRepository{ // --> Step 4: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
      dbCtx: dbCtx,
     }
    }
    
    // --> Step 5: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    func (r *customerRepository) Create(ctx context.Context, customer *model.Customer) error {
     // ...
    }
    
    // --> Step 6: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    func (r *customerRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
     // ...
    }
    
    // --> Step 7: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    func (r *customerRepository) FindByID(ctx context.Context, id int) (*model.Customer, error) {
     // ...
    }
    
    // --> Step 8: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    func (r *customerRepository) UpdateCredit(ctx context.Context, m *model.Customer) error {
     // ...
    }
    ```

- แก้ไขไฟล์ `repository/order.go`

    ```go
    package repository
    
    // ...
    
    // --> Step 1: สร้าง interface
    type OrderRepository interface {
     Create(ctx context.Context, order *model.Order) error
     FindByID(ctx context.Context, id int) (*model.Order, error)
     Cancel(ctx context.Context, id int) error
    }
    
    type orderRepository struct { // --> Step 2: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
     dbCtx transactor.DBContext
    }
    
    // --> Step 3: return เป็น interface
    func NewOrderRepository(dbCtx transactor.DBContext) OrderRepository {
     return &orderRepository{ // --> Step 4: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
      dbCtx: dbCtx,
     }
    }
    
    // --> Step 5: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    func (r *orderRepository) Create(ctx context.Context, m *model.Order) error {
     // ...
    }
    
    // --> Step 6: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    func (r *orderRepository) FindByID(ctx context.Context, id int) (*model.Order, error) {
     // ...
    }
    
    // --> Step 7: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    func (r *orderRepository) Cancel(ctx context.Context, id int) error {
     // ...
    }
    ```

### Service Layer

- แก้ไขไฟล์ `service/notification.go`

    ```go
    package service
    
    import (
     "fmt"
     "go-mma/util/logger"
    )
    
    // --> Step 1: สร้าง interface
    type NotificationService interface {
     SendEmail(to string, subject string, payload map[string]any) error
    }
    
    // --> Step 2: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    type notificationService struct {
    }
    
    // --> Step 3: return เป็น interface
    func NewNotificationService() NotificationService {
     return &notificationService{} // --> Step 4: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    }
    
    // --> Step 5: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    func (s *notificationService) SendEmail(to string, subject string, payload map[string]any) error {
     // ...
    }
    ```

- แก้ไขไฟล์ `service/customer.go`

    ```go
    package service
    
    // ...
    
    // --> Step 1: สร้าง interface
    type CustomerService interface {
     CreateCustomer(ctx context.Context, req *dto.CreateCustomerRequest) (*dto.CreateCustomerResponse, error)
    }
    
    // --> Step 2: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    type customerService struct {
     transactor transactor.Transactor
     custRepo   repository.CustomerRepository // --> step 3: เปลี่ยนจาก pointer เป็น interface
     notiSvc    NotificationService // --> step 4: เปลี่ยนจาก pointer เป็น interface
    }
    
    func NewCustomerService(
     transactor transactor.Transactor,
     custRepo repository.CustomerRepository, // --> step 5: เปลี่ยนจาก pointer เป็น interface
     notiSvc NotificationService, // --> step 6: เปลี่ยนจาก pointer เป็น interface
    ) CustomerService {            // --> Step 7: return เป็น interface
     return &customerService{     // --> Step 8: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
      transactor: transactor,
      custRepo:   custRepo,
      notiSvc:    notiSvc,
     }
    }
    
    // --> Step 9: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    func (s *customerService) CreateCustomer(ctx context.Context, req *dto.CreateCustomerRequest) (*dto.CreateCustomerResponse, error) {
     // ...
    }
    ```

- แก้ไขไฟล์ `service/order.go`

    ```go
    package service
    
    // ...
    
    // --> Step 1: สร้าง interface
    type OrderService interface {
     CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) (*dto.CreateOrderResponse, error)
     CancelOrder(ctx context.Context, id int) error
    }
    
    // --> Step 2: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    type orderService struct {
     transactor transactor.Transactor
     custRepo   repository.CustomerRepository // --> step 3: เปลี่ยนจาก pointer เป็น interface
     orderRepo  repository.OrderRepository // --> step 4: เปลี่ยนจาก pointer เป็น interface
     notiSvc    NotificationService // --> step 5: เปลี่ยนจาก pointer เป็น interface
    }
    
    func NewOrderService(
     transactor transactor.Transactor,
     custRepo repository.CustomerRepository, // --> step 6: เปลี่ยนจาก pointer เป็น interface
     orderRepo repository.OrderRepository, // --> step 7: เปลี่ยนจาก pointer เป็น interface
     notiSvc NotificationService, // --> step 8: เปลี่ยนจาก pointer เป็น interface
     ) OrderService {.            // --> Step 9: return เป็น interface
     return &orderService{.       // --> Step 10: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
      transactor: transactor,
      custRepo:   custRepo,
      orderRepo:  orderRepo,
      notiSvc:    notiSvc,
     }
    }
    
    // --> Step 11: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    func (s *orderService) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) (*dto.CreateOrderResponse, error) {
     // ...
    }
    
    // --> Step 12: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
    func (s *orderService) CancelOrder(ctx context.Context, id int) error {
     // ...
    }
    ```

### Presentation Layer

- แก้ไขไฟล์ `handler/customer.go` แก้ให้รับ service มาเป็น interface

    ```go
    package handler
    
    // ...
    
    type CustomerHandler struct {
     custService service.CustomerService // <-- ตรงนี้
    }
    
    func NewCustomerHandler(custService service.CustomerService) // <-- ตรงนี้
    *CustomerHandler {
     return &CustomerHandler{
      custService: custService,
     }
    }
    ```

- แก้ไขไฟล์ `handler/order.go` แก้ให้รับ service มาเป็น interface

    ```go
    package handler
    
    // ...
    
    type OrderHandler struct {
     orderSvc service.OrderService // <-- ตรงนี้
    }
    
    func NewOrderHandler(orderSvc service.OrderService) // <-- ตรงนี้
    *OrderHandler {
     return &OrderHandler{orderSvc: orderSvc}
    }
    ```

## จัดวางโครงสร้างแบบ Modular

ถัดมาเราจะมาเปลี่ยนโครงสร้างจากที่แยกตาม "layer" (เช่น handler, service, repository) ไปเป็นการแยกตาม "feature หรือ use case” โดยใช้หลักการของ [Vertical Slice Architecture](https://somprasongd.work/blog/architecture/vertical-slice) คือ

- แยกตามฟีเจอร์ เช่น `customer`, `order`, `notification`
- ภายในแต่ละฟีเจอร์มีโค้ดของมันเอง: `handler`, `dto`, `service`, `model`, `repository`, `test`
- ทำให้ แยกอิสระ, ลดการพึ่งพาข้าม slice, เพิ่ม modularity

### โครงสร้างใหม่

```bash
.
├── cmd
│   └── api
│       └── main.go         # bootstraps all modules
├── config
│   └── config.go
├── modules                 
│   ├── customer
│   │   ├── handler
│   │   ├── dto
│   │   ├── model
│   │   ├── repository
│   │   ├── service
│   │   └── module.go       # wiring
│   ├── notification
│   │   ├── service
│   │   └── module.go 
│   └── order
│       ├── handler
│       ├── dto
│       ├── model
│       ├── repository
│       ├── service
│       └── module.go
├── application
│   ├── application.go      # register all modules
│   ├── http.go             # remove register all routes
│   └── middleware
│       ├── request_logger.go
│       └── response_error.go
├── migrations
│   └── ...sql
├── util
│   ├── module              # new
│   │   └── module.go       # module interface
│   └── ...
└── go.mod
```

### Notification Module

ทำการย้ายโค้ดทีเกี่ยวกับ notification มาไว้ที่ `modules/notification`

- ย้ายไฟล์ `service/notification.go` มาไว้ที่ `modules/notification/service/notification.go`
- สร้างไฟล์ `modules/notification/module.go`

    ```go
    package notification
    
    import (
     "go-mma/util/module"
    
     "github.com/gofiber/fiber/v3"
    )
    
    func New(mCtx *module.ModuleContext) module.Module {
     return &moduleImp{mCtx}
    }
    
    type moduleImp struct {
     mCtx *module.ModuleContext
    }
    
    func (m *moduleImp) RegisterRoutes(router fiber.Router) {
    
    }
    ```

### Customer Module

ทำการย้ายโค้ดทีเกี่ยวกับ customer มาไว้ที่ `modules/customer`

- ย้ายไฟล์ `model/customer.go` มาไว้ที่ `modules/customer/model/customer.go`
- ย้ายไฟล์ `dto/customer_*.go` มาไว้ที่ `modules/customer/dto/customer_*.go`
- ย้ายไฟล์ `repository/customer.go` มาไว้ที่ `modules/customer/repository/customer.go`

    ```go
    package repository
    
    import (
     "context"
     "database/sql"
     "fmt"
     "go-mma/modules/customer/model" // <-- แก้ตรงนี้ด้วย
     "go-mma/util/errs"
     "go-mma/util/storage/sqldb/transactor"
     "time"
    )
    ```

- ย้ายไฟล์ `service/customer.go` มาไว้ที่ `modules/customer/service/customer.go`

    ```go
    package service
    
    import (
     "context"
     "go-mma/modules/customer/dto"        // <-- แก้ตรงนี้ด้วย
     "go-mma/modules/customer/model"      // <-- แก้ตรงนี้ด้วย
     "go-mma/modules/customer/repository" // <-- แก้ตรงนี้ด้วย
     "go-mma/util/errs"
     "go-mma/util/logger"
     "go-mma/util/storage/sqldb/transactor"
    
     notiService "go-mma/modules/notification/service" // <-- แก้ตรงนี้ด้วย
    )
    
    // ...
    
    type customerService struct {
     transactor transactor.Transactor
     custRepo   repository.CustomerRepository
     notiSvc    notiService.NotificationService // <-- แก้ตรงนี้ด้วย
    }
    
    func NewCustomerService(
     transactor transactor.Transactor,
     custRepo repository.CustomerRepository,
     notiSvc notiService.NotificationService, // <-- แก้ตรงนี้ด้วย
    ) CustomerService {
     // ...
    }
    ```

- ย้ายไฟล์ `handler/customer.go` มาไว้ที่ `modules/customer/handler/customer.go`

    ```go
    package handler
    
    import (
     "go-mma/modules/customer/dto".    // <-- แก้ตรงนี้ด้วย
     "go-mma/modules/customer/service" // <-- แก้ตรงนี้ด้วย
     "go-mma/util/errs"
     "strings"
    
     "github.com/gofiber/fiber/v3"
    )
    ```

- ย้ายไฟล์ `tests/customer.http` มาไว้ที่ `modules/customer/test/customer.http`

### Order Module

ทำการย้ายโค้ดทีเกี่ยวกับ order มาไว้ที่ `modules/order`

- ย้ายไฟล์ `model/order.go` มาไว้ที่ `modules/customer/model/order.go`
- ย้ายไฟล์ `dto/order*.go` มาไว้ที่ `modules/customer/dto/order*.go`
- ย้ายไฟล์ `repository/order.go` มาไว้ที่ `modules/customer/repository/order.go`

    ```go
    package repository
    
    import (
     "context"
     "database/sql"
     "fmt"
     "go-mma/modules/order/model" // <-- แก้ตรงนี้ด้วย
     "go-mma/util/errs"
     "go-mma/util/storage/sqldb/transactor"
     "time"
    )
    ```

- ย้ายไฟล์ `service/order.go` มาไว้ที่ `modules/customer/service/order.go`

    ```go
    package service
    
    import (
     "context"
     "go-mma/modules/order/dto"        // <-- แก้ตรงนี้ด้วย
     "go-mma/modules/order/model"      // <-- แก้ตรงนี้ด้วย
     "go-mma/modules/order/repository" // <-- แก้ตรงนี้ด้วย
     "go-mma/util/errs"
     "go-mma/util/logger"
     "go-mma/util/storage/sqldb/transactor"
    
     custRepository "go-mma/modules/customer/repository" // <-- แก้ตรงนี้ด้วย
     notiService "go-mma/modules/notification/service"   // <-- แก้ตรงนี้ด้วย
    )
    
    // ...
    
    type orderService struct {
     transactor transactor.Transactor
     custRepo   custRepository.CustomerRepository // <-- แก้ตรงนี้ด้วย
     orderRepo  repository.OrderRepository
     notiSvc    notiService.NotificationService   // <-- แก้ตรงนี้ด้วย
    }
    
    func NewOrderService(
     transactor transactor.Transactor,
     custRepo custRepository.CustomerRepository,   // <-- แก้ตรงนี้ด้วย
     orderRepo repository.OrderRepository,
     notiSvc notiService.NotificationService       // <-- แก้ตรงนี้ด้วย
     ) OrderService {
     // ...
    }
    ```

- ย้ายไฟล์ `handler/order.go` มาไว้ที่ `modules/customer/handler/order.go`

    ```go
    package handler
    
    import (
     "go-mma/modules/order/dto"      // <-- แก้ตรงนี้ด้วย
     "go-mma/modules/order/service"  // <-- แก้ตรงนี้ด้วย
     "go-mma/util/errs"
     "strconv"
     "strings"
    
     "github.com/gofiber/fiber/v3"
    )
    ```

- ย้ายไฟล์ `tests/order.http` มาไว้ที่ `modules/customer/test/order.http`

### Feature-level constructor

คือ แนวคิดที่ใช้ *constructor function* เฉพาะสำหรับ "feature" หรือ "module" หนึ่ง ๆ ในระบบ เพื่อ ประกอบ dependencies ทั้งหมดของ "feature" หรือ "module" นั้นเข้าเป็นหน่วยเดียว และซ่อนไว้เบื้องหลัง interface หรือ struct เพื่อให้ใช้งานได้ง่ายและยืดหยุ่น

ตัวอย่างการใช้งาน

```go
// module/customer/module.go
func NewCustomerModule(mCtx *module.ModuleContext) module.Module {
 repo := repository.NewCustomerRepository(mCtx.DBCtx)
 svc := service.NewCustomerService(repo)
 hdl := handler.NewCustomerHandler(svc)

 return &customerModule{
  handler: hdl,
 }
}
```

ขั้นตอนการสร้าง

- สร้าง Module Interface

    สร้างไฟล์ `util/module/module.go`

    ```go
    package module
    
    import (
     "go-mma/util/transactor"
    
     "github.com/gofiber/fiber/v3"
    )
    
    type Module interface {
     APIVersion() string
     RegisterRoutes(r fiber.Router)
    }
    
    type ModuleContext struct {
     Transactor transactor.Transactor
     DBCtx      transactor.DBContext
    }
    
    func NewModuleContext(transactor transactor.Transactor, dbCtx transactor.DBContext) *ModuleContext {
     return &ModuleContext{
      Transactor: transactor,
      DBCtx:      dbCtx,
     }
    }
    ```

- สร้าง Notification Module โดยใช้ Factory pattern

    สร้างไฟล์ `modules/notification/module.go`

    ```go
    package notification
    
    import (
     "go-mma/util/module"
    
     "github.com/gofiber/fiber/v3"
    )
    
    func NewModule(mCtx *module.ModuleContext) module.Module {
     return &moduleImp{mCtx}
    }
    
    type moduleImp struct {
     mCtx *module.ModuleContext
    }
    
    func (m *moduleImp) APIVersion() string {
     return "v1"
    }
    
    func (m *moduleImp) RegisterRoutes(router fiber.Router) {
    
    }
    ```

- สร้าง Customer Module โดยใช้ Factory pattern และย้ายการ wiring component ต่าง ๆ (เช่น repository, service, handler) สำหรับ customer จาก `application/http.go` มาใส่ `RegisterRoutes()`

    สร้างไฟล์ `modules/customer/module.go`

    ```go
    package customer
    
    import (
     "go-mma/modules/customer/handler"
     "go-mma/modules/customer/repository"
     "go-mma/modules/customer/service"
     "go-mma/util/module"
    
     notiService "go-mma/modules/notification/service"
    
     "github.com/gofiber/fiber/v3"
    )
    
    func NewModule(mCtx *module.ModuleContext) module.Module {
     return &moduleImp{mCtx}
    }
    
    type moduleImp struct {
     mCtx *module.ModuleContext
    }
    
    func (m *moduleImp) APIVersion() string {
     return "v1"
    }
    
    func (m *moduleImp) RegisterRoutes(router fiber.Router) {
     // wiring dependencies
     repo := repository.NewCustomerRepository(m.mCtx.DBCtx)
     svcNoti := notiService.NewNotificationService()
     svc := service.NewCustomerService(m.mCtx.Transactor, repo, svcNoti)
     hdl := handler.NewCustomerHandler(svc)
    
     customers := router.Group("/customers")
     customers.Post("", hdl.CreateCustomer)
    }
    ```

- สร้าง Order Module โดยใช้ Factory pattern และย้ายการ wiring component ต่าง ๆ (เช่น repository, service, handler) สำหรับ order จาก `application/http.go` มาใส่ `RegisterRoutes()`

    สร้างไฟล์ `modules/order/module.go`

    ```go
    package order
    
    import (
     "go-mma/modules/order/handler"
     "go-mma/modules/order/repository"
     "go-mma/modules/order/service"
     "go-mma/util/module"
    
     custRepository "go-mma/modules/customer/repository"
     notiService "go-mma/modules/notification/service"
    
     "github.com/gofiber/fiber/v3"
    )
    
    func NewModule(mCtx *module.ModuleContext) module.Module {
     return &moduleImp{mCtx}
    }
    
    type moduleImp struct {
     mCtx *module.ModuleContext
    }
    
    func (m *moduleImp) APIVersion() string {
     return "v1"
    }
    
    func (m *moduleImp) RegisterRoutes(router fiber.Router) {
     // wiring dependencies
     repoCust := custRepository.NewCustomerRepository(m.mCtx.DBCtx)
     repoOrder := repository.NewOrderRepository(m.mCtx.DBCtx)
     svcNoti := notiService.NewNotificationService()
     svc := service.NewOrderService(m.mCtx.Transactor, repoCust, repoOrder, svcNoti)
     hdl := handler.NewOrderHandler(svc)
    
     orders := router.Group("/orders")
     orders.Post("", hdl.CreateOrder)
     orders.Delete("/:orderID", hdl.CancelOrder)
    }
    ```

- ลบ `RegisterRoutes()` ใน `application/http.go` และเพิ่มเติมโค้ดตามนี้

    ```go
    type HTTPServer interface {
     Start()
     Shutdown() error
     Group(prefix string) fiber.Router  // <-- ตรงนี้
    }
    
    func (s *httpServer) Group(prefix string) fiber.Router {
     return s.app.Group(prefix)
    }
    ```

- ลบ `RegisterRoutes()` ใน `application/application.go` และเพิ่ม `RegisterModules()` เข้าไปแทน

    ```go
    func (app *Application) RegisterModules(modules ...module.Module) error {
     for _, m := range modules {
      app.registerModuleRoutes(m)
     }
    
     return nil
    }
    
    func (app *Application) registerModuleRoutes(m module.Module) {
     prefix := app.buildGroupPrefix(m)
     group := app.httpServer.Group(prefix)
     m.RegisterRoutes(group)
    }
    
    func (app *Application) buildGroupPrefix(m module.Module) string {
     apiBase := "/api"
     version := m.APIVersion()
     if version != "" {
      return fmt.Sprintf("%s/%s", apiBase, version)
     }
     return apiBase
    }
    ```

- ลบ `app.RegisterRoutes()` ใน `cmd/api/main.go` และเพิ่มโค้ดเพื่อสร้างโมดูล

    ```go
    package main
    
    import (
     "fmt"
     "go-mma/application"
     "go-mma/config"
     "go-mma/data/sqldb"
     "go-mma/modules/customer"
     "go-mma/modules/notification"
     "go-mma/modules/order"
     "go-mma/util/logger"
     "go-mma/util/module"
     "go-mma/util/transactor"
     "os"
     "os/signal"
     "syscall"
    )
    
    func main() {
     // log
     // config
     // db
    
     app := application.New(*config, db)
    
     transactor, dbCtx := transactor.New(db.DB())
     mCtx := module.NewModuleContext(transactor, dbCtx)
     app.RegisterModules(
      notification.NewModule(mCtx),
      customer.NewModule(mCtx),
      order.NewModule(mCtx),
     )
    
     app.Run()
    
     // ...
    }
    ```

## Encapsulating a subdomain behind a facade

คือแนวคิดในการ ซ่อนรายละเอียดภายในของ subdomain (กลุ่มของ business logic ที่อยู่ภายใต้โดเมนหลัก เช่น `Customer`, `Order`, `Notification`) ไว้เบื้องหลัง "facade" ซึ่งเป็น interface หรือ entry point เดียว ที่ใช้ติดต่อกับ subdomain นั้น ๆ เพื่อใช้แยกขอบเขตของระบบ (Bounded Context) ให้ชัด และลดความซับซ้อนในการสื่อสารระหว่าง module ต่าง ๆ

<aside>
💡

Facade = จุดเชื่อมต่อเดียว (public API) ที่ให้ module อื่นเข้าถึง functionality ของ subdomain โดยไม่รู้โครงสร้างภายใน

</aside>

### ตัวอย่างการใช้งาน

ก่อน: ระบบเข้าถึงหลายชั้นโดยตรง

```
Order Handler
     │
     ▼
Order Service
     │
     ├──────────────▶ Order Repository
     │
     └──────────────▶ Customer Repository
```

ตัวอย่าง

```go
// OrderService เรียก CustomerRepository ตรง ๆ
customer, err := customerRepo.FindByID(ctx, order.CustomerID)
if customer.Credit < order.Total {
    return errors.New("insufficient credit")
}
```

- `Order Service` เรียกทั้ง `OrderRepo` และ `CustomerRepo` โดยตรง

หลัง: ใช้ Encapsulation

```
Order Handler
     │
     ▼
Order Service
     │
     ├──────────────▶ Order Repository
     │
     └──────────────▶ Customer Service
                             │
                             └────────▶ Customer Repository

```

ตัวอย่าง

```go
// OrderService ใช้ CustomerFacade แทน
ok, err := customerService.HasSufficientCredit(ctx, order.CustomerID, order.Total)
if !ok {
    return errors.New("insufficient credit")
}
```

- `CustomerService`  เป็นจุดเดียวที่เปิดเผย logic ภายใน subdomain customer
- ภายใน facade จะจัดการ repository, validation, business rule ทั้งหมดเอง

### ซ่อนรายละเอียดภายในของ Customer

ตอนนี้ใน OrderService มีการเรียกใช้ `model` และ `repository` ของโมดูล customer โดยตรง ถ้าจะซ่อนรายละเอียดภายใน ทำได้ ดังนี้

- สร้าง DTO สำหรับส่งค่า customer กลับออกไปจาก CustomerService สำหรับการค้นหาลูกค้าจาก id

    สร้างไฟล์ `module/customer/dto/customer_info.go`

    ```go
    package dto
    
    type CustomerInfo struct {
     ID     int    `json:"id"`
     Email  string `json:"email"`
     Credit int    `json:"credit"`
    }
    ```

- เพื่อเพิ่มฟังก์ชันสำหรับการค้นหาจาก id

    แก้ไข `module/customer/service/customer.go`

    ```go
    var (
     ErrEmailExists      = errs.ConflictError("email already exists")
     ErrCustomerNotFound = errs.ResourceNotFoundError("the customer with given id was not found") // <-- ตรงนี้
    )
    
    type CustomerService interface {
     CreateCustomer(ctx context.Context, req *dto.CreateCustomerRequest) (*dto.CreateCustomerResponse, error)
     GetCustomerByID(ctx context.Context, id int) (*dto.CustomerInfo, error) // <-- ตรงนี้
    }
    
    // ...
    
    // <-- ตรงนี้
    func (s *customerService) GetCustomerByID(ctx context.Context, id int) (*dto.CustomerInfo, error) {
     customer, err := s.custRepo.FindByID(ctx, id)
     if err != nil {
      // error logging
      logger.Log.Error(err.Error())
      return nil, err
     }
    
     if customer == nil {
      return nil, ErrCustomerNotFound
     }
    
     // สร้าง DTO Response
     return &dto.CustomerInfo{
      ID:     customer.ID,
      Email:  customer.Email,
      Credit: customer.Credit,
     }, nil
    }
    ```

- เพื่อเพิ่มฟังก์ชันสำหรับการตัดยอด และคืนยอด credit

    แก้ไข `module/customer/service/customer.go`

    ```go
    var (
     ErrEmailExists                  = errs.ConflictError("email already exists")
     ErrCustomerNotFound             = errs.ResourceNotFoundError("the customer with given id was not found")
     ErrOrderTotalExceedsCreditLimit = errs.BusinessRuleError("order total exceeds credit limit") // <-- ตรงนี้
    )
    
    type CustomerService interface {
     // ...
     ReserveCredit(ctx context.Context, id int, amount int) error // <-- ตรงนี้
     ReleaseCredit(ctx context.Context, id int, amount int) error // <-- ตรงนี้
    }
    
    // ...
    
    // <-- ตรงนี้
    func (s *customerService) ReserveCredit(ctx context.Context, id int, amount int) error {
     err := h.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
      customer, err := s.custRepo.FindByID(ctx, id)
      if err != nil {
       logger.Log.Error(err.Error())
       return err
      }
     
      if customer == nil {
       return ErrCustomerNotFound
      }
     
      if err := customer.ReserveCredit(amount); err != nil {
       return ErrOrderTotalExceedsCreditLimit
      }
     
      if err := s.custRepo.UpdateCredit(ctx, customer); err != nil {
       logger.Log.Error(err.Error())
       return err
      }
     
      return nil
     }
     return err
    }
    
    func (s *customerService) ReleaseCredit(ctx context.Context, id int, amount int) error {
     err := h.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
      customer, err := s.custRepo.FindByID(ctx, id)
      if err != nil {
       logger.Log.Error(err.Error())
       return err
      }
     
      if customer == nil {
       return ErrCustomerNotFound
      }
     
      customer.ReleaseCredit(amount)
     
      if err := s.custRepo.UpdateCredit(ctx, customer); err != nil {
       logger.Log.Error(err.Error())
       return err
      }
     
      return nil
     }
     
     return err
    }
    ```

### เรียกใช้ CustomerService ในโมดูล Order

- ทำให้ `OrderService` เรียกใช้งาน `CustomerService` แทน `CustomerRepository`

    แก้ไข `module/order/service/order.go`

    ```go
    package service
    
    import (
     "context"
     "go-mma/modules/order/dto"
     "go-mma/modules/order/model"
     "go-mma/modules/order/repository"
     "go-mma/util/errs"
     "go-mma/util/logger"
     "go-mma/util/transactor"
    
     custService "go-mma/modules/customer/service" // <-- ตรงนี้
     notiService "go-mma/modules/notification/service"
    )
    
    var (
     ErrNoOrderID = errs.ResourceNotFoundError("the order with given id was not found") // <-- ตรงนี้ เหลือแค่ตัวเดียว
    )
    
    // ...
    
    type orderService struct {
     transactor transactor.Transactor
     custSvc    custService.CustomerService // <-- ตรงนี้
     orderRepo  repository.OrderRepository
     notiSvc    notiService.NotificationService
    }
    
    func NewOrderService(
     transactor transactor.Transactor,
     custSvc custService.CustomerService, // <-- ตรงนี้
     orderRepo repository.OrderRepository,
     notiSvc notiService.NotificationService) OrderService {
     // ...
    }
    
    func (s *orderService) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) (*dto.CreateOrderResponse, error) {
     // Business Logic Rule: ตรวจสอบ customer id
     customer, err := s.custSvc.GetCustomerByID(ctx, req.CustomerID) // <-- ตรงนี้
     if err != nil {
      return nil, err
     }
     // ...
     // ย้ายส่วนที่ติดต่อฐานข้อมูล กับส่งอีเมลมาทำงานใน WithinTransaction
     var order *model.Order
     err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
    
      // ตัดยอด credit ในตาราง customer
      if err := s.custSvc.ReserveCredit(ctx, customer.ID, req.OrderTotal); err != nil { // <-- ตรงนี้
       return err
      }
    
      // ...
     })
    
     // ...
    }
    
    func (s *orderService) CancelOrder(ctx context.Context, id int64) error {
     // Business Logic Rule: ตรวจสอบ order id
     order, err := s.orderRepo.FindByID(ctx, id)
     if err != nil {
      logger.Log.Error(err.Error())
      return err
     }
    
     if order == nil {
      return ErrNoOrderID
     }
    
     err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
    
      // ยกเลิก order
      if err := s.orderRepo.Cancel(ctx, order.ID); err != nil {
       logger.Log.Error(err.Error())
       return err
      }
    
      // Business Logic: คืนยอด credit // <-- ตรงนี้
      err := s.custSvc.ReleaseCredit(ctx, order.CustomerID, order.OrderTotal)
      if err != nil {
       return err
      }
    
      return nil
     })
    
     return err
    }
    ```

### Nested Transactions

เมื่อทดสอบรันโปรแกรมใหม่อีกครั้ง แล้วลองสร้างออเดอร์ใหม่ จะได้รับ error ว่า

```go
HTTP/1.1 500 Internal Server Error
Date: Fri, 06 Jun 2025 03:50:32 GMT
Content-Type: application/json
Content-Length: 106
X-Request-Id: 771c27c3-7528-4b93-bc6d-e1696c4727ae
Connection: close

{
  "type": "operation_failed",
  "message": "failed to begin transaction: nested transactions are not supported"
}
```

เนื่องจากในฟังก์ชันการตัดยอด และคืนยอด credit ใน `CustomerService` นั้น มีการเปิดใช้งาน transaction ขึ้นมาใหม่ ซึ่งที่ถูกต้องจะต้องเป็น transaction เดียวกันที่ได้มาจาก `OrderService`

ดังนั้น ตอนสร้าง `transactor` ใน `main.go` ต้องระบุว่าด้วยว่าให้มีการใช้งาน nested transactions

```go
// src/app/cmd/api/main.go

func main() {
 // ...

 app := application.New(*config)

 transactor, dbCtx := transactor.New(
  db.DB(),
  transactor.WithNestedTransactionStrategy(transactor.NestedTransactionsSavepoints), // <-- ตรงนี้
 )
 mCtx := module.NewModuleContext(transactor, dbCtx)
 
 // ...
}
```

### ป้องกันการเข้าถึงข้ามโมดูลด้วยโฟลเดอร์ `internal`

หลังจากที่เรา แยกขอบเขตของแต่ละ sub-domain (Encapsulation) แล้ว ปัญหาที่ยังเหลือคือ โค้ดในโมดูล order ยังสามารถ `import` `model` หรือ `repository` ของโมดูล customer ได้โดยตรง นั่นทำให้ละเมิดขอบเขต (boundary) ของโดเมนและสร้างความพึ่งพา (coupling) ที่ไม่พึงประสงค์

ในภาษา Go สามารถย้ายไฟล์ที่ “ห้ามภายนอกใช้” เข้าไปไว้ภายใต้โฟลเดอร์ **`internal`** ได้ ตัวคอมไพเลอร์จะบังคับไม่ให้ path นอกโฟลเดอร์แม่ (root) ของ `internal` ทำ `import` ได้เลย

```go
customer/
├── internal/
│   ├── model/        // โครงสร้างข้อมูลเฉพาะ customer
│   └── repository/   // DB logic ของ customer
└── service/          // business logic (export)
```

ถ้าโมดูลอื่น เช่น `order` พยายาม `import "go-mma/modules/customer/internal/repository"` จะขึ้นข้อความ error แบบนี้

```go
could not import go-mma/modules/customer/internal/repository (invalid use of internal package "go-mma/modules/customer/internal/repository")
```

## Service Registry

โค้ดปัจจุบันจะเห็นว่ามีการสร้าง service ตัวเดิมซ้ำๆ กัน ในแต่ละโมดูล เนื้อหาในส่วนนี้จะแสดงวิธีการใช้ Service Registry ในการเก็บ service ทั้งหมดในระบบ เพื่อเรียกใช้งานได้เลย ไม่ต้องสร้างใหม่ทุกครั้ง

### สร้าง Service Registry

- สร้างไฟล์ `util/registry/service_registry.go`

    ```go
    package registry
    
    import "fmt"
    
    // สำหรับกำหนด key ของ service ที่จะ export
    type ServiceKey string
    
    // สำหรับ map key กับ service ที่จะ export
    type ProvidedService struct {
     Key   ServiceKey
     Value any
    }
    
    type ServiceRegistry interface {
     Register(key ServiceKey, svc any)
     Resolve(key ServiceKey) (any, error)
    }
    
    type serviceRegistry struct {
     services map[ServiceKey]any
    }
    
    func NewServiceRegistry() ServiceRegistry {
     return &serviceRegistry{
      services: make(map[ServiceKey]any),
     }
    }
    
    func (r *serviceRegistry) Register(key ServiceKey, svc any) {
     r.services[key] = svc
    }
    
    func (r *serviceRegistry) Resolve(key ServiceKey) (any, error) {
     svc, ok := r.services[key]
     if !ok {
      return nil, fmt.Errorf("service not found: %s", key)
     }
     return svc, nil
    }
    
    ```

- สร้างฟังก์ชันสำหรับช่วยแปลง Service กลับมาให้ถูกต้อง

    สร้างไฟล์ `util/registry/helper.go`

    ```go
    package registry
    
    import "fmt"
    
    func ResolveAs[T any](r ServiceRegistry, key ServiceKey) (T, error) {
     var zero T
     svc, err := r.Resolve(key)
     if err != nil {
      return zero, err
     }
     typedSvc, ok := svc.(T)
     if !ok {
      return zero, fmt.Errorf("service registered under key %s does not implement the expected type", key)
     }
     return typedSvc, nil
    }
    ```

### แก้ไข Module Interface

แก้ไขให้ Module มีฟังก์ชันสำหรับ เพิ่ม service ของตัวเองเข้า Registry

- แก้ไขไฟล์ `util/module/module.go`

    ```go
    package module
    
    import (
     "go-mma/util/registry"    // <-- ตรงนี้
     "go-mma/util/transactor"
    
     "github.com/gofiber/fiber/v3"
    )
    
    type Module interface {
     APIVersion() string
     Init(reg registry.ServiceRegistry) error // <-- ตรงนี้
     RegisterRoutes(r fiber.Router)
    }
    
    // <-- ตรงนี้
    // แยกออกมาเพราะว่า บางโมดูลอาจไม่ต้อง export service
    type ServiceProvider interface {
     Services() []registry.ProvidedService
    }
    ```

### แก้ไข Application

แก้ไข Application ให้เป็นที่เก็บ service registry

- แก้ไขไฟล์ `application/application.go`

    ```go
    package application
    
    import (
     "fmt"
     "go-mma/config"
     "go-mma/data/sqldb"
     "go-mma/util/logger"
     "go-mma/util/module"
     "go-mma/util/registry"  // <-- ตรงนี้
    )
    
    type Application struct {
     config          config.Config
     httpServer      HTTPServer
     serviceRegistry registry.ServiceRegistry // <-- ตรงนี้
    }
    
    func New(config config.Config, db sqldb.DBContext) *Application {
     return &Application{
      config:          config,
      httpServer:      newHTTPServer(config),
      serviceRegistry: registry.NewServiceRegistry(), // <-- ตรงนี้
     }
    }
    
    // ...
    
    func (app *Application) RegisterModules(modules ...module.Module) error {
     for _, m := range modules {
      // Initialize each module
      if err := app.initModule(m); err != nil {
       return fmt.Errorf("failed to init module [%T]: %w", m, err)
      }
    
      // ถ้าโมดูลเป็น ServiceProvider ให้เอา service มาลง registry
      if sp, ok := m.(module.ServiceProvider); ok {
       for _, p := range sp.Services() {
        app.serviceRegistry.Register(p.Key, p.Value)
       }
      }
    
      // Register routes for each module
      app.registerModuleRoutes(m)
     }
    
     return nil
    }
    
    func (app *Application) initModule(m module.Module) error {
     return m.Init(app.serviceRegistry)
    }
    
    // ...
    ```

### เพิ่มการ Initialize แต่ละโมดูล

ปรับให้แต่ละโมดูลเพิ่ม `Init()` เพื่อสร้าง service ของตัวเอง

- แก้ไขไฟล์ `modules/notification/module.go`

    ```go
    package notification
    
    import (
     "go-mma/modules/notification/service"
     "go-mma/util/module"
     "go-mma/util/registry"
    
     "github.com/gofiber/fiber/v3"
    )
    
    const (
     NotificationServiceKey registry.ServiceKey = "NotificationService"
    )
    
    func NewModule(mCtx *module.ModuleContext) module.Module {
     return &moduleImp{mCtx: mCtx}
    }
    
    type moduleImp struct {
     mCtx    *module.ModuleContext
     notiSvc service.NotificationService
    }
    
    func (m *moduleImp) APIVersion() string {
     return "v1"
    }
    
    func (m *moduleImp) Init(reg registry.ServiceRegistry) error {
     m.notiSvc = service.NewNotificationService()
    
     return nil
    }
    
    func (m *moduleImp) Services() []registry.ProvidedService {
     return []registry.ProvidedService{
      {Key: NotificationServiceKey, Value: m.notiSvc},
     }
    }
    
    func (m *moduleImp) RegisterRoutes(router fiber.Router) {
    
    }
    ```

- แก้ไขไฟล์ `modules/customer/module.go`

    ```go
    package customer
    
    import (
     "go-mma/modules/customer/handler"
     "go-mma/modules/customer/repository"
     "go-mma/modules/customer/service"
     "go-mma/util/module"
     "go-mma/util/registry"
    
     notiModule "go-mma/modules/notification"
     notiService "go-mma/modules/notification/service"
    
     "github.com/gofiber/fiber/v3"
    )
    
    const (
     CustomerServiceKey registry.ServiceKey = "CustomerService"
    )
    
    func NewModule(mCtx *module.ModuleContext) module.Module {
     return &moduleImp{mCtx: mCtx}
    }
    
    type moduleImp struct {
     mCtx    *module.ModuleContext
     custSvc service.CustomerService
    }
    
    func (m *moduleImp) APIVersion() string {
     return "v1"
    }
    
    func (m *moduleImp) Init(reg registry.ServiceRegistry) error {
     // Resolve NotificationService from the registry
     notiSvc, err := registry.ResolveAs[notiService.NotificationService](reg, notiModule.NotificationServiceKey)
     if err != nil {
      return err
     }
    
     repo := repository.NewCustomerRepository(m.mCtx.DBCtx)
     m.custSvc = service.NewCustomerService(m.mCtx.Transactor, repo, notiSvc)
    
     reg.Register(CustomerServiceKey, m.custSvc)
    
     return nil
    }
    
    func (m *moduleImp) RegisterRoutes(router fiber.Router) {
     // wiring dependencies
     hdl := handler.NewCustomerHandler(m.custSvc)
    
     customers := router.Group("/customers")
     customers.Post("", hdl.CreateCustomer)
    }
    ```

    ทำไมถึงสร้าง handler ใน `RegisterRoutes`

  - แยก concern ชัด: `RegisterRoutes` ดูแล “transport layer” ทั้งหมดในฟังก์ชันเดียว
  - อ่านง่าย: เห็นเส้นทางและ handler คู่กันทันที
  - ใช้ที่เดียว: ไม่มี state เพิ่มใน `moduleImp`
- แก้ไขไฟล์ `modules/order/module.go`

    ```go
    package order
    
    import (
     "go-mma/modules/order/handler"
     "go-mma/modules/order/repository"
     "go-mma/modules/order/service"
     "go-mma/util/module"
     "go-mma/util/registry"
    
     custModule "go-mma/modules/customer"
     custService "go-mma/modules/customer/service"
     notiModule "go-mma/modules/notification"
     notiService "go-mma/modules/notification/service"
    
     "github.com/gofiber/fiber/v3"
    )
    
    func NewModule(mCtx *module.ModuleContext) module.Module {
     return &moduleImp{mCtx: mCtx}
    }
    
    type moduleImp struct {
     mCtx     *module.ModuleContext
     orderSvc service.OrderService
    }
    
    func (m *moduleImp) APIVersion() string {
     return "v1"
    }
    
    func (m *moduleImp) Init(reg registry.ServiceRegistry) error {
     // Resolve CustomerService from the registry
     custSvc, err := registry.ResolveAs[custService.CustomerService](reg, custModule.CustomerServiceKey)
     if err != nil {
      return err
     }
    
     // Resolve NotificationService from the registry
     notiSvc, err := registry.ResolveAs[notiService.NotificationService](reg, notiModule.NotificationServiceKey)
     if err != nil {
      return err
     }
    
     repo := repository.NewOrderRepository(m.mCtx.DBCtx)
     m.orderSvc = service.NewOrderService(m.mCtx.Transactor, custSvc, repo, notiSvc)
    
     return nil
    }
    
    func (m *moduleImp) RegisterRoutes(router fiber.Router) {
     // wiring dependencies
     hdl := handler.NewOrderHandler(m.orderSvc)
    
     orders := router.Group("/orders")
     orders.Post("", hdl.CreateOrder)
     orders.Delete("/:orderID", hdl.CancelOrder)
    }
    ```

## จัดวางโครงสร้างแบบ Mono-Repository

เพื่อเพิ่มความชัดเจนและปลอดภัยยิ่งขึ้น ให้แยกโมดูลหลักแต่ละตัว (`customer`, `order`, `notification`) ออกเป็น โฟลเดอร์โปรเจกต์ย่อยที่มี `go.mod` ของตัวเอง แต่ยังเก็บอยู่ใน Git repository เดียวกัน (Mono Repository)

### โครงสร้างใหม่

โดยจะแบ่งโค้ดออกเป็น 3 ส่วน หลักๆ คือ

- app: สำหรับโหลดโมดูล และรันโปรแกรม
- modules: สำหรับสร้างโมดูลต่างๆ
- shared: สำหรับโค้ดที่ใช้งานร่วมกัน

```bash
.
├── docker-compose.dev.yml
├── docker-compose.yml
├── go-mma.code-workspace
├── Makefile
├── migrations
│   ├── 20250529103238_create_customer.down.sql
│   ├── 20250529103238_create_customer.up.sql
│   ├── 20250529103715_create_order.down.sql
│   └── 20250529103715_create_order.up.sql
└── src
    ├── app
    │   ├── application
    │   │   ├── application.go
    │   │   ├── http.go
    │   │   └── middleware.go
    │   │   │   ├── request_logger.go
    │   │   │   └── response_error.go
    │   ├── cmd
    │   │   └── api
    │   │       └── main.go
    │   ├── config
    │   │   └── config.go
    │   ├── go.mod
    │   ├── go.sum
    │   └── util
    │       └── env
    │           └── env.go
    ├── modules
    │   ├── customers
    │   │   ├── dtos
    │   │   │   ├── customer_request.go
    │   │   │   ├── customer_response.go
    │   │   │   └── customer.go
    │   │   ├── handler
    │   │   │   └── customer.go
    │   │   ├── internal
    │   │   │   ├── model
    │   │   │   │   └── customer.go
    │   │   │   └── repository
    │   │   │       └── customer.go
    │   │   ├── module.go
    │   │   ├── service
    │   │   │   └── customer.go
    │   │   ├── test
    │   │   │   └── customer.http
    │   │   ├── go.mod
    │   │   └── go.sum
    │   ├── notifications
    │   │   ├── module.go
    │   │   ├── service
    │   │   │   └── notification.go
    │   │   ├── go.mod
    │   │   └── go.sum
    │   └── orders
    │   │   ├── dtos
    │   │   │   ├── order_request.go
    │   │   │   └── order_response.go
    │   │   ├── handler
    │   │   │   └── order.go
    │   │   ├── internal
    │   │   │   ├── model
    │   │   │   │   └── order.go
    │   │   │   └── repository
    │   │   │       └── order.go
    │   │   ├── module.go
    │   │   ├── service
    │   │   │   └── order.go
    │   │   ├── test
    │   │   │   └── order.http
    │   │   ├── go.mod
    │   │   └── go.sum
    └── shared
        └──common
            ├── errs
            │   ├── errs.go
            │   ├── helper.go
            │   └── types.go
            ├── logger
            │   └── logger.go
            ├── module
            │   └── module.go
            ├── registry
            │   ├── helper.go
            │   └── service_registry.go
            ├── storage
            │   └── db
            │       ├── db.go
            │       └── transactor
            │           ├── nested_transactions_none.go
            │           ├── nested_transactions_savepoints.go
            │           ├── transactor.go
            │           └── types.go
            ├── go.mod
            └── go.sum
```

### สร้างโปรเจกต์ใหม่

- สร้าง Folder ใหม่ ดังนี้

    ```bash
    mkdir -p src/app
    mkdir -p src/modules/customer
    mkdir -p src/modules/notification
    mkdir -p src/modules/order
    mkdir -p src/shared/common
    ```

- สร้าง app โปรเจกต์

    ```bash
    cd src/app
    go mod init go-mma
    ```

- สร้าง customer โปรเจกต์

    ```bash
    cd src/modules/customer
    go mod init go-mma/modules/customer
    ```

- สร้าง notification โปรเจกต์

    ```bash
    cd src/modules/notification
    go mod init go-mma/modules/notification
    ```

- สร้าง order โปรเจกต์

    ```bash
    cd src/modules/order
    go mod init go-mma/modules/order
    ```

- สร้าง common โปรเจค

    ```bash
    cd src/shared/common
    go mod init go-mma/shared/common
    ```

### ทำ L**ocal module replacement**

เมื่อพัฒนาแบบ Monorepo เพื่อให้แต่ละโมดูลสามารถอ้างถึงกันได้โดยตรงจากไฟล์ในเครื่อง โดยไม่ต้อง publish ไปที่ remote repo ใน Go ทำได้โดย การใช้คำสั่ง `replace` ใน `go.mod`

- โปรเจกต์ notification มีการใช้งาน common

    แก้ไขไฟล์ `src/modules/notification/go.mod`

    ```bash
    module go-mma/modules/notification
    
    go 1.24.1
    
    replace go-mma/shared/common v0.0.0 => ../../shared/common
    ```

- โปรเจกต์ customer มีการใช้งาน common, notification

    แก้ไขไฟล์ `src/modules/customer/go.mod`

    ```bash
    module go-mma/modules/customer
    
    go 1.24.1
    
    replace go-mma/shared/common v0.0.0 => ../../shared/common
    
    replace go-mma/modules/notification v0.0.0 => ../../modules/notification
    ```

- โปรเจกต์ order มีการใช้งาน common, notification, customer

    แก้ไขไฟล์ `src/modules/order/go.mod`

    ```bash
    module go-mma/modules/order
    
    go 1.24.1
    
    replace go-mma/shared/common v0.0.0 => ../../shared/common
    
    replace go-mma/modules/notification v0.0.0 => ../../modules/notification
    
    replace go-mma/modules/customer v0.0.0 => ../../modules/customer
    ```

- โปรเจกต์ app มีการใช้งาน common, notification, customer, order

    แก้ไขไฟล์ `src/app/go.mod`

    ```bash
    module go-mma
    
    go 1.24.1
    
    replace go-mma/shared/common v0.0.0 => ../shared/common
    
    replace go-mma/modules/notification v0.0.0 => ../modules/notification
    
    replace go-mma/modules/customer v0.0.0 => ../modules/customer
    
    replace go-mma/modules/order v0.0.0 => ../modules/order
    ```

### สร้าง VS Code Workspace

สำหรับการทำ Mono-Repo ใน VS Code ต้องเปิดแบบ Workspace ถึงจะสามารถทำงานได้ถูกต้อง

- สร้างไฟล์ `go-mma.code-workspace`

    ```bash
    {
     "folders": [
      {
       "path": "."
      },
      {
       "path": "src/app"
      },
      {
       "path": "src/modules/customer"
      },
      {
       "path": "src/modules/order"
      },
      {
       "path": "src/modules/notification"
      },
      {
       "path": "src/shared/common"
      }
     ],
     "settings": {}
    }
    ```

- เลือกที่เมนู File เลือก Open Workspace from file…
- เลือกที่ไฟล์ `go-mma.code-workspace`
- กด Open
- ใน Explorer จะแสดง แบบนี้

    ```bash
    go-mma
    app
    customer
    order
    notification
    common
    ```

### โปรเจกต์ common

- ให้ทำการย้ายโค้ดใน `util` ทั้งหมด ยกเว้น `env` มาไว้ในโปรเจกต์ `common`

    ```bash
    common
    ├── go.mod
    ├── errs
    │   ├── errs.go
    │   ├── helpers.go
    │   └── types.go
    ├── idgen
    │   └── idgen.go
    ├── logger
    │   └── logger.go
    ├── module
    │   └── module.go
    ├── registry
    │   ├── helper.go
    │   └── service_registry.go
    └── storage
        └── sqldb
            ├── sqldb.go
            └── transactor
                ├── nested_transactions_none.go
                ├── nested_transactions_savepoints.go
                ├── transactor.go
                └── types.go
    ```

- ติดตั้ง package ที่ต้องใช้งาน รันคำสั่ง

    ```bash
    go mod tidy
    ```

### โปรเจกต์ notification

- ให้ทำการย้ายโค้ดใน `modules/notification` ทั้งหมด  มาไว้ในโปรเจกต์ `notification`

    ```bash
    notification
    ├── go.mod
    ├── go.sum
    ├── module.go
    └── service
        └── notification.go
    ```

- แก้ไข path ของการ `import` ดังนี้
  - `go-mma/util/logger` → `go-mma/shared/common/logger`
- ติดตั้ง package ที่ต้องใช้งาน รันคำสั่ง

    ```bash
    go mod tidy
    ```

### โปรเจกต์ customer

- ให้ทำการย้ายโค้ดใน `modules/customer` ทั้งหมด  มาไว้ในโปรเจกต์ `customer`

    ```bash
    customer
    ├── dto
    │   ├── customer_request.go
    │   ├── customer_response.go
    │   └── customer.go
    ├── go.mod
    ├── go.sum
    ├── handler
    │   └── customer.go
    ├── internal
    │   ├── model
    │   │   └── customer.go
    │   └── repository
    │       └── customer.go
    ├── module.go
    ├── service
    │   └── customer.go
    └── test
        └── customers.http
    ```

- แก้ไข path ของการ `import` ดังนี้
  - `go-mma/util/errs` → `go-mma/shared/common/errs`
  - `go-mma/util/logger` → `go-mma/shared/common/logger`
  - `go-mma/util/module` → `go-mma/shared/common/module`
  - `go-mma/util/registry` → `go-mma/shared/common/registry`
  - `go-mma/util/storage/sqldb/transactor` → `go-mma/shared/common/storage/sqldb/transactor`
- ติดตั้ง package ที่ต้องใช้งาน รันคำสั่ง

    ```bash
    go mod tidy
    ```

### โปรเจกต์ order

- ให้ทำการย้ายโค้ดใน `modules/order` ทั้งหมด  มาไว้ในโปรเจกต์ `order`

    ```bash
    order
    ├── dto
    │   ├── customer_request.go
    │   ├── customer_response.go
    │   └── customer.go
    ├── go.mod
    ├── go.sum
    ├── handler
    │   └── customer.go
    ├── internal
    │   ├── model
    │   │   └── customer.go
    │   └── repository
    │       └── customer.go
    ├── module.go
    ├── service
    │   └── customer.go
    └── test
        └── customers.http
    ```

- แก้ไข path ของการ `import` ดังนี้
  - `go-mma/util/errs` → `go-mma/shared/common/errs`
  - `go-mma/util/logger` → `go-mma/shared/common/logger`
  - `go-mma/util/module` → `go-mma/shared/common/module`
  - `go-mma/util/registry` → `go-mma/shared/common/registry`
  - `go-mma/util/storage/sqldb/transactor` → `go-mma/shared/common/storage/sqldb/transactor`
- ติดตั้ง package ที่ต้องใช้งาน รันคำสั่ง

    ```bash
    go mod tidy
    ```

### โปรเจกต์ app

- ให้ทำการย้ายโค้ดใน `application`, `cmd`, `config` และ `util`   มาไว้ในโปรเจกต์ `app`

    ```bash
    app
    ├── application
    │   ├── application.go
    │   ├── http.go
    │   └── middleware
    │       ├── request_logger.go
    │       └── response_error.go
    ├── cmd
    │   └── api
    │       └── main.go
    ├── config
    │   └── config.go
    ├── go.mod
    ├── go.sum
    └── util
        └── env
            └── env.go
    ```

- แก้ไข path ของการ `import` ดังนี้
  - `go-mma/util/errs` → `go-mma/shared/common/errs`
  - `go-mma/util/logger` → `go-mma/shared/common/logger`
  - `go-mma/util/module` → `go-mma/shared/common/module`
  - `go-mma/util/registry` → `go-mma/shared/common/registry`
  - `go-mma/util/storage/sqldb` → `go-mma/shared/common/storage/sqldb`
  - `go-mma/util/storage/sqldb/transactor` → `go-mma/shared/common/storage/sqldb/transactor`
- ติดตั้ง package ที่ต้องใช้งาน รันคำสั่ง

    ```bash
    go mod tidy
    ```

### รันโปรแกรม

- แก้ไฟล์ `Makefile` เพื่อแก้ path ในการรัน `main.go`

    ```bash
    .PHONY: run
    run:
     cd src/app && \
     go run cmd/api/main.go
    ```

- ทดลองรันโปรแกรม

    ```bash
    make run
    ```

## Public API contract

จากโค้ดในปัจจุบัน จะเห็นว่าโมดูล order สามารถเรียกใช้ CustomerService สำหรับการสร้างลูกค้าใหม่ได้ด้วย ซึ่งเราไม่ควรที่จะเปิดให้โมดูลอื่นๆ สามารถใช้งานได้ทั้งหมด

ดังนั้นเนื้อหาในส่วนจะพามาทำ Public API contract หรือ ข้อตกลง (interface/contract) ที่ โมดูลหนึ่งเปิดเผยให้โมดูลอื่นใช้ (public use) โดยระบุว่า

- โมดูลนี้ ให้บริการอะไร (method, input, output)
- โมดูลอื่น ควรเรียกใช้อย่างไร
- โดย ไม่เปิดเผยรายละเอียดภายใน (implementation)

ตัวอย่าง

```bash
                        ┌────────────────────────────┐
                        │     customercontract       │
                        │ ┌────────────────────────┐ │
                        │ │  CreditManager         │ │
                        │ │                        │ │
                        │ │ + ReserveCredit()      │ │
                        │ │ + ReleaseCredit()      │ │
                        │ └────────────────────────┘ │
                        └────────────▲───────────────┘
                                     │
        implements                   │  depends on
                                     │
┌────────────────────┐     uses      │   ┌────────────────────┐
│     customer       │───────────────┘   │       order        │
│ ┌────────────────┐ │                   │ ┌─────────────────┐│
│ │ CustomerService│◄────────────────────┤ │ OrderService    ││
│ │ (implements    │ │                   │ │ (depends on     ││
│ │  CreditManager)│ │                   │ │  CreditManager) ││
│ └────────────────┘ │                   │ └─────────────────┘│
└────────────────────┘                   └────────────────────┘
```

### สร้าง Customer Contract

`customer-contract` เป็น โปรเจกต์กลาง ที่เก็บ public interfaces เช่น `CreditManager` ในการสร้างนั้นใช้ 2 หลักการนี้

1. Interface Segregation Principle (ISP) ใช้เพื่อแยก interface ของ `CustomerService` ให้เป็น interface ย่อยๆ
2. เนื่องจากเราทำเป็น mono-repo ดังนั้น จะสร้าง contract เป็นโปรเจกต์แยกออกมาจากโปรเจกต์โมดูล customer เพราะว่า
    - Low Coupling: `order` ไม่ต้อง import logic หรือ dependency ของ `customer` โดยตรง
    - เปลี่ยน implementation ได้อิสระ: เปลี่ยน logic ภายใน `customer` โดยไม่กระทบ `order`
    - Encapsulation: ป้องกันการ import โค้ดภายใน customer ที่ไม่ได้ตั้งใจเปิดเผย

ขั้นตอนการสร้าง customer contract

- สร้างโปรเจกต์ใหม่

    ```bash
    mkdir -p src/shared/contract/customer-contract
    cd src/shared/contract/customer-contract
    go mod init go-mma/shared/contract/customercontract
    ```

- แก้ไขไฟล์ `go.mod`

    ```go
    module go-mma/shared/contract/customercontract
    
    go 1.24.1
    
    replace go-mma/shared/common v0.0.0 => ../../common
    
    require go-mma/shared/common v0.0.0
    ```

- เพิ่มโปรเจกต์เข้า workspace โดยให้แก้ไขไฟล์ `go-mma.code-workspace`

    ```bash
    {
      "folders": [
        {
          "path": "."
        },
        {
          "path": "src/app"
        },
        {
          "path": "src/modules/customer"
        },
        {
          "path": "src/shared/contract/customer-contract"
        },
        {
          "path": "src/modules/order"
        },
        {
          "path": "src/modules/notification"
        },
        {
          "path": "src/shared/common"
        }
      ],
      "settings": {}
    }
    ```

- สร้าง customer contract โดยให้สร้างไฟล์ `src/shared/contract/customer-contract/contract.go`

    ```go
    package customercontract
    
    import (
     "context"
     "go-mma/shared/common/registry"
    )
    
    const (
     CreditManagerKey registry.ServiceKey = "customer:contract:credit"
    )
    
    type CustomerInfo struct {
     ID     int64    `json:"id"`
     Email  string   `json:"email"`
     Credit int      `json:"credit"`
    }
    
    type CustomerReader interface {
     GetCustomerByID(ctx context.Context, id int64) (*CustomerInfo, error)
    }
    
    type CreditManager interface {
     CustomerReader // embed เพื่อ reuse
     ReserveCredit(ctx context.Context, id int64, amount int) error
     ReleaseCredit(ctx context.Context, id int64, amount int) error
    }
    ```

### โมดูล Customer

ต้องปรับให้ `CustomerService` มา implement `customercontract`

- แก้ไขไฟล์ `src/modules/customer/go.mod`

    ```go
    module go-mma/modules/customer
    
    go 1.24.1
    
    replace go-mma/shared/common v0.0.0 => ../../shared/common
    
    replace go-mma/modules/notification v0.0.0 => ../../modules/notification
    
    replace go-mma/shared/contract/customercontract v0.0.0 => ../../shared/contract/customer-contract
    
    require (
     github.com/gofiber/fiber/v3 v3.0.0-beta.4
     go-mma/modules/notification v0.0.0
     go-mma/shared/common v0.0.0
     go-mma/shared/contract/customercontract v0.0.0
    )
    
    // ...
    ```

- แก้ไขไฟล์ `src/modules/customer/service/customer.go`

    ```go
    package service
    
    import (
     "context"
    
      "go-mma/modules/customer/dto"
     "go-mma/modules/customer/internal/model"
     "go-mma/modules/customer/internal/repository"
     "go-mma/shared/common/errs"
     "go-mma/shared/common/logger"
     "go-mma/shared/common/storage/sqldb/transactor"
     "go-mma/shared/contract/customercontract"  // <-- ตรงนี้
    
     notiService "go-mma/modules/notification/service"
    )
    
    // ...
    
    type CustomerService interface {
     CreateCustomer(ctx context.Context, req *dto.CreateCustomerRequest) (*dto.CreateCustomerResponse, error)
     customercontract.CreditManager   // <-- ตรงนี้
    }
    
    // ...
    
    func (s *customerService) CreateCustomer(ctx context.Context, req *customercontract.CreateCustomerRequest) (*customercontract.CreateCustomerResponse, error) { // <-- ตรงนี้
     // ...
    }
    ```

- แก้ไขไฟล์ `src/modules/customer/module.go` เพื่อส่งออก service ตัว key `customercontract.CreditManagerKey`

    ```go
    func (m *moduleImp) Services() []registry.ProvidedService {
     return []registry.ProvidedService{
      {Key: customercontract.CreditManagerKey, Value: m.custSvc},
     }
    }
    ```

### โมดูล Order

รู้จักแค่ interface `CreditManager` ที่มาจาก `customercontract`

- แก้ไขไฟล์ `src/modules/order/go.mod` เพื่อเปลี่ยนจากโมดูล `customer` ไปเป็น `customercontract` แทน

    ```go
    module go-mma/modules/order
    
    go 1.24.1
    
    replace go-mma/shared/common v0.0.0 => ../../shared/common
    
    replace go-mma/modules/notification v0.0.0 => ../../modules/notification
    
    replace go-mma/shared/contract/customercontract v0.0.0 => ../../shared/contract/customer-contract
    
    require (
     github.com/gofiber/fiber/v3 v3.0.0-beta.4
     go-mma/modules/notification v0.0.0
     go-mma/shared/common v0.0.0
     go-mma/shared/contract/customercontract v0.0.0
    )
    
    // ...
    ```

- แก้ไขไฟล์ `src/modules/order/service/order.go` เพื่อเปลี่ยนมาใช้ `customercontract`

    ```go
    package service
    
    import (
     "context"
     "go-mma/modules/order/dto"
     "go-mma/modules/order/internal/model"
     "go-mma/modules/order/internal/repository"
     "go-mma/shared/common/errs"
     "go-mma/shared/common/logger"
     "go-mma/shared/common/storage/sqldb/transactor"
     "go-mma/shared/contract/customercontract" // <-- ตรงนี้
    
     notiService "go-mma/modules/notification/service"
    )
    
    var (
     ErrNoOrderID = errs.ResourceNotFoundError("the order with given id was not found")
    )
    
    type OrderService interface {
     CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) (*dto.CreateOrderResponse, error)
     CancelOrder(ctx context.Context, id int) error
    }
    
    type orderService struct {
     transactor transactor.Transactor
     custSvc    customercontract.CreditManager // <-- ตรงนี้
     orderRepo  repository.OrderRepository
     notiSvc    notiService.NotificationService
    }
    
    func NewOrderService(
     transactor transactor.Transactor,
     custSvc customercontract.CreditManager, // <-- ตรงนี้
     orderRepo repository.OrderRepository,
     notiSvc notiService.NotificationService) OrderService {
     return &orderService{
      transactor: transactor,
      custSvc:    custSvc,
      orderRepo:  orderRepo,
      notiSvc:    notiSvc,
     }
    }
    
    // ...
    ```

- แก้ไขไฟล์ `src/modules/order/module.go` เพื่อเปลี่ยนมาใช้ `customercontract`

    ```go
    package order
    
    import (
     "go-mma/modules/order/handler"
     "go-mma/modules/order/internal/repository"
     "go-mma/modules/order/service"
     "go-mma/shared/common/module"
     "go-mma/shared/common/registry"
     "go-mma/shared/contract/customercontract" // <-- ตรงนี้
    
     notiModule "go-mma/modules/notification"
     notiService "go-mma/modules/notification/service"
    
     "github.com/gofiber/fiber/v3"
    )
    
    // ...
    
    func (m *moduleImp) Init(reg registry.ServiceRegistry) error {
     // Resolve CustomerService from the registry
     custSvc, err := registry.ResolveAs[customercontract.CreditManager](reg, customercontract.CreditManagerKey) // <-- ตรงนี้
     if err != nil {
      return err
     }
      
      // ...
    }
    ```

### รันโปรแกรม

ก่อนจะรันโปรแกรมต้องทำให้โปรเจกต์ `app` รู้จัก `customercontract`  ด้วย

- แก้ไขไฟล์ `src/app/go.mod`

    ```go
    module go-mma
    
    go 1.24.1
    
    replace go-mma/shared/common v0.0.0 => ../shared/common
    
    replace go-mma/modules/notification v0.0.0 => ../modules/notification
    
    replace go-mma/modules/customer v0.0.0 => ../modules/customer
    
    replace go-mma/modules/order v0.0.0 => ../modules/order
    
    replace go-mma/shared/contract/customercontract v0.0.0 => ../shared/contract/customer-contract
    
    // ...
    ```

- รันคำสั่ง `go mod tidy`
- รันโปรแกรม `make run`

## จัดโครงสร้างโมดูลแยกตาม feature

โครงสร้างเดิมของโมดูล customer นั้นจะรวมการทำงานทุกอย่างไว้ที่ interface `CustomerService` ซึ่งควรแยกออกมาเป็น feature ออกจากกันให้ชัดเจน โดยจะใช้

- CQRS (Command Query Responsibility Segregation): เพื่อแยกการ *เขียนข้อมูล (Command)* และ *อ่านข้อมูล (Query)* ออกจากกัน เพื่อให้โค้ดแต่ละส่วนมีความชัดเจน และสามารถปรับแต่ง/สเกลได้แยกจากกัน
- Medaitor Pattern: เพื่อใช้เป็นตัวกลางในการเรียกใช้ Command และ Query แทน Service Registgry

### Medaitor

สร้างตัวจัดการ `Request/Response` ของแต่ละการเขียนข้อมูล (Command) และ อ่านข้อมูล (Query)

- สร้างไฟล์ `src/common/mediator/mediator.go`

    ```go
    package mediator
    
    import (
     "context"
     "errors"
     "fmt"
     "reflect"
    )
    
    // You can define an empty struct to represent no response.
    type NoResponse struct{}
    
    type RequestHandler[TRequest any, TResponse any] interface {
     Handle(ctx context.Context, request TRequest) (TResponse, error)
    }
    
    var handlers = map[reflect.Type]func(ctx context.Context, req interface{}) (interface{}, error){}
    
    // Register adds a handler for a specific request type.
    func Register[TRequest any, TResponse any](handler RequestHandler[TRequest, TResponse]) {
     // Create a zero value to extract the type.
     var req TRequest
     reqType := reflect.TypeOf(req)
    
     // Wrap the handler's Handle method in a function that accepts an empty interface.
     handlers[reqType] = func(ctx context.Context, request interface{}) (interface{}, error) {
      typedReq, ok := request.(TRequest)
      if !ok {
       return nil, errors.New("invalid request type")
      }
      return handler.Handle(ctx, typedReq)
     }
    }
    
    // Send dispatches the request to the registered handler.
    func Send[TRequest any, TResponse any](ctx context.Context, req TRequest) (TResponse, error) {
     reqType := reflect.TypeOf(req)
     handler, ok := handlers[reqType]
     if !ok {
      var empty TResponse
      return empty, fmt.Errorf("no handler for request %T", req)
     }
    
     result, err := handler(ctx, req)
     if err != nil {
      var empty TResponse
      return empty, err
     }
    
     typedRes, ok := result.(TResponse)
     if !ok {
      var empty TResponse
      return empty, errors.New("invalid response type")
     }
    
     return typedRes, nil
    }
    ```

### Customer Features

จาก `CustomerService` เดิม เราจะเอามาเขียนแยกเป็น ได้ดังนี้

1. **create**: สร้างลูกค้าใหม่
2. **get-by-id**: ค้นหาลูกค้าจาก ID
3. **reserve-credit**: ตัดยอด credit
4. **release-credit**: คืนยอด credit

โดยจัดวางโครงสร้างใหม่แบบนี้

```bash
customer
├── domainerrors
│   └── domainerrors.go             # ไว้รวบรวม error ทั้่งหมด ของ customer
├── internal
│   ├── feature                     # สร้างใน internal ป้องกันไม่ให้ import
│   │   ├── create
│   │   │   ├── dto.go              # ย้าย dto มาที่นี่
│   │   │   ├── endpoint.go         # ย้าย http handler มาที่นี่
│   │   │   ├── command.go          # กำหนดรูปแบบของ Request/Response ของ command
│   │   │   └── command_handler.go  # จัดการ command handler
│   │   ├── get-by-id
│   │   │   └── query_handler.go    # จัดการ query handler
│   │   ├── release-credit
│   │   │   └── command_handler.go
│   │   └── reserve-credit
│   │       └── command_handler.go
│   ├── model
│   │   └── customer.go
│   └── repository
│       └── customer.go
├── test
│   └── customers.http
├── module.go          # เปลี่ยนจาก register service เป็น command/query handler แทน
├── go.mod
└── go.sum
```

### Customer Domain Error

เริ่มจากรวบรวม error ที่จะเกิดขึ้นจาก command handler, query handler และ rich model มาไว้ที่นี่

สร้างไฟล์ `customer/domainerrors/domainerrors.go`

```go
package domainerrors

import "go-mma/shared/common/errs"

var (
 ErrCreditValue        = errs.BusinessRuleError("credit must be greater than 0")
 ErrEmailExists        = errs.ConflictError("email already exists")
 ErrCustomerNotFound   = errs.ResourceNotFoundError("the customer with given id was not found")
 ErrInsufficientCredit = errs.BusinessRuleError("insufficient credit")
)
```

### ฟีเจอร์ **get-by-id**: ค้นหาลูกค้าจาก ID

ฟีเจอร์์นี้ เป็นการค้นหาข้อมูลลูกค้า จัดเป็น Query ตาม CQRS และมีการเรียกใช้ในโมดูล order ด้วย ให้เริ่มจากสร้าง contract ขึ้นมาก่อน

<aside>
💡

ให้ลบไฟล์ `customer-contract/contract.go` เพราะจะเปลี่ยนจาก interface ของ public api contract เป็น struct ของ command กับ query แทน

</aside>

- สร้างไฟล์ `customer-contract/query_customer_by_id.go`

    ```go
    package customercontract
    
    type GetCustomerByIDQuery struct {
     ID int64 `json:"id"`
    }
    
    type GetCustomerByIDQueryResult struct {
     ID     int64    `json:"id"`
     Email  string   `json:"email"`
     Credit int      `json:"credit"`
    }
    ```

- สร้างฟีเจอร์ get-by-id

    สร้างไฟล์ `customer/internal/feature/get-by-id/query_handler.go`

    ```go
    package getbyid
    
    import (
     "context"
     "go-mma/modules/customer/domainerrors"
     "go-mma/modules/customer/internal/model"
     "go-mma/modules/customer/internal/repository"
     "go-mma/shared/contract/customercontract"
    )
    
    type getCustomerByIDQueryHandler struct {
     custRepo repository.CustomerRepository
    }
    
    func NewGetCustomerByIDQueryHandler(custRepo repository.CustomerRepository) *getCustomerByIDQueryHandler {
     return &getCustomerByIDQueryHandler{
      custRepo: custRepo,
     }
    }
    
    func (h *getCustomerByIDQueryHandler) Handle(ctx context.Context, query *customercontract.GetCustomerByIDQuery) (*customercontract.GetCustomerByIDQueryResult, error) {
     customer, err := h.custRepo.FindByID(ctx, query.ID)
     if err != nil {
      return nil, err
     }
     if customer == nil {
      return nil, domainerrors.ErrCustomerNotFound
     }
     return h.newGetCustomerByIDQueryResult(customer), nil
    }
    
    func (h *getCustomerByIDQueryHandler) newGetCustomerByIDQueryResult(customer *model.Customer) *customercontract.GetCustomerByIDQueryResult {
     return &customercontract.GetCustomerByIDQueryResult{
      ID:     customer.ID,
      Email:  customer.Email,
      Credit: customer.Credit,
     }
    }
    ```

### ฟีเจอร์ **reserve-credit**: ตัดยอด credit

ฟีเจอร์์นี้ เป็นการตัดยอด credit ซึ่งเป็นการอัพเดทค่าในฐานข้อมูล จัดเป็น Command ตาม CQRS และมีการเรียกใช้ในโมดูล order ด้วย ให้เริ่มจากสร้าง contract ขึ้นมาก่อน

- สร้างไฟล์ `customer-contract/command_reserve_credit.go`

    ```go
    package customercontract
    
    type ReserveCreditCommand struct {
     CustomerID   int64 `json:"customer_id"`
     CreditAmount int   `json:"credit_amount"`
    }
    ```

- สร้างฟีเจอร์ reserve-credit

    สร้างไฟล์ `customer/internal/feature/reserve-credit/command_handler.go`

    ```go
    package reservecredit
    
    import (
     "context"
     "go-mma/modules/customer/domainerrors"
     "go-mma/modules/customer/internal/repository"
     "go-mma/shared/common/errs"
     "go-mma/shared/common/logger"
     "go-mma/shared/common/mediator"
     "go-mma/shared/common/storage/sqldb/transactor"
     "go-mma/shared/contract/customercontract"
    )
    
    type reserveCreditCommandHandler struct {
     transactor transactor.Transactor
     custRepo   repository.CustomerRepository
    }
    
    func NewReserveCreditCommandHandler(
     transactor transactor.Transactor,
     repo repository.CustomerRepository) *reserveCreditCommandHandler {
     return &reserveCreditCommandHandler{
      transactor: transactor,
      custRepo:   repo,
     }
    }
    
    func (h *reserveCreditCommandHandler) Handle(ctx context.Context, cmd *customercontract.ReserveCreditCommand) (*mediator.NoResponse, error) {
     err := h.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
      customer, err := h.custRepo.FindByID(ctx, cmd.CustomerID)
      if err != nil {
       logger.Log.Error(err.Error())
       return err
      }
    
      if customer == nil {
       return domainerrors.ErrCustomerNotFound
      }
    
      if err := customer.ReserveCredit(cmd.CreditAmount); err != nil {
       return err
      }
    
      if err := h.custRepo.UpdateCredit(ctx, customer); err != nil {
       logger.Log.Error(err.Error())
       return errs.DatabaseFailureError(err.Error())
      }
    
      return nil
     })
    
     return nil, err
    }
    ```

### ฟีเจอร์ **release-credit**: คืนยอด credit

ฟีเจอร์์นี้ เป็นการคืนยอด credit ซึ่งเป็นการอัพเดทค่าในฐานข้อมูล จัดเป็น Command ตาม CQRS และมีการเรียกใช้ในโมดูล order ด้วย ให้เริ่มจากสร้าง contract ขึ้นมาก่อน

- สร้างไฟล์ `customer-contract/command_release_credit.go`

    ```go
    package customercontract
    
    type ReleaseCreditCommand struct {
     CustomerID   int64 `json:"customer_id"`
     CreditAmount int   `json:"credit_amount"`
    }
    ```

- สร้างฟีเจอร์ release-credit

    สร้างไฟล์ `customer/internal/feature/release-credit/command_handler.go`

    ```go
    package releasecredit
    
    import (
     "context"
     "go-mma/modules/customer/domainerrors"
     "go-mma/modules/customer/internal/repository"
     "go-mma/shared/common/logger"
     "go-mma/shared/common/mediator"
     "go-mma/shared/common/storage/sqldb/transactor"
     "go-mma/shared/contract/customercontract"
    )
    
    type releaseCreditCommandHandler struct {
     transactor transactor.Transactor
     custRepo   repository.CustomerRepository
    }
    
    func NewReleaseCreditCommandHandler(
     transactor transactor.Transactor,
     repo repository.CustomerRepository) *releaseCreditCommandHandler {
     return &releaseCreditCommandHandler{
      transactor: transactor,
      custRepo:   repo,
     }
    }
    
    func (h *releaseCreditCommandHandler) Handle(ctx context.Context, cmd *customercontract.ReleaseCreditCommand) (*mediator.NoResponse, error) {
     err := h.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
      customer, err := h.custRepo.FindByID(ctx, cmd.CustomerID)
      if err != nil {
       logger.Log.Error(err.Error())
       return err
      }
    
      if customer == nil {
       return domainerrors.ErrCustomerNotFound
      }
    
      customer.ReleaseCredit(cmd.CreditAmount)
    
      if err := h.custRepo.UpdateCredit(ctx, customer); err != nil {
       logger.Log.Error(err.Error())
       return err
      }
    
      return nil
     })
    
     return nil, err
    }
    ```

### ฟีเจอร์ **create**: สร้างลูกค้าใหม่

ฟีเจอร์์นี้ เป็นการบันทึกข้อมูลลูกค้าใหม่ลงในฐานข้อมูล จัดเป็น Command ตาม CQRS และไม่มีการเรียกใช้ที่โมดูลอื่น จึงไม่จำเป็นต้องมี contract

- เนื่องฟีเจอร์นี้จะมีการเรียกใช้งานผ่าน REST API จึงต้องมี endpoint สำหรับจัดการ request/response ด้วย เริ่มจากย้าย `dto` มาไว้ที่นี้

    สร้างไฟล์ `customer/internal/feature/create/dto.go`

    ```go
    package create
    
    import (
     "errors"
     "net/mail"
    )
    
    type CreateCustomerRequest struct {
     Email  string `json:"email"`
     Credit int    `json:"credit"`
    }
    
    func (r *CreateCustomerRequest) Validate() error {
     var errs error
     if r.Email == "" {
      errs = errors.Join(errs, errors.New("email is required"))
     }
     if _, err := mail.ParseAddress(r.Email); err != nil {
      errs = errors.Join(errs, errors.New("email is invalid"))
     }
     if r.Credit <= 0 {
      errs = errors.Join(errs, errors.New("credit must be greater than 0"))
     }
     return errs
    }
    
    type CreateCustomerResponse struct {
     ID int64 `json:"id"`
    }
    ```

- ออกแบบ Command สำหรับฟีเจอร์ create

    สร้างไฟล์ `customer/internal/feature/create/command.go`

    ```go
    package create
    
    type CreateCustomerCommand struct {
     CreateCustomerRequest  // embeded type มาเพราะหน้าตาเหมือนกัน
    }
    
    type CreateCustomerCommandResult struct {
     CreateCustomerResponse // embeded type มาเพราะหน้าตาเหมือนกัน
    }
    
    // ฟังก์ชันช่วยสร้าง CreateCustomerCommandResult
    func NewCreateCustomerCommandResult(id int) *CreateCustomerCommandResult {
     return &CreateCustomerCommandResult{
      CreateCustomerResponse{
       ID: id,
      },
     }
    }
    ```

- สร้างฟีเจอร์ create

    สร้างไฟล์ `customer/internal/feature/create/command_handler.go`

    ```go
    package create
    
    import (
     "context"
     "go-mma/modules/customer/domainerrors"
     "go-mma/modules/customer/internal/model"
     "go-mma/modules/customer/internal/repository"
     "go-mma/shared/common/logger"
     "go-mma/shared/common/storage/sqldb/transactor"
    
     notiService "go-mma/modules/notification/service"
    )
    
    type createCustomerCommandHandler struct {
     transactor transactor.Transactor
     custRepo   repository.CustomerRepository
     notiSvc    notiService.NotificationService
    }
    
    func NewCreateCustomerCommandHandler(
     transactor transactor.Transactor,
     custRepo repository.CustomerRepository,
     notiSvc notiService.NotificationService) *createCustomerCommandHandler {
     return &createCustomerCommandHandler{
      transactor: transactor,
      custRepo:   custRepo,
      notiSvc:    notiSvc,
     }
    }
    
    func (h *createCustomerCommandHandler) Handle(ctx context.Context, cmd *CreateCustomerCommand) (*CreateCustomerCommandResult, error) {
     // ตรวจสอบ business rule/invariant
     if err := h.validateBusinessInvariant(ctx, cmd); err != nil {
      return nil, err
     }
    
     // แปลง Command → Model
     customer := model.NewCustomer(cmd.Email, cmd.Credit)
    
     // ย้ายส่วนที่ติดต่อฐานข้อมูล กับส่งอีเมลมาทำงานใน WithinTransaction
     err := h.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
    
      // ส่งไปที่ Repository Layer เพื่อบันทึกข้อมูลลงฐานข้อมูล
      if err := h.custRepo.Create(ctx, customer); err != nil {
       // error logging
       logger.Log.Error(err.Error())
       return err
      }
    
      // ส่งอีเมลต้อนรับ
      if err := h.notiSvc.SendEmail(customer.Email, "Welcome to our service!", map[string]any{
       "message": "Thank you for joining us! We are excited to have you as a member.",
      }); err != nil {
       // error logging
       logger.Log.Error(err.Error())
       return err
      }
    
      return nil
     })
    
     if err != nil {
      return nil, err
     }
    
     return NewCreateCustomerCommandResult(customer.ID), nil
    }
    
    func (h *createCustomerCommandHandler) validateBusinessInvariant(ctx context.Context, cmd *CreateCustomerCommand) error {
     // ตรวจสอบ Credit ต้องมากกว่า 0
     if cmd.Credit <= 0 {
      return domainerrors.ErrCreditValue
     }
    
     // ตรวจสอบ email ซ้ำ
     exists, err := h.custRepo.ExistsByEmail(ctx, cmd.Email)
     if err != nil {
      // error logging
      logger.Log.Error(err.Error())
      return err
     }
    
     if exists {
      return domainerrors.ErrEmailExists
     }
     return nil
    }
    ```

- สร้าง endpoint ของฟีเจอร์นี้

    สร้างไฟล์ `customer/internal/feature/create/endpoint.go`

    ```go
    package create
    
    import (
     "go-mma/shared/common/errs"
     "go-mma/shared/common/mediator"
     "strings"
    
     "github.com/gofiber/fiber/v3"
    )
    
    func NewEndpoint(router fiber.Router, path string) {
     router.Post(path, createCustomerHTTPHandler)
    }
    
    func createCustomerHTTPHandler(c fiber.Ctx) error {
     // 1. รับ request body มาเป็น DTO
     var req CreateCustomerRequest
     if err := c.Bind().Body(&req); err != nil {
      return errs.InputValidationError(err.Error())
     }
    
     // 2. ตรวจสอบความถูกต้อง (validate)
     if err := req.Validate(); err != nil {
      return errs.InputValidationError(strings.Join(strings.Split(err.Error(), "\n"), ", "))
     }
    
     // 3. ส่งไปที่ Command Handler
     resp, err := mediator.Send[*CreateCustomerCommand, *CreateCustomerCommandResult](
      c.Context(),
      &CreateCustomerCommand{CreateCustomerRequest: req},
     )
    
     // 4. จัดการ error จาก feature หากเกิดขึ้น
     if err != nil {
      return err
     }
    
     // 5. ตอบกลับ client
     return c.Status(fiber.StatusCreated).JSON(resp)
    }
    ```

### โมดูล Customer

จากเดิมใน `customer/module.go` จะมีการระบุว่าต้องการเปิด service อะไรให้ใช้งานบ้าง เราจะเอาตรงนี้ โดยจะใช้ mediator มาจัดการแทน

```go
package customer

import (
 "go-mma/modules/customer/internal/feature/create"
 getbyid "go-mma/modules/customer/internal/feature/get-by-id"
 releasecredit "go-mma/modules/customer/internal/feature/release-credit"
 reservecredit "go-mma/modules/customer/internal/feature/reserve-credit"
 "go-mma/modules/customer/internal/repository"
 "go-mma/shared/common/mediator"
 "go-mma/shared/common/module"
 "go-mma/shared/common/registry"

 notiModule "go-mma/modules/notification"
 notiService "go-mma/modules/notification/service"

 "github.com/gofiber/fiber/v3"
)

func NewModule(mCtx *module.ModuleContext) module.Module {
 return &moduleImp{mCtx: mCtx}
}

type moduleImp struct {
 mCtx *module.ModuleContext
 // เอา service ออก
}

func (m *moduleImp) APIVersion() string {
 return "v1"
}

func (m *moduleImp) Init(reg registry.ServiceRegistry) error {
 // Resolve NotificationService from the registry
 notiSvc, err := registry.ResolveAs[notiService.NotificationService](reg, notiModule.NotificationServiceKey)
 if err != nil {
  return err
 }

 repo := repository.NewCustomerRepository(m.mCtx.DBCtx)

  // <-- ตรงนี้
  // ให้ทำการ register handler เข้า mediator
 mediator.Register(create.NewCreateCustomerCommandHandler(m.mCtx.Transactor, repo, notiSvc))
 mediator.Register(getbyid.NewGetCustomerByIDQueryHandler(repo))
 mediator.Register(reservecredit.NewReserveCreditCommandHandler(m.mCtx.Transactor, repo))
 mediator.Register(releasecredit.NewReleaseCreditCommandHandler(m.mCtx.Transactor, repo))

 return nil
}

// ลบ Services() []registry.ProvidedService ออก

func (m *moduleImp) RegisterRoutes(router fiber.Router) {
 customers := router.Group("/customers")
 create.NewEndpoint(customers, "")
}
```

### ปรับแก้โมดูล Order

ปรับโมดูล Order ให้เรียกใช้ Command/Query ของโมดูล Customer แทนการเรียกจาก service

เริ่มจากแยก OrderService เป็นฟีเจอร์

```bash
order
├── domainerrors
│   └── domainerrors.go             # ไว้รวบรวม error ทั้่งหมด ของ order
├── internal
│   ├── feature                     # สร้างใน internal ป้องกันไม่ให้ import
│   │   ├── create
│   │   │   ├── dto.go              # ย้าย dto มาที่นี่
│   │   │   ├── endpoint.go         # ย้าย http handler มาที่นี่
│   │   │   ├── command.go          # กำหนดรูปแบบของ Request/Response ของ command
│   │   │   └── command_handler.go  # จัดการ command handler
│   │   └── cancel
│   │       ├── dto.go              # ย้าย dto มาที่นี่
│   │       ├── endpoint.go         # ย้าย http handler มาที่นี่
│   │       ├── command.go          # กำหนดรูปแบบของ Request/Response ของ command
│   │       └── command_handler.go  # จัดการ command handler
│   ├── model
│   │   └── order.go
│   └── repository
│       └── order.go
├── test
│   └── orders.http
├── module.go                        # register command/query handler
├── go.mod
└── go.sum
```

- สร้างไฟล์ `order/domainerrors/domainerrors.go`

    ```go
    package domainerrors
    
    import "go-mma/shared/common/errs"
    
    var (
     ErrNoOrderID = errs.ResourceNotFoundError("the order with given id was not found")
    )
    ```

- สร้างฟีเจอร์ create สำหรับสร้างออเดอร์ใหม่ และมีการเรียกใช้งานผ่าน REST API

    สร้างไฟล์ `order/internal/feature/create/dto.go`

    ```go
    package create
    
    import "fmt"
    
    type CreateOrderRequest struct {
     CustomerID int64 `json:"customer_id"`
     OrderTotal int   `json:"order_total"`
    }
    
    func (r *CreateOrderRequest) Validate() error {
     if r.CustomerID <= 0 {
      return fmt.Errorf("customer_id is required")
     }
     if r.OrderTotal <= 0 {
      return fmt.Errorf("order_total must be greater than 0")
     }
     return nil
    }
    
    type CreateOrderResponse struct {
     ID int64 `json:"id"`
    }
    ```

    สร้างไฟล์ `order/internal/feature/create/command.go`

    ```go
    package create
    
    type CreateOrderCommand struct {
     CreateOrderRequest
    }
    
    type CreateOrderCommandResult struct {
     CreateOrderResponse
    }
    
    func NewCreateOrderCommandResult(id int64) *CreateOrderCommandResult {
     return &CreateOrderCommandResult{
      CreateOrderResponse{ID: id},
     }
    }
    ```

    สร้างไฟล์ `order/internal/feature/create/command_handler.go`

    ```go
    package create
    
    import (
     "context"
     "go-mma/modules/order/internal/model"
     "go-mma/modules/order/internal/repository"
     "go-mma/shared/common/logger"
     "go-mma/shared/common/mediator"
     "go-mma/shared/common/storage/sqldb/transactor"
     "go-mma/shared/contract/customercontract"
    
     notiService "go-mma/modules/notification/service"
    )
    
    type createOrderCommandHandler struct {
     transactor transactor.Transactor
     orderRepo  repository.OrderRepository
     notiSvc    notiService.NotificationService
    }
    
    func NewCreateOrderCommandHandler(
     transactor transactor.Transactor,
     orderRepo repository.OrderRepository,
     notiSvc notiService.NotificationService) *createOrderCommandHandler {
     return &createOrderCommandHandler{
      transactor: transactor,
      orderRepo: orderRepo,
      notiSvc:   notiSvc,
     }
    }
    
    func (h *createOrderCommandHandler) Handle(ctx context.Context, cmd *CreateOrderCommand) (*CreateOrderCommandResult, error) {
     // ตรวจสอบ customer id
     customer, err := mediator.Send[*customercontract.GetCustomerByIDQuery, *customercontract.GetCustomerByIDQueryResult](
      ctx,
      &customercontract.GetCustomerByIDQuery{ID: cmd.CustomerID},
     )
     if err != nil {
      return nil, err
     }
    
     var order *model.Order
     err = h.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
    
      // ตัดยอด credit ในตาราง customer
      if _, err := mediator.Send[*customercontract.ReserveCreditCommand, *mediator.NoResponse](
       ctx,
       &customercontract.ReserveCreditCommand{CustomerID: cmd.CustomerID, CreditAmount: cmd.OrderTotal},
      ); err != nil {
       return err
      }
    
      // สร้าง order ใหม่
      order = model.NewOrder(cmd.CustomerID, cmd.OrderTotal)
      err := h.orderRepo.Create(ctx, order)
      if err != nil {
       logger.Log.Error(err.Error())
       return err
      }
    
      err = h.notiSvc.SendEmail(customer.Email, "Order Created", map[string]any{
       "order_id": order.ID,
       "total":    order.OrderTotal,
      })
      if err != nil {
       logger.Log.Error(err.Error())
       return err
      }
    
      return nil
     })
    
     if err != nil {
      return nil, err
     }
    
     return NewCreateOrderCommandResult(order.ID), nil
    }
    ```

    สร้างไฟล์ `order/internal/feature/create/endpoint.go`

    ```go
    package create
    
    import (
     "go-mma/shared/common/errs"
     "go-mma/shared/common/mediator"
     "strings"
    
     "github.com/gofiber/fiber/v3"
    )
    
    func NewEndpoint(router fiber.Router, path string) {
     router.Post(path, createOrderHTTPHandler)
    }
    
    func createOrderHTTPHandler(c fiber.Ctx) error {
     // 1. รับ request body มาเป็น DTO
     var req CreateOrderRequest
     if err := c.Bind().Body(&req); err != nil {
      return errs.InputValidationError(err.Error())
     }
    
     // 2. ตรวจสอบความถูกต้อง (validate)
     if err := req.Validate(); err != nil {
      return errs.InputValidationError(strings.Join(strings.Split(err.Error(), "\n"), ", "))
     }
    
     // 3. ส่งไปที่ Command Handler
     resp, err := mediator.Send[*CreateOrderCommand, *CreateOrderCommandResult](
      c.Context(),
      &CreateOrderCommand{CreateOrderRequest: req},
     )
    
     // 4. จัดการ error จาก feature หากเกิดขึ้น
     if err != nil {
      return err
     }
    
     // 5. ตอบกลับ client
     return c.Status(fiber.StatusCreated).JSON(resp)
    }
    ```

- สร้างฟีเจอร์ cancel สำหรับยกเลิกออเดอร์ และมีการเรียกใช้งานผ่าน REST API

    สร้างไฟล์ `order/internal/feature/cancel/command.go`

    ```go
    package cancel
    
    type CancelOrderCommand struct {
     ID int64 `json:"id"`
    }
    ```

    สร้างไฟล์ `order/internal/feature/cancel/command_handler.go`

    ```go
    package create
    
    import (
     "context"
     "go-mma/modules/order/internal/model"
     "go-mma/modules/order/internal/repository"
     "go-mma/shared/common/logger"
     "go-mma/shared/common/mediator"
     "go-mma/shared/common/storage/sqldb/transactor"
     "go-mma/shared/contract/customercontract"
    
     notiService "go-mma/modules/notification/service"
    )
    
    type createOrderCommandHandler struct {
     transactor transactor.Transactor
     orderRepo  repository.OrderRepository
     notiSvc    notiService.NotificationService
    }
    
    func NewCreateOrderCommandHandler(
     transactor transactor.Transactor,
     orderRepo repository.OrderRepository,
     notiSvc notiService.NotificationService) *createOrderCommandHandler {
     return &createOrderCommandHandler{
      transactor: transactor,
      // custSvc:    custSvc,
      orderRepo: orderRepo,
      notiSvc:   notiSvc,
     }
    }
    
    func (h *createOrderCommandHandler) Handle(ctx context.Context, cmd *CreateOrderCommand) (*CreateOrderCommandResult, error) {
     // ตรวจสอบ customer id
     customer, err := mediator.Send[*customercontract.GetCustomerByIDQuery, *customercontract.GetCustomerByIDQueryResult](
      ctx,
      &customercontract.GetCustomerByIDQuery{ID: cmd.CustomerID},
     )
     if err != nil {
      return nil, err
     }
    
     var order *model.Order
     err = h.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
    
      // ตัดยอด credit ในตาราง customer
      if _, err := mediator.Send[*customercontract.ReserveCreditCommand, *mediator.NoResponse](
       ctx,
       &customercontract.ReserveCreditCommand{CustomerID: cmd.CustomerID, CreditAmount: cmd.OrderTotal},
      ); err != nil {
       return err
      }
    
      // สร้าง order ใหม่
      order = model.NewOrder(cmd.CustomerID, cmd.OrderTotal)
      err := h.orderRepo.Create(ctx, order)
      if err != nil {
       logger.Log.Error(err.Error())
       return err
      }
    
      err = h.notiSvc.SendEmail(customer.Email, "Order Created", map[string]any{
       "order_id": order.ID,
       "total":    order.OrderTotal,
      })
      if err != nil {
       logger.Log.Error(err.Error())
       return err
      }
    
      return nil
     })
    
     if err != nil {
      return nil, err
     }
    
     return NewCreateOrderCommandResult(order.ID), nil
    }
    ```

    สร้างไฟล์ `order/internal/feature/cancel/endpoint.go`

    ```go
    package cancel
    
    import (
     "go-mma/shared/common/errs"
     "go-mma/shared/common/mediator"
     "strconv"
    
     "github.com/gofiber/fiber/v3"
    )
    
    func NewEndpoint(router fiber.Router, path string) {
     router.Delete(path, cancelOrderHTTPHandler)
    }
    
    func cancelOrderHTTPHandler(c fiber.Ctx) error {
     // 1. อ่านค่า id จาก path param
     id := c.Params("orderID")
    
     // 2. ตรวจสอบรูปแบบ order id
     orderID, err := strconv.Atoi(id)
     if err != nil {
      return errs.InputValidationError("invalid order id")
     }
    
     // 3. ส่งไปที่ Command Handler
     _, err = mediator.Send[*CancelOrderCommand, *mediator.NoResponse](
      c.Context(),
      &CancelOrderCommand{ID: int64(orderID)},
     )
    
     // 4. จัดการ error จาก feature หากเกิดขึ้น
     if err != nil {
      return err
     }
    
     // 5. ตอบกลับ client
     return c.SendStatus(fiber.StatusNoContent)
    }
    ```

- เพิ่มการ register command handlers ทั้งหมด ใน `order/module.go`

    ```go
    package order
    
    import (
     "go-mma/modules/order/internal/feature/cancel"
     "go-mma/modules/order/internal/feature/create"
     "go-mma/modules/order/internal/repository"
     "go-mma/shared/common/mediator"
     "go-mma/shared/common/module"
     "go-mma/shared/common/registry"
    
     notiModule "go-mma/modules/notification"
     notiService "go-mma/modules/notification/service"
    
     "github.com/gofiber/fiber/v3"
    )
    
    func NewModule(mCtx *module.ModuleContext) module.Module {
     return &moduleImp{mCtx: mCtx}
    }
    
    type moduleImp struct {
     mCtx *module.ModuleContext
    }
    
    func (m *moduleImp) APIVersion() string {
     return "v1"
    }
    
    func (m *moduleImp) Init(reg registry.ServiceRegistry) error {
    
     // Resolve NotificationService from the registry
     notiSvc, err := registry.ResolveAs[notiService.NotificationService](reg, notiModule.NotificationServiceKey)
     if err != nil {
      return err
     }
    
     repo := repository.NewOrderRepository(m.mCtx.DBCtx)
    
     mediator.Register(create.NewCreateOrderCommandHandler(m.mCtx.Transactor, repo, notiSvc))
     mediator.Register(cancel.NewCancelOrderCommandHandler(m.mCtx.Transactor, repo))
    
     return nil
    }
    
    func (m *moduleImp) RegisterRoutes(router fiber.Router) {
     orders := router.Group("/orders")
     create.NewEndpoint(orders, "")
     cancel.NewEndpoint(orders, "/:orderID")
    }
    ```

### โมดูล Notification

ขอข้ามการแปลงโมดูล notification ไปก่อน

## Event-Driven Architecture

จากโค้ดปัจจุบัน เช่น  ใน logic ของ `CreateCustomerCommandHandler` ที่ใช้สร้างลูกค้าใหม่ โดยในตอนนี้โค้ดเป็นการเขียนในรูปแบบ Imperative Style คือสั่งให้ `SendEmail` โดยตรง

```bash
+-------------------------------+
| CreateCustomerCommandHandler  |
+-------------------------------+
           |
           | creates Customer
           v
+-------------------------------+
|  CustomerRepository           |
+-------------------------------+
           |
           | persists to DB
           v
+-------------------------------+
|  NotificationService          |
+-------------------------------+
           |
           | sends welcome email
           v
       External System
```

ซึ่งมี coupling สูง และไม่สามารถแยก concern ได้ชัดเจน

โดยเราจะเอาแนวคิด Event-Driven Architecture (EDA) ที่เป็นการออกแบบซอฟต์แวร์โดยใช้ “เหตุการณ์ (Event)” เป็นแกนกลางในการสื่อสารระหว่างส่วนต่างๆ ของระบบ ซึ่งแต่ละส่วนจะ ตอบสนองต่อเหตุการณ์แทนที่จะเรียกกันโดยตรง ทำให้ระบบ หลวมตัว (loosely coupled) และขยายตัวได้ง่าย ซึ่งจะมี domain events กับ integration events ([อ่านเพิ่มเติม](https://somprasongd.work/blog/architecture/domain-event-vs-integration-event))

### Domain Events

- เป็น event ที่เกิดภายใน domain (bounded context)
- ใช้เพื่อแจ้งว่า *"สิ่งนี้เกิดขึ้นแล้ว"* (เช่น `CustomerCreated`)
- ถูก publish และ consume *ในกระบวนการเดียวกัน* (in-process)
- มักใช้กับ business logic ภายใน

### Integration Events

- ถูกใช้เพื่อสื่อสารข้าม bounded context / microservice
- ใช้ messaging system เช่น Kafka, RabbitMQ
- มักเกิดจาก domain event แล้วถูกแปลง (map) เป็น integration event
- ทำงานแบบ async

### โครงสร้างหลังเพิ่ม Domain Events กับ Integration Events

ตัวอย่าง การแยก logic การส่งอีเมลออกจาก Handler และรองรับการทำงานแบบ asynchronous ซึ่งจะมีขั้นตอนการทำงานแบบนี้

```bash
+-------------------------------+
| CreateCustomerCommandHandler  |
+-------------------------------+
           |
           | creates Customer
           v
+-------------------------+
|  Customer Entity        |
|  + AddDomainEvent()     |
+-------------------------+
           |
           | emits domain event
           v
+-------------------------------+
| DomainEventDispatcher         |
| (in-process, synchronous)     |
+-------------------------------+
           |
           | calls domain handler
           v
+------------------------------------------+
| CustomerCreatedDomainEventHandler        |
| - Converts to Integration Event          |
| - Calls EventBus.Publish()               |
+------------------------------------------+
           |
           | emits async message (Kafka/Outbox)
           v
+------------------------------+
|  NotificationService         |
| (another module/microservice)|
+------------------------------+
           |
           | sends welcome email
           v
       External System
```

1. `CreateCustomerHandler` → สร้าง Customer และเพิ่ม `CustomerCreated` domain event
2. `DomainEventDispatcher` → dispatch event นี้ให้ `CustomerCreatedDomainEventHandler`
3. Handler → สร้าง `CustomerCreatedIntegrationEvent` แล้วส่งผ่าน EventBus
4. ระบบภายนอก (เช่น Notification Module) consume แล้วจัดการเรื่อง Email

## Refactor เพิ่ม Domain Event

การทำ Domain Event ให้สมบูรณ์ในระบบที่ใช้ DDD (Domain-Driven Design) และ Event-Driven Architecture มีองค์ประกอบหลัก ดังนี้

1. Domain Event
    - เป็น struct ที่บรรยายเหตุการณ์ที่ “เกิดขึ้นแล้ว” ใน domain
    - อยู่ใน layer `domain` หรือ `internal/domain/event`
2. Aggregate/Entity ที่สร้าง Event
    - Entity เช่น `Customer` ต้องมีช่องทางในการเก็บ domain events (เช่น slice `[]DomainEvent`)
    - เมื่อเกิดเหตุการณ์ ให้ `append()` ลงไป
3. DomainEvent Interface
    - ใช้เป็น abstraction สำหรับ event ทั้งหมด เช่น: มี method `EventName()` หรือ `OccurredAt()`
4. Event Dispatcher
    - ดึง events จาก aggregate แล้ว dispatch ไปยังผู้รับ (handler)
5. Event Handler
    - โค้ดที่รับ event และทำงานตอบสนอง
    - อยู่ใน layer `domain` หรือ `internal/domain/eventhandler`
6. Trigger Point
    - จุดที่ pull domain events เพื่อนำไปส่งผ่าน dispatcher (มักอยู่หลัง transaction สำเร็จ)
7. Dispatch Events มี 2 แนวทางหลัก
    - ภายใน transaction (immediate dispatch) เหมาะกับ use case ที่ event handler แค่ปรับ state ภายใน เช่น update model อื่น ซึ่งจะ coupling กับ transaction logic ถ้า event handler fail จะต้อง rollback transaction ด้วย
    - หลังจาก commit แล้ว คือ ดึง domain events → รอ DB commit → dispatch เช่น post-commit hook เหมาะกับ handler ที่มี side-effect เช่น ส่งอีเมล, call external service แต่ต้องมีการจัดการ error และ retry เอง แยกออกมาจาก transaction logic

### DomainEvent Interface

ใช้เป็น abstraction สำหรับ event ทั้งหมด เช่น: มี method `EventName()` หรือ `OccurredAt()`

- สร้างไฟล์ `common/domain/event.go`

    ```go
    package domain
    
    import "time"
    
    type EventName string
    type DomainEvent interface {
     EventName() EventName
     OccurredAt() time.Time
    }
    
    type BaseDomainEvent struct {
     Name EventName
     At   time.Time
    }
    
    // ไม่ใช้ pointer receiver
    // Read-only (ไม่มีการเปลี่ยนค่า)
    // Struct ขนาดเล็ก
    
    func (e BaseDomainEvent) EventName() EventName {
     return e.Name
    }
    
    func (e BaseDomainEvent) OccurredAt() time.Time {
     return e.At
    }
    ```

### Aggregate

สำหรับเก็บ domain events

- สร้างไฟล์ `common/domain/aggregate.go`

    ```go
    package domain
    
    type Aggregate struct {
     domainEvents []DomainEvent
    }
    
    func (a *Aggregate) AddDomainEvent(dv DomainEvent) {
     if a.domainEvents == nil {
      a.domainEvents = make([]DomainEvent, 0)
     }
     a.domainEvents = append(a.domainEvents, dv)
    }
    
    func (a *Aggregate) PullDomainEvents() []DomainEvent {
     events := a.domainEvents
     a.domainEvents = nil
     return events
    }
    ```

### Event Dispatcher

สำหรับการ register handler และ dispatch ไปยังผู้รับ (handler)

- สร้างไฟล์ `common/domain/event_dispatcher.go`

    ```go
    package domain
    
    import (
     "context"
     "fmt"
     "sync"
    )
    
    var (
     ErrInvalidEvent = fmt.Errorf("invalid domain event")
    )
    
    // DomainEventHandler คือฟังก์ชันที่ handle event โดยเฉพาะ
    type DomainEventHandler interface {
     Handle(ctx context.Context, event DomainEvent) error
    }
    
    // DomainEventDispatcher is the centralized event dispatcher
    type DomainEventDispatcher interface {
     Register(eventType EventName, handler DomainEventHandler)
     Dispatch(ctx context.Context, events []DomainEvent) error
    }
    
    // simpleDomainEventDispatcher manages event handlers
    type simpleDomainEventDispatcher struct {
     handlers map[EventName][]DomainEventHandler
     mu       sync.RWMutex
    }
    
    // NewSimpleDomainEventDispatcher creates a new dispatcher
    func NewSimpleDomainEventDispatcher() DomainEventDispatcher {
     return &simpleDomainEventDispatcher{handlers: make(map[EventName][]DomainEventHandler)}
    }
    
    // Register handler สำหรับแต่ละ event name
    func (d *simpleDomainEventDispatcher) Register(eventType EventName, handler DomainEventHandler) {
     d.mu.Lock()
     defer d.mu.Unlock()
    
     d.handlers[eventType] = append(d.handlers[eventType], handler)
    }
    
    // Dispatch จะ loop event และ call handler ที่ลงทะเบียนไว้
    func (d *simpleDomainEventDispatcher) Dispatch(ctx context.Context, events []DomainEvent) error {
     for _, event := range events {
      d.mu.RLock()
      handlers := append([]DomainEventHandler(nil), d.handlers[event.EventName()]...) // เป็นการ copy slice เพื่อหลีกเลี่ยง race ถ้า handler ถูกแก้ไขระหว่าง dispatch
      d.mu.RUnlock()
    
      for _, handler := range handlers {
       err := func(h DomainEventHandler) error {
        err := h.Handle(ctx, event)
        if err != nil {
         return fmt.Errorf("error handling event %s: %w", event.EventName(), err)
        }
        return nil
       }(handler)
       if err != nil {
        return err
       }
      }
     }
     return nil
    }
    
    ```

### Domain Event

สร้าง domain event สำหรับเมื่อสร้างลูกค้าใหม่สำเร็จ

- สร้างไฟล์ `customer/internal/domain/event/customer_created.go`

    ```go
    package event
    
    import (
     "go-mma/shared/common/domain"
     "time"
    )
    
    const (
     CustomerCreatedDomainEventType domain.EventName = "CustomerCreated"
    )
    
    type CustomerCreatedDomainEvent struct {
     domain.BaseDomainEvent
     CustomerID int64
     Email      string
    }
    
    func NewCustomerCreatedDomainEvent(custID int64, email string) *CustomerCreatedDomainEvent {
     return &CustomerCreatedDomainEvent{
      BaseDomainEvent: domain.BaseDomainEvent{
       Name: CustomerCreatedDomainEventType,
       At:   time.Now(),
      },
      CustomerID: custID,
      Email:      email,
     }
    }
    ```

### Domain Event Handler

สำหรับโค้ดที่รับ event “`CustomerCreated`” มาทำงานต่อ

- สร้างไฟล์ `customer/internal/domain/eventhandler/customer_created_handler.go`

    ```go
    package eventhandler
    
    import (
     "context"
     "go-mma/modules/customer/internal/domain/event"
     notiService "go-mma/modules/notification/service"
     "go-mma/shared/common/domain"
    )
    
    type customerCreatedDomainEventHandler struct {
     notiSvc notiService.NotificationService
    }
    
    func NewCustomerCreatedDomainEventHandler(notiSvc notiService.NotificationService) domain.DomainEventHandler {
     return &customerCreatedDomainEventHandler{
      notiSvc: notiSvc,
     }
    }
    
    func (h *customerCreatedDomainEventHandler) Handle(ctx context.Context, evt domain.DomainEvent) error {
     e, ok := evt.(*event.CustomerCreatedDomainEvent) // ใช้ pointer
    
     if !ok {
      return domain.ErrInvalidEvent
     }
     // ส่งอีเมลต้อนรับ
     if err := h.notiSvc.SendEmail(e.Email, "Welcome to our service!", map[string]any{
      "message": "Thank you for joining us! We are excited to have you as a member.",
     }); err != nil {
      return err
     }
    
     return nil
    }
    ```

### Trigger Point

เป็นจุดที่ดึงเอา domain events ออกมาหลังจาก transaction logic ทั้งหมดทำงานเสร็จแล้ว แต่ยังไม่ได้ commit

- แก้ไขไฟล์ `customer/internal/feature/create/handler.go`

    ```go
    package create
    
    import (
     "context"
     "go-mma/modules/customer/domainerrors"
     "go-mma/modules/customer/internal/model"
     "go-mma/modules/customer/internal/repository"
     "go-mma/shared/common/domain" // เพิ่มตรงนี้
     "go-mma/shared/common/logger"
     "go-mma/shared/common/storage/sqldb/transactor"
    )
    
    type createCustomerCommandHandler struct {
     transactor transactor.Transactor
     custRepo   repository.CustomerRepository
     dispatcher domain.DomainEventDispatcher // เพิ่มตรงนี้ มีการใช้ dispatcher
    }
    
    func NewCreateCustomerCommandHandler(
     transactor transactor.Transactor,
     custRepo repository.CustomerRepository,
     dispatcher domain.DomainEventDispatcher, // เพิ่มตรงนี้
    ) *createCustomerCommandHandler {
     return &createCustomerCommandHandler{
      transactor: transactor,
      custRepo:   custRepo,
      dispatcher: dispatcher, // เพิ่มตรงนี้
     }
    }
    
    func (h *createCustomerCommandHandler) Handle(ctx context.Context, cmd *CreateCustomerCommand) (*CreateCustomerCommandResult, error) {
     // ...
     err := h.transactor.WithinTransaction(ctx, func(ctx context.Context, registerPostCommitHook func(transactor.PostCommitHook)) error {
    
      // ส่งไปที่ Repository Layer เพื่อบันทึกข้อมูลลงฐานข้อมูล
      if err := h.custRepo.Create(ctx, customer); err != nil {
       // error logging
       logger.Log.Error(err.Error())
       return err
      }
      
      // เพิ่มตรงนี้ หลังจากบันทึกสำเร็จแล้ว
    
      // ดึง domain events จาก customer model
      events := customer.PullDomainEvents()
    
      // ให้ dispatch หลัง commit แล้ว
      registerPostCommitHook(func(ctx context.Context) error {
       return h.dispatcher.Dispatch(ctx, events)
      })
    
      return nil
     })
    
     // ..
    }
    ```

### Dispatch Events

เนื่องจาก domain event handler ที่สร้างมาจะเป็นการส่งอีเมล ซึ่งมี side-effect จึงเหมาะกับแบบ post-commit dispatch หรือ รอให้ DB commit ก่อนค่อย dispatch

- แก้ไขไฟล์ `customer/internal/feature/create/handler.go`

    ```go
    func (h *createCustomerCommandHandler) Handle(ctx context.Context, cmd *CreateCustomerCommand) (*CreateCustomerCommandResult, error) {
     // ...
     err := h.transactor.WithinTransaction(ctx, func(ctx context.Context, registerPostCommitHook func(transactor.PostCommitHook)) error {
      // ...
      // ดึง domain events จาก customer model
      events := customer.PullDomainEvents()
    
      // ให้ dispatch หลัง commit แล้ว
      registerPostCommitHook(func(ctx context.Context) error {
       return h.dispatcher.Dispatch(ctx, events)
      })
    
      return nil
     })
    
     // ..
    }
    ```

### Register domain event

เนื่องจาก domain events เป็นการทำงานเฉพาะในโมดูลนั้นๆ เท่านั้น ดังนั้น ให้สร้าง dispatcher แยกของแต่ละโมดูลได้เลย

- แก้ไขไฟล์ `customer/module.go`

    ```go
    package customer
    
    import (
     "go-mma/modules/customer/internal/domain/event"         // เพิ่มตรงนี้่
     "go-mma/modules/customer/internal/domain/eventhandler"  // เพิ่มตรงนี้่
     "go-mma/modules/customer/internal/feature/create"
     getbyid "go-mma/modules/customer/internal/feature/get-by-id"
     releasecredit "go-mma/modules/customer/internal/feature/release-credit"
     reservecredit "go-mma/modules/customer/internal/feature/reserve-credit"
     "go-mma/modules/customer/internal/repository"
     "go-mma/shared/common/domain"                           // เพิ่มตรงนี้่
     "go-mma/shared/common/mediator"
     "go-mma/shared/common/module"
     "go-mma/shared/common/registry"
    
     notiModule "go-mma/modules/notification"
     notiService "go-mma/modules/notification/service"
    
     "github.com/gofiber/fiber/v3"
    )
    
    // ...
    
    func (m *moduleImp) Init(reg registry.ServiceRegistry) error {
     // Resolve NotificationService from the registry
     // ...
    
     // เพิ่มตรงนี้่
     // Register domain event handlerAdd commentMore actions
     dispatcher := domain.NewSimpleDomainEventDispatcher()
     dispatcher.Register(event.CustomerCreatedDomainEventType, eventhandler.NewCustomerCreatedDomainEventHandler(notiSvc))
    
     repo := repository.NewCustomerRepository(m.mCtx.DBCtx)
    
     mediator.Register(create.NewCreateCustomerCommandHandler(m.mCtx.Transactor, repo, dispatcher)) // เพิ่มส่ง dispatcher เข้า
     
     // ...
    }
    ```

## Refactor เพิ่ม Integration Event

ในการทำ Integration Event ใน Event-Driven Architecture (EDA) มีหลาย รูปแบบ (patterns) ที่สามารถเลือกใช้ได้ ขึ้นอยู่กับความ ซับซ้อนของระบบ, ระดับการ decouple, และ ความน่าเชื่อถือที่ต้องการ โดยทั่วไปสามารถแบ่งได้เป็น 3 รูปแบบหลัก ๆ ดังนี้

1. In-Memory Event Bus (Monolith)

    **ลักษณะ**

    - Event ถูกส่งแบบ in-process (memory) ไปยัง handler ที่ลงทะเบียนไว้ใน runtime เดียวกัน
    - ใช้ในระบบ monolith หรือระบบที่แยกโมดูลแต่ยังรันใน process เดียว

    **ข้อดี**

    - ง่าย
    - เร็ว

    **ข้อเสีย**

    - ไม่ทนต่อ crash
    - ถ้า handler พังหรือ panic → ไม่มี retry
    - ไม่สามารถ scale ข้าม service/process ได้
2. Outbox Pattern (Reliable Messaging in Monolith / Microservices)

    **ลักษณะ**

    - เมื่อมี event เกิดขึ้น → บันทึกทั้ง business data + integration event ใน transaction เดียวกัน
    - Event ถูกเก็บใน outbox table
    - Worker (หรือ background process) คอยอ่านและส่งไปยัง message broker (Kafka, RabbitMQ)

    **ข้อดี**

    - ปลอดภัย (atomic): business data + event commit พร้อมกัน
    - ทนต่อ crash
    - Decouple services ได้ (publish ไป Kafka)

    **ข้อเสีย**

    - ต้องมี worker ดึงและส่ง
    - ซับซ้อนกว่า in-memory
3. Change Data Capture (CDC)

    **ลักษณะ**

    - ใช้ระบบอย่าง Debezium หรือ Kafka Connect ฟังการเปลี่ยนแปลงใน DB (ผ่าน WAL หรือ binlog)
    - เมื่อมี insert/update → สร้างเป็น event และส่งออกไป message broker

    **ข้อดี**

    - ไม่ต้องมี Worker (หรือ background process) คอยอ่านและส่งไปยัง message broker
    - มองเห็นทุกการเปลี่ยนแปลงของฐานข้อมูล

    **ข้อเสีย**

    - ต้องจัดการ schema evolution และ data format ให้ดี

### In-Memory Event Bus (Monolith)

ในบทความนี้จะเลือกใช้วิธีแบบ In-Memory Event Bus เพราะระบบเป็น Monolith และง่ายต่อการทำความเข้าใจ

การทำ Integration Event แบบ In-Memory Event Bus ภายใน Monolith คือการสื่อสารระหว่างโมดูล (bounded contexts) โดยไม่ใช้ messaging system ภายนอก เช่น Kafka หรือ RabbitMQ แต่ยังแยก "Integration Event" ออกจาก "Domain Event" เพื่อรักษา separation of concerns

มีองค์ประกอบหลัก ดังนี้

1. Integration Event
    - เป็น struct ที่ใช้สื่อสารข้ามโมดูล (context) ภายในระบบเดียวกัน
    - มี payload ที่ module ปลายทางต้องใช้ เช่น `CustomerCreatedIntegrationEvent`
2. Integration Event Interface
    - ใช้เป็น abstraction สำหรับ event ทั้งหมด เช่น: มี method `EventID()`หรือ `EventName()` หรือ `OccurredAt()`
3. Event Bus (In-Memory Implementation)
    - ตัวกลางในการ publish → ไปยัง handler ที่ลงทะเบียนไว้
    - เก็บ handler เป็น map จาก event name → handler list
4. Register / Subscribe
    - Module ที่สนใจ event ต้องลงทะเบียน handler ไว้กับ EventBus
5. Publish
    - เมื่อ module ต้นทางสร้าง event แล้วเรียก `eventBus.Publish(...)`
    - EventBus จะกระจาย event ไปยัง handler ที่ลงทะเบียนไว้
6. Event Handlers
    - แต่ละ handler มี logic ของตัวเอง เช่นส่งอีเมล

### สร้าง Integration Event Interface

- ใช้เป็น abstraction สำหรับ event ทั้งหมด เช่น: มี method `EventID()`หรือ `EventName()` หรือ `OccurredAt()`
- สร้างไฟล์ `common/eventbus/event.go`

    ```go
    package eventbus
    
    import (
     "time"
    )
    
    type EventName string
    
    type Event interface {
     EventID() string       // UUID หรือ ULID
     EventName() EventName  // เช่น "CustomerCreated"
     OccurredAt() time.Time // เวลาที่ event เกิด
    }
    
    type BaseEvent struct {
     ID   string
     Name EventName
     At   time.Time
    }
    
    func (e BaseEvent) EventID() string {
     return e.ID
    }
    
    func (e BaseEvent) EventName() EventName {
     return e.Name
    }
    
    func (e BaseEvent) OccurredAt() time.Time {
     return e.At
    }
    ```

### สร้าง EventBus (In-Memory Implementation)

สำหรับเป็นตัวกลางในการ publish → ไปยัง handler ที่ลงทะเบียนไว้

- สร้างไฟล์ `common/eventbus/eventbus.go`

    ```go
    package eventbus
    
    import (
     "context"
    )
    
    type IntegrationEventHandler interface {
     Handle(ctx context.Context, event Event) error
    }
    
    type EventBus interface {
     Publish(ctx context.Context, event Event) error
     Subscribe(eventName string, handler IntegrationEventHandler)
    }
    ```

- สร้างไฟล์ `common/eventbus/in_memory_eventbus.go`

    ```go
    package eventbus
    
    import (
     "context"
     "log"
     "sync"
    )
    
    // InMemoryEventBus is a simple event bus
    type InMemoryEventBus struct {
     subscribers map[string][]IntegrationEventHandler
     mu          sync.RWMutex
    }
    
    // NewInMemoryEventBus creates an event bus instance
    func NewInMemoryEventBus() *InMemoryEventBus {
     return &InMemoryEventBus{
      subscribers: make(map[string][]IntegrationEventHandler),
     }
    }
    
    // Subscribe registers a handler for a specific event
    func (eb *InMemoryEventBus) Subscribe(eventName string, handler IntegrationEventHandler) {
     eb.mu.Lock()
     defer eb.mu.Unlock()
    
     eb.subscribers[eventName] = append(eb.subscribers[eventName], handler)
    }
    
    // Publish sends an event to all subscribers
    func (eb *InMemoryEventBus) Publish(ctx context.Context, event Event) error {
     eb.mu.RLock()
     defer eb.mu.RUnlock()
    
     handlers, ok := eb.subscribers[event.EventName()]
     if !ok {
      return nil
     }
    
     busCtx := context.WithValue(ctx, "name", "context in event bus")
     for _, handler := range handlers {
      go func(h IntegrationEventHandler) {
       err := h.Handle(busCtx, event)
       if err != nil {
        log.Printf("error handling event %s: %v", event.EventName(), err)
       }
      }(handler)
     }
     return nil
    }
    ```

### สร้าง Integration Event

เนื่องจาก integration event จะต้องใช้ร่วมกันหลายโมดูล ให้สร้างโปรเจกต์ใหม่ชื่อ messaging ใน shared

- สร้างโปรเจกต์ `messaging`

    ```bash
    mkdir -p src/shared/messaging
    cd src/shared/messaging
    go mod init go-mma/shared/messaging
    ```

    เพิ่ม module replace สำหรับโปรเจกต์ `common`

    ```go
    module go-mma/shared/messaging
    
    go 1.24.1
    
    replace go-mma/shared/common v0.0.0 => ../common
    
    require go-mma/shared/common v0.0.0
    ```

    อย่าลืมเพิ่มลง workspace ด้วย

    ```go
    {
      "folders": [
        // ...,
        {
          "path": "src/shared/messaging"
        }
      ],
      "settings": {}
    }
    ```

- เพิ่ม module replace ในทุกโมดูลโปรเจกต์ และ `app`

    ```go
    // app
    replace go-mma/shared/messaging v0.0.0 => ../shared/messaging
    
    // customer
    replace go-mma/shared/messaging v0.0.0 => ../../shared/messaging
    
    // order
    replace go-mma/shared/messaging v0.0.0 => ../../shared/messaging
    
    // notification
    replace go-mma/shared/messaging v0.0.0 => ../../shared/messaging
    ```

- สร้างไฟล์ `messaging/customer_created.go`

    ```go
    package messaging
    
    import (
     "go-mma/shared/common/eventbus"
     "go-mma/shared/common/idgen"
     "time"
    )
    
    const (
     CustomerCreatedIntegrationEventName eventbus.EventName = "CustomerCreated"
    )
    
    type CustomerCreatedIntegrationEvent struct {
     eventbus.BaseEvent
     CustomerID int64  `json:"customer_id"`
     Email      string `json:"email"`
    }
    
    func NewCustomerCreatedIntegrationEvent(customerID int64, email string) *CustomerCreatedIntegrationEvent {
     return &CustomerCreatedIntegrationEvent{
      BaseEvent: eventbus.BaseEvent{
       ID:   idgen.GenerateUUIDLikeID(),
       Name: CustomerCreatedIntegrationEventName,
       At:   time.Now(),
      },
      CustomerID: customerID,
      Email:      email,
     }
    }
    ```

### สร้าง Integration Event Handler

สำหรับโค้ดที่รับ event “`CustomerCreated`” มาทำงานต่อ โดยใยที่นี่จะทำที่โมดูล notification เพื่อส่ง welcome email

- สร้างไฟล์ `notification/internal/integration/customer/welcome_email_handler.go` (สื่อว่า integration จากโมดูล customer)

    ```go
    package customer
    
    import (
     "context"
     "fmt"
     "go-mma/modules/notification/service"
     "go-mma/shared/common/eventbus"
     "go-mma/shared/messaging"
    )
    
    type welcomeEmailHandler struct {
     notiService service.NotificationService
    }
    
    func NewWelcomeEmailHandler(notiService service.NotificationService) *welcomeEmailHandler {
     return &welcomeEmailHandler{
      notiService: notiService,
     }
    }
    
    func (h *welcomeEmailHandler) Handle(ctx context.Context, evt eventbus.Event) error {
     e, ok := evt.(messaging.CustomerCreatedIntegrationEvent)
     if !ok {
      return fmt.Errorf("invalid event type")
     }
    
     return h.notiService.SendEmail(e.Email, "Welcome to our service!", map[string]any{
      "message": "Thank you for joining us! We are excited to have you as a member.",
     })
    }
    ```

### สร้าง Integration Event Publisher

เดิมใน CustomerCreatedDomainEventHandler จะมีการเรียก notiService เพื่อส่งอีเมลโดยตรง เราจะเปลี่ยนส่งนี้ให้ส่งไปเป็น integration event แทน

- แก้ไขไฟล์  `customer/internal/domain/eventhandler/customer_created_handler.go`

    ```go
    package eventhandler
    
    import (
     "context"
     "go-mma/modules/customer/internal/domain/event"
     "go-mma/shared/common/domain"
     "go-mma/shared/common/eventbus"
     "go-mma/shared/messaging"
    )
    
    type customerCreatedDomainEventHandler struct {
     eventBus eventbus.EventBus // เปลี่ยนมาใช้ eventbus
    }
    
    // เปลี่ยนมาใช้ eventbus
    func NewCustomerCreatedDomainEventHandler(eventBus eventbus.EventBus) domain.DomainEventHandler {
     return &customerCreatedDomainEventHandler{
      eventBus: eventBus, // เปลี่ยนมาใช้ eventbus
     }
    }
    
    func (h *customerCreatedDomainEventHandler) Handle(ctx context.Context, evt domain.DomainEvent) error {
     e, ok := evt.(*event.CustomerCreatedDomainEvent) // ใช้ pointer
    
     if !ok {
      return domain.ErrInvalidEvent
     }
    
     // สร้าง IntegrationEvent จาก Domain Event
     integrationEvent := messaging.NewCustomerCreatedIntegrationEvent(
      e.CustomerID,
      e.Email,
     )
    
     return h.eventBus.Publish(ctx, integrationEvent)
    }
    ```

- แก้ไขไฟล์  `common/module/module.go` เพื่อรองรับ event bus

    ```go
    type Module interface {
     APIVersion() string
     Init(reg registry.ServiceRegistry, eventBus eventbus.EventBus) error // รับ eventBus เพิ่ม
     RegisterRoutes(r fiber.Router)
    }
    ```

- แก้ไขไฟล์  `customer/module.go` เพื่อลบ notification service ออก

    ```go
    func (m *moduleImp) Init(reg registry.ServiceRegistry, eventbus eventbus.EventBus) error {
     // เอา notiSvc ออก
     
     // Register domain event handlerAdd commentMore actions
     dispatcher := domain.NewSimpleDomainEventDispatcher()
     dispatcher.Register(event.CustomerCreatedDomainEventType, eventhandler.NewCustomerCreatedDomainEventHandler(eventbus)) // ส่ง eventBus เข้าไปแทน
    
     repo := repository.NewCustomerRepository(m.mCtx.DBCtx)
    
     mediator.Register(create.NewCreateCustomerCommandHandler(m.mCtx.Transactor, repo, dispatcher))
     mediator.Register(getbyid.NewGetCustomerByIDQueryHandler(repo))
     mediator.Register(reservecredit.NewReserveCreditCommandHandler(m.mCtx.Transactor, repo))
     mediator.Register(releasecredit.NewReleaseCreditCommandHandler(m.mCtx.Transactor, repo))
    
     return nil
    }
    ```

### สร้าง Register / Subscribe

ให้โมดูล notification คอยรับ integration event

- แก้ไขไฟล์ `notification/module.go`

    ```go
    func (m *moduleImp) Init(reg registry.ServiceRegistry, eventBus eventbus.EventBus) error {
     m.notiSvc = service.NewNotificationService()
    
     // subscribe to integration events
     eventBus.Subscribe(messaging.CustomerCreatedIntegrationEventName, customer.NewWelcomeEmailHandler(m.notiSvc))
    
     return nil
    }
    ```

- แก้ไฟล์ `app/application/application.go` เพื่อสร้าง event bus

    ```go
    package application
    
    import (
      // ...
     "go-mma/shared/common/eventbus"
      // ...
    )
    
    type Application struct {
     config          config.Config
     httpServer      HTTPServer
     serviceRegistry registry.ServiceRegistry
     eventBus        eventbus.EventBus // เพ่ิม
    }
    
    func New(config config.Config) *Application {
     return &Application{
      config:          config,
      httpServer:      newHTTPServer(config),
      serviceRegistry: registry.NewServiceRegistry(),
      eventBus:        eventbus.NewInMemoryEventBus(), // เพ่ิม
     }
    }
    
    // ...
    
    func (app *Application) initModule(m module.Module) error {
     return m.Init(app.serviceRegistry, app.eventBus) // เพ่ิมส่ง eventBus
    }
    
    // ...
    ```

เพียงเท่านี้ก็สามารถใช้ integration event แบบ in-memory event bus ได้แล้ว แต่อย่าลืมว่าวิธีมีข้อเสียคือ ถ้า handler พังหรือ panic จะไม่มี retry อาจทำให้เกิด inconsistency เช่น ลูกค้าถูกสร้างแล้ว (`INSERT INTO customers`) แต่ไม่ส่งอีเมลต้อนรับ วิธีแก้ ได้แก่

- อาจเพิ่ม retry logic ตอนส่งอีเมล
- ใช้แนวทาง Hybrid Approach คือ ใช้ Domain Event → แปลง (map) เป็น Integration Event → เขียน Outbox table (ต้องทำใน transaction เดียวกับ business data)→ ใช้ CDC tools ส่ง event
