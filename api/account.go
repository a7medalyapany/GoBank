package api

import (
	"math/big"
	"net/http"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)


type createAccountRequest struct {
	Owner    string         `json:"owner" binding:"required"`
	Currency string         `json:"currency" binding:"required,oneof=USD EUR EGP"`
}


func (server *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest

	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.CreateAccountParams{
		Owner: req.Owner,
		Currency: req.Currency,
		Balance: pgtype.Numeric{Int: big.NewInt(0), Exp: 0, Valid: true},
	}

	account, err := server.store.CreateAccount(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)
}