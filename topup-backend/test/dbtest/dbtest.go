// Package dbtest adalah helper test untuk PostgreSQL sungguhan yang sudah
// dimigrasi. Koneksi dibaca dari TEST_DB_* (lihat test/integration); tanpa
// TEST_DB_NAME, test di-skip. Hanya untuk file _test.go.
package dbtest

import (
	"os"
	"reflect"
	"slices"
	"sync"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/config"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/database"
)

// Open membuka koneksi ke database test, atau men-skip test bila TEST_DB_NAME kosong.
func Open(t testing.TB) *gorm.DB {
	t.Helper()
	if os.Getenv("TEST_DB_NAME") == "" {
		t.Skip("TEST_DB_NAME tidak di-set")
	}
	cfg, err := config.LoadDatabaseWithPrefix("TEST_")
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.Open(cfg.URL()), &gorm.Config{
		TranslateError: true,
		Logger:         gormlogger.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close(db) })
	return db
}

type column struct {
	Name     string `gorm:"column:column_name"`
	Nullable string `gorm:"column:is_nullable"`
}

var deletedAtType = reflect.TypeFor[gorm.DeletedAt]()

// AssertModelsMatchSchema memastikan setiap model GORM punya kolom yang sama
// persis dengan tabelnya di database (RULES §4.4), dan bahwa kolom nullable
// dipetakan ke tipe yang bisa nil, kolom NOT NULL tidak ke pointer.
func AssertModelsMatchSchema(t *testing.T, db *gorm.DB, models ...any) {
	t.Helper()
	for _, m := range models {
		s, err := schema.Parse(m, &sync.Map{}, db.NamingStrategy)
		if err != nil {
			t.Fatalf("parse %T: %v", m, err)
		}
		var cols []column
		if err := db.Raw(`SELECT column_name, is_nullable FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = ? ORDER BY ordinal_position`, s.Table).
			Scan(&cols).Error; err != nil {
			t.Fatal(err)
		}
		if len(cols) == 0 {
			t.Errorf("%T: tabel %q tidak ada", m, s.Table)
			continue
		}
		dbNames := make([]string, len(cols))
		for i, c := range cols {
			dbNames[i] = c.Name
			f := s.LookUpField(c.Name)
			if f == nil {
				continue
			}
			kind := f.FieldType.Kind()
			nilable := kind == reflect.Pointer || kind == reflect.Slice || f.FieldType == deletedAtType
			switch {
			case c.Nullable == "YES" && !nilable:
				t.Errorf("%s.%s nullable, tetapi field %s bertipe %s", s.Table, c.Name, f.Name, f.FieldType)
			case c.Nullable == "NO" && (kind == reflect.Pointer || f.FieldType == deletedAtType):
				t.Errorf("%s.%s NOT NULL, tetapi field %s bertipe %s", s.Table, c.Name, f.Name, f.FieldType)
			}
		}
		modelNames := slices.Clone(s.DBNames)
		slices.Sort(modelNames)
		sortedDB := slices.Clone(dbNames)
		slices.Sort(sortedDB)
		if !slices.Equal(modelNames, sortedDB) {
			t.Errorf("%s: kolom model %v ≠ kolom tabel %v", s.Table, modelNames, sortedDB)
		}
	}
}
