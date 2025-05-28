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
