package gapi

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testStore *db.Store
var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("..")
	if err != nil {
		panic(fmt.Sprintf("cannot load config: %v", err))
	}

	testDB, err = pgxpool.New(context.Background(), config.TESTING_DB_URL)
	if err != nil {
		panic("cannot connect to the database: " + err.Error())
	}
	defer testDB.Close()

	testStore = db.NewStore(testDB)
	os.Exit(m.Run())
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	server, err := NewServer(testStore, util.Config{
		TOKEN_SYMMETRIC_KEY:   util.RandomString(32),
		ACCESS_TOKEN_DURATION: time.Minute,
	}, nil)
	if err != nil {
		t.Fatalf("cannot create test server: %v", err)
	}

	return server
}
