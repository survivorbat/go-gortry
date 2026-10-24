# ♻️ Gorm Retry

Gortry (gorm-retry) is a gorm plugin that aims to retry READ queries that failed
due to a connection failure.
This would allow a database to reboot without queries immediately failing.

## ⬇️ Installation

```bash
go get github.com/survivorbat/go-gortry
```

## 📋 Usage

```go
package main

import (
  "github.com/survivorbat/go-gortry"

  "gorm.io/driver/sqlite"
  "gorm.io/gorm"
)

func main() {
  db, err := gorm.Open(sqlite.Open(":memory:"))
  if err != nil {
    panic(err)
  }

  gortry, err := gortry.New(gortry.DefaultConfig())
  if err != nil {
    panic(err)
  }

  err = db.Use(gortry.New())
  if err != nil {
    panic(err)
  }
}
```

## 🔭 Plans

Not much yet
