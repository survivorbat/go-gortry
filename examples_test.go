package gortry

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DummyObject struct {
	Name string
}

var data = []DummyObject{{Name: "foo"}, {Name: "bar"}}

func ExamplePlugin() {
	db, err := gorm.Open(sqlite.Open(":memory:"))
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(&DummyObject{})
	if err != nil {
		panic(err)
	}

	gortry, err := New(DefaultConfig())
	if err != nil {
		panic(err)
	}

	err = db.Use(gortry)
	if err != nil {
		panic(err)
	}

	err = db.Create(data).Error
	if err != nil {
		panic(err)
	}

	var actual []DummyObject
	err = db.Order("name ASC").Find(&actual).Error
	if err != nil {
		panic(err)
	}

	fmt.Println(actual)

	// Output:
	// [{bar} {foo}]
}
