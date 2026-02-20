package api

import (
	"fmt"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/token"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Server represents the API server, it serves HTTP requests.
type Server struct {
	store *db.Store
	router *gin.Engine
	config util.Config
	tokenMaker token.Maker
}


// NewServer creates a new HTTP server and setup routing.
func NewServer(store *db.Store, config util.Config) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TOKEN_SYMMETRIC_KEY)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		store: store,
		config: config,
		tokenMaker: tokenMaker,
	}

	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		panic("failed to initialize validator")
	}
	v.RegisterValidation("currency", validCurrency)


	server.setupRouter()
	return server, nil
}


// Start runs the HTTP server on a specific address.
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

// errorResponse returns a JSON response with the error message.
func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

// setupRouter sets up the routing of the HTTP server.
func (server *Server) setupRouter() {
	router := gin.Default()

	// users' APIs
	router.POST("/users", server.createUser)

	// auth's APIs
	router.POST("/auth/login", server.loginUser)

	// accounts' APIs
	router.POST("/accounts", server.createAccount)
	router.GET("/accounts/:id", server.getAccount)
	router.GET("/accounts", server.listAccounts)
	router.PUT("/accounts/:id", server.updateAccount)
	router.DELETE("/accounts/:id", server.deleteAccount)

	// transfers' APIs
	router.POST("/transfers", server.createTransfer)

	server.router = router
}