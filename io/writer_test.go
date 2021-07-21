package io_test

import (
	"context"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/viant/sqlx/io"
	"github.com/viant/sqlx/metadata"
	_ "github.com/viant/sqlx/metadata/product/sqlite"
	"github.com/viant/sqlx/opt"
	"testing"
)

func TestWriter_Insert(t *testing.T) {

	type entity struct {
		Id   int    `sqlx:"name=foo_id,primaryKey=true"`
		Name string `sqlx:"foo_name"`
		Desc string `sqlx:"-"`
		Bar  float64
	}
	type entityWithAutoIncrement struct {
		Id   int    `sqlx:"name=foo_id,autoincrement=true"`
		Name string `sqlx:"foo_name"`
		Desc string `sqlx:"-"`
		Bar  float64
	}

	var useCases = []struct {
		description string
		table       string
		driver      string
		dsn         string
		options     []opt.Option
		records     interface{}
		params      []interface{}
		expect      interface{}
		initSQL     []string
		affected    int64
		lastID      int64
	}{
		{
			description: "Writer.Insert ",
			driver:      "sqlite3",
			dsn:         "/tmp/sqllite.db",
			table:       "t1",
			initSQL: []string{
				"DROP TABLE IF EXISTS t1",
				"CREATE TABLE t1 (foo_id INTEGER PRIMARY KEY, foo_name TEXT, bar INTEGER)",
			},
			records: []interface{}{
				&entity{Id: 1, Name: "John1", Desc: "description", Bar: 17},
				&entity{Id: 2, Name: "John2", Desc: "description", Bar: 18},
				&entity{Id: 3, Name: "John3", Desc: "description", Bar: 19},
			},
			affected: 3,
			lastID: 3,
		},
		{
			description: "Writer.Insert: batch size: 2 ",
			driver:      "sqlite3",
			dsn:         "/tmp/sqllite.db",
			table:       "t2",
			initSQL: []string{
				"DROP TABLE IF EXISTS t2",
				"CREATE TABLE t2 (foo_id INTEGER PRIMARY KEY, foo_name TEXT, Bar INTEGER)",
			},
			records: []entity{
				{Id: 1, Name: "John1", Desc: "description", Bar: 17},
				{Id: 2, Name: "John2", Desc: "description", Bar: 18},
				{Id: 3, Name: "John3", Desc: "description", Bar: 19},
			},
			affected: 3,
			lastID: 3,
			options: []opt.Option{
				&opt.BatchOption{2},
			},
		},
		{
			description: "Writer.Insert - autoincrement batch - empty table ",
			driver:      "sqlite3",
			dsn:         "/tmp/sqllite.db",
			table:       "t3",
			initSQL: []string{
				"DROP TABLE IF EXISTS t3",
				"CREATE TABLE t3 (foo_id INTEGER PRIMARY KEY AUTOINCREMENT, foo_name TEXT, Bar INTEGER)",
			},
			records: []*entityWithAutoIncrement{
				{Id: 0, Name: "John1", Desc: "description", Bar: 17},
				{Id: 0, Name: "John2", Desc: "description", Bar: 18},
				{Id: 0, Name: "John3", Desc: "description", Bar: 19},
			},
			affected: 3,
			lastID: 3,
			options: []opt.Option{
				&opt.BatchOption{2},
			},
		},
		{
			description: "Writer.Insert - autoincrement batch - existing data",
			driver:      "sqlite3",
			dsn:         "/tmp/sqllite.db",
			table:       "t4",
			initSQL: []string{
				"DROP TABLE IF EXISTS t4",
				"CREATE TABLE t4 (foo_id INTEGER PRIMARY KEY AUTOINCREMENT, foo_name TEXT, Bar INTEGER)",
				"INSERT INTO t4(foo_name) VALUES('test')",
			},
			records: []*entityWithAutoIncrement{
				{Id: 0, Name: "John1", Desc: "description", Bar: 17},
				{Id: 0, Name: "John2", Desc: "description", Bar: 18},
				{Id: 0, Name: "John3", Desc: "description", Bar: 19},
				{Id: 0, Name: "John4", Desc: "description", Bar: 19},
				{Id: 0, Name: "John5", Desc: "description", Bar: 19},
			},
			affected: 5,
			lastID: 6,
			options: []opt.Option{
				&opt.BatchOption{3},
			},
		},
	}

outer:

	for _, testCase := range useCases {

		//ctx := context.Background()
		var db *sql.DB

		db, err := sql.Open(testCase.driver, testCase.dsn)
		if !assert.Nil(t, err, testCase.description) {
			continue
		}

		for _, SQL := range testCase.initSQL {
			_, err := db.Exec(SQL)
			if !assert.Nil(t, err, testCase.description) {
				continue outer
			}
		}

		meta := metadata.New()
		product, err := meta.DetectProduct(context.TODO(), db)
		if !assert.Nil(t, err, testCase.description) {
			continue
		}
		testCase.options = append(testCase.options, product)
		writer, err := io.NewWriter(context.TODO(), db, testCase.table, testCase.options...)
		if !assert.Nil(t, err, testCase.description) {
			continue
		}
		affected, lastID, err := writer.Insert(testCase.records)
		assert.Nil(t, err, testCase.description)
		assert.EqualValues(t, testCase.affected, affected, testCase.description)
		assert.EqualValues(t, testCase.lastID, lastID, testCase.description)

	}

}
