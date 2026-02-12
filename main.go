package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/a7medalyapany/GoBank.git/api"
	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	port := os.Getenv("PORT")
	serverAddress := os.Getenv("SERVER_ADDRESS")

	if dbURL == "" || port == "" || serverAddress == "" {
		panic("DB_URL, PORT, and SERVER_ADDRESS must be set in environment variables")
	}

	serverAddress = strings.TrimPrefix(serverAddress, "http://")
	serverAddress = strings.TrimPrefix(serverAddress, "https://")

	conn, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		panic(fmt.Sprintf("cannot connect to the database; verify DB_URL credentials and host: %v", err))
	}
	defer conn.Close()

	store := db.NewStore(conn)
	server := api.NewServer(store)

	err = server.Start(serverAddress + ":" + port)
	if err != nil {
		panic("cannot start the server: " + err.Error())
	}
}
