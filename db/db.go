package db

import (
	"context"
	"fmt"
	"itk-tt/config"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Init(ctx context.Context, conf *config.Config) (*pgxpool.Pool, error) {
	connString := getConnString(conf)

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

func getConnString(conf *config.Config) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(conf.DB.User, conf.DB.Password),
		Host:   fmt.Sprintf("%s:%s", conf.DB.Host, conf.DB.Port),
		Path:   conf.DB.Name,
	}

	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()

	return u.String()
}
