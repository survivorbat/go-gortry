package gortry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type dummyObject struct {
	Name string
}

func TestPlugin_RetriesOnErrors(t *testing.T) {
	t.Parallel()
	// Arrange
	db := newDB(t)

	plugin, _ := New(DefaultConfig())

	var calls int

	// Override the default query behaviour and return a net.Error
	_ = db.Callback().Query().Replace("gorm:query", func(d *gorm.DB) {
		calls++

		if calls < 3 {
			_ = d.AddError(new(retryError))
			return
		}

		dest := d.Statement.Dest.(*[]*dummyObject)
		*dest = []*dummyObject{{Name: "foo"}}
	})

	err := db.Use(plugin)
	require.NoError(t, err)

	var actual []*dummyObject

	// Act
	err = db.Find(&actual).Error

	// Assert
	require.NoError(t, err)

	assert.Equal(t, 3, calls, "expected 3 calls")
	assert.Equal(t, []*dummyObject{{Name: "foo"}}, actual)
}

// newDB makes it easy to instantiate a database with a dummy object migrated
func newDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"))
	require.NoError(t, err)

	err = db.AutoMigrate(&dummyObject{})
	require.NoError(t, err)

	return db
}
