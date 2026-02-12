package api

import (
	"database/sql"
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

type getAccountRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}
func (server *Server) getAccount(ctx *gin.Context) {
	var req getAccountRequest

	err := ctx.ShouldBindUri(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}


	account, err := server.store.GetAccount(ctx, req.ID)
	if err != nil {

		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)
}


type listAccountsRequest struct {
	PageID int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (server *Server) listAccounts(ctx *gin.Context) {
	var req listAccountsRequest

	err := ctx.ShouldBindQuery(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.ListAccountsParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	accounts, err := server.store.ListAccounts(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, accounts)
}