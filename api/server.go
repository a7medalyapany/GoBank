package api

import (
	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
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


	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		panic("failed to initialize validator")
	}
	v.RegisterValidation("currency", validCurrency)


	// users' APIs
	router.POST("/users", server.createUser)


	// accounts' APIs
	router.POST("/accounts", server.createAccount)
	router.GET("/accounts/:id", server.getAccount)
	router.GET("/accounts", server.listAccounts)
	router.PUT("/accounts/:id", server.updateAccount)
	router.DELETE("/accounts/:id", server.deleteAccount)


	// transfers' APIs
	router.POST("/transfers", server.createTransfer)

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