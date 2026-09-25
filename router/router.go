package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Init(db *pgxpool.Pool) *gin.Engine {
	rr := gin.New()

	rr.Use(gin.Logger())
	rr.Use(gin.Recovery())

	registerWalletsRoutes(rr, db)

	return rr
}
