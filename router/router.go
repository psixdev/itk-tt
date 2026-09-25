package router

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Init(db *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()

	registerWalletsRoutes(mux, db)

	return registerMiddleware(mux)
}
