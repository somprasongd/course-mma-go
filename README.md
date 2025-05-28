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
