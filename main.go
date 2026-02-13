package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/a7medalyapany/GoBank.git/api"
	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		panic(fmt.Sprintf("cannot load config: %v", err))
	}

	if config.DB_URL == "" || config.PORT == "" || config.SERVER_ADDRESS == "" {
		panic("DB_URL, PORT, and SERVER_ADDRESS must be set in environment variables")
	}

	serverAddress := strings.TrimPrefix(config.SERVER_ADDRESS, "http://")
	serverAddress = strings.TrimPrefix(serverAddress, "https://")

	conn, err := pgxpool.New(context.Background(), config.DB_URL)
	if err != nil {
		panic(fmt.Sprintf("cannot connect to the database; verify DB_URL credentials and host: %v", err))
	}
	defer conn.Close()

	store := db.NewStore(conn)
	server := api.NewServer(store)

	err = server.Start(serverAddress + ":" + config.PORT)
	if err != nil {
		panic("cannot start the server: " + err.Error())
	}
}
