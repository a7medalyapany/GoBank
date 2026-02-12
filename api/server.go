package api

import (
	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/gin-gonic/gin"
)

// Server represents the API server, it serves HTTP requests.
type Server struct {
	store *db.Store
	router *gin.Engine
}


// NewServer creates a new HTTP server and setup routing.
func NewServer(store *db.Store) *Server {
	server := &Server{store: store}
	router := gin.Default()

	// add route to router
	router.POST("/accounts", server.createAccount)

	server.router = router
	return server
}


// Start runs the HTTP server on a specific address.
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

// errorResponse returns a JSON response with the error message.
func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}