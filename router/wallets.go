package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"itk-tt/services/wallets"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const internalErrorMessage = "internal server error"

type ErrorResponse struct {
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	json.NewEncoder(w).Encode(ErrorResponse{Message: message})
}

// TODO: add json validator
func registerWalletsRoutes(mux *http.ServeMux, db *pgxpool.Pool) {
	mux.HandleFunc("GET /api/v1/wallets", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		list, err := wallets.GetWallets(ctx, db)
		if err != nil {
			log.Println(err)
			writeError(w, internalErrorMessage, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)
	})

	mux.HandleFunc("GET /api/v1/wallets/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			writeError(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		wallet, err := wallets.GetWallet(ctx, db, id)
		if err != nil {
			switch {
			case errors.Is(err, wallets.ErrWalletNotFound):
				writeError(w, err.Error(), http.StatusNotFound)
			default:
				log.Println(err)
				writeError(w, internalErrorMessage, http.StatusInternalServerError)
			}

			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wallet)
	})

	mux.HandleFunc("POST /api/v1/wallets", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		wallet, err := wallets.CreateWallet(ctx, db)
		if err != nil {
			log.Println(err)
			writeError(w, internalErrorMessage, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(wallet)
	})

	mux.HandleFunc("POST /api/v1/wallets/{id}/transaction", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// parse id and params
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			log.Println(err)
			writeError(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		var params wallets.TransactionParams

		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			writeError(w, "invalid json", http.StatusBadRequest)
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
			writeError(w, errorString, http.StatusBadRequest)
			return
		}

		if params.Amount < 1 || params.Amount > 1e6 {
			writeError(w, "the amount must be between 1 and 1000000", http.StatusBadRequest)
			return
		}

		// do transaction
		err = wallets.DoTransaction(ctx, db, id, &params)
		if err != nil {
			switch {
			case errors.Is(err, wallets.ErrWalletNotFound):
				writeError(w, err.Error(), http.StatusNotFound)
			case errors.Is(err, wallets.ErrWalletNegativeBalance):
				writeError(w, err.Error(), http.StatusBadRequest)
			default:
				log.Println(err)
				writeError(w, internalErrorMessage, http.StatusInternalServerError)
			}

			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
