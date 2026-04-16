package worker

import (
	"context"
	"fmt"
	"os"
	"testing"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

var workerTestStore *db.Store
var workerTestDB *pgxpool.Pool

func TestMain(m *testing.M) {
	config, err := util.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("cannot load config: %v", err))
	}

	workerTestDB, err = pgxpool.New(context.Background(), config.TESTING_DB_URL)
	if err != nil {
		panic("cannot connect to the database: " + err.Error())
	}
	defer workerTestDB.Close()

	workerTestStore = db.NewStore(workerTestDB)
	os.Exit(m.Run())
}
