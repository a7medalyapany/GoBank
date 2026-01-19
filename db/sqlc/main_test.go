package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	dbSource = "postgresql://root:password@localhost:5432/bank?sslmode=disable"
)

var testQueries *Queries
var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	var err error
	testDB, err = pgxpool.New(context.Background(), dbSource)
	if err != nil {
		panic("Can not connect to the database: " + err.Error())
	}
	testQueries = New(testDB)
	defer testDB.Close()
	os.Exit(m.Run())
}