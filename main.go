package main

import (
	"context"
	"itk-tt/config"
	"itk-tt/db"
	"itk-tt/router"
	"log"
	"net/http"
)

func main() {
	ctx := context.Background()

	conf, err := config.Init("config.env")
	if err != nil {
		log.Fatalf("couldn't initialize config: %v", err)
	}

	pool, err := db.Init(ctx, conf)
	if err != nil {
		log.Fatalf("couldn't connect to the database:\n%v", err)
	}

	rr := router.Init(pool)

	log.Printf("Server started on :%s\n", conf.App.Port)

	err = http.ListenAndServe(":"+conf.App.Port, rr)
	if err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
