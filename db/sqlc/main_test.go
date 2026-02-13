package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5/pgxpool"
)


var testQueries *Queries
var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	var err error

	config, err := util.LoadConfig("../..")
	if err != nil {
		panic(fmt.Sprintf("cannot load config: %v", err))
	}

	testDB, err = pgxpool.New(context.Background(), config.TESTING_DB_URL)
	if err != nil {
		panic("Can not connect to the database: " + err.Error())
	}
	testQueries = New(testDB)
	defer testDB.Close()
	os.Exit(m.Run())
}