package api

import (
	"errors"
	"fmt"
	"net/http"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/token"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type transferRequest struct {
    FromAccountID int64   `json:"from_account_id" binding:"required,min=1"`
    ToAccountID   int64   `json:"to_account_id" binding:"required,min=1"`
    Amount        float64 `json:"amount" binding:"required,gt=0"`
    Currency      string  `json:"currency" binding:"required,currency"`
}

func (server *Server) createTransfer(ctx *gin.Context) {
    var req transferRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, errorResponse(err))
        return
    }

    // Convert dollars to cents
    amountCents := util.FloatToCents(req.Amount) // ← e.g., 10.50 → 1050

    // Validate accounts
    fromAccount, valid := server.validAccount(ctx, req.FromAccountID, req.Currency)
    if !valid {
        return
    }

    authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
    if fromAccount.Owner != authPayload.Username {
        err := errors.New("account doesn't belong to the authenticated user")
        ctx.JSON(http.StatusUnauthorized, errorResponse(err))
        return
    }

    if !server.validAccountCurrency(ctx, req.ToAccountID, req.Currency) {
        return
    }

    // Check sufficient balance (simple integer comparison!)
    if fromAccount.Balance < amountCents {
        err := fmt.Errorf("insufficient balance: account %d has $%.2f, but transfer requires $%.2f",
            req.FromAccountID,
            util.CentsToFloat(fromAccount.Balance),
            req.Amount,
        )
        ctx.JSON(http.StatusBadRequest, errorResponse(err))
        return
    }

    arg := db.TransferTxParams{
        FromAccountID: req.FromAccountID,
        ToAccountID:   req.ToAccountID,
        Amount:        amountCents,
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
        if errors.Is(err, pgx.ErrNoRows) {
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