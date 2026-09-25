package router

import (
	"errors"
	"fmt"
	"itk-tt/services/wallets"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const internalErrorMessage = "internal server error"

type ErrorResponse struct {
	Message string `json:"message"`
}

func writeError(c *gin.Context, message string, code int) {
	c.JSON(code, ErrorResponse{Message: message})
}

// TODO: add json validator
func registerWalletsRoutes(rr *gin.Engine, db *pgxpool.Pool) {
	rr.GET("/api/v1/wallets", func(c *gin.Context) {
		ctx := c.Request.Context()

		list, err := wallets.GetWallets(ctx, db)
		if err != nil {
			log.Println(err)
			writeError(c, internalErrorMessage, http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, list)
	})

	rr.GET("/api/v1/wallets/:id", func(c *gin.Context) {
		ctx := c.Request.Context()

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			writeError(c, "invalid uuid", http.StatusBadRequest)
			return
		}

		wallet, err := wallets.GetWallet(ctx, db, id)
		if err != nil {
			switch {
			case errors.Is(err, wallets.ErrWalletNotFound):
				writeError(c, err.Error(), http.StatusNotFound)
			default:
				log.Println(err)
				writeError(c, internalErrorMessage, http.StatusInternalServerError)
			}

			return
		}

		c.JSON(http.StatusOK, wallet)
	})

	rr.POST("/api/v1/wallets", func(c *gin.Context) {
		ctx := c.Request.Context()

		wallet, err := wallets.CreateWallet(ctx, db)
		if err != nil {
			log.Println(err)
			writeError(c, internalErrorMessage, http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusCreated, wallet)
	})

	rr.POST("/api/v1/wallets/:id/transaction", func(c *gin.Context) {
		ctx := c.Request.Context()

		// parse id and params
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			log.Println(err)
			writeError(c, "invalid uuid", http.StatusBadRequest)
			return
		}

		var params wallets.TransactionParams

		if err := c.ShouldBindJSON(&params); err != nil {
			writeError(c, "invalid json", http.StatusBadRequest)
			return
		}

		// validate params
		switch params.OperationType {
		case wallets.OperationTypeDeposit, wallets.OperationTypeWithdraw:
		default:
			errorString := fmt.Sprintf(
				"the operation type must be one of ('%s', '%s')",
				wallets.OperationTypeDeposit,
				wallets.OperationTypeWithdraw,
			)
			writeError(c, errorString, http.StatusBadRequest)
			return
		}

		if params.Amount < 1 || params.Amount > 1e6 {
			writeError(c, "the amount must be between 1 and 1000000", http.StatusBadRequest)
			return
		}

		// do transaction
		err = wallets.DoTransaction(ctx, db, id, &params)
		if err != nil {
			switch {
			case errors.Is(err, wallets.ErrWalletNotFound):
				writeError(c, err.Error(), http.StatusNotFound)
			case errors.Is(err, wallets.ErrWalletNegativeBalance):
				writeError(c, err.Error(), http.StatusBadRequest)
			default:
				log.Println(err)
				writeError(c, internalErrorMessage, http.StatusInternalServerError)
			}

			return
		}

		c.Status(http.StatusNoContent)
	})
}
