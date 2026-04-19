package db

import (
	"database/sql"
	"log"
	"os"
	"simplebank/util"
	"testing"

	_ "github.com/lib/pq"
)

var testQueries *Queries
var testDB *sql.DB

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../..")
	testDB, err = sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	// 验证连接是否活着
	if err = testDB.Ping(); err != nil {
		log.Fatal("cannot ping db:", err)
	}

	testQueries = New(testDB)

	os.Exit(m.Run())
}
