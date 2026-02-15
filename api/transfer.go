package api

import (
	"database/sql"
	"fmt"
	"math"
	"math/big"
	"net/http"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type transferRequest struct {
	FromAccountID int64   `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64   `json:"to_account_id" binding:"required,min=1"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	Currency      string  `json:"currency" binding:"required,oneof=USD EUR EGP"`
}

func (server *Server) createTransfer(ctx *gin.Context) {
	var req transferRequest

	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Validate from account (includes balance check)
	fromAccount, valid := server.validAccount(ctx, req.FromAccountID, req.Currency)
	if !valid {
		return
	}

	// Validate to account (no balance check needed)
	if !server.validAccountCurrency(ctx, req.ToAccountID, req.Currency) {
		return
	}

	// Check sufficient balance
	amountInCents := int64(math.Round(req.Amount * 100))
	amount := pgtype.Numeric{
		Int:   big.NewInt(amountInCents),
		Exp:   -2,
		Valid: true,
	}

	// Compare using the helper function
	if db.CompareNumeric(fromAccount.Balance, amount) < 0 {
		err := fmt.Errorf("insufficient balance: account %d has %s %s, but transfer requires %.2f %s",
			req.FromAccountID,
			db.FormatMoney(fromAccount.Balance),
			req.Currency,
			req.Amount,
			req.Currency,
		)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.TransferTxParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        amount,
	}

	result, err := server.store.TransferTx(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// validAccount checks if account exists and has matching currency
// Returns the account if valid, and a boolean indicating validity
func (server *Server) validAccount(ctx *gin.Context, accountID int64, currency string) (db.Account, bool) {
	account, err := server.store.GetAccount(ctx, accountID)

	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return account, false
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return account, false
	}

	if account.Currency != currency {
		err := fmt.Errorf("account %d currency mismatch: expected %s, got %s", 
			accountID, currency, account.Currency)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return account, false
	}

	return account, true
}

// validAccountCurrency only checks existence and currency (for to_account)
func (server *Server) validAccountCurrency(ctx *gin.Context, accountID int64, currency string) bool {
	_, valid := server.validAccount(ctx, accountID, currency)
	return valid
}