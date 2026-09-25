package wallets

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Wallet struct {
	ID        uuid.UUID `json:"id"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

type OperationType string

const (
	OperationTypeDeposit  OperationType = "deposit"
	OperationTypeWithdraw OperationType = "withdraw"
)

type TransactionParams struct {
	OperationType OperationType `json:"operationType"`
	Amount        int64         `json:"amount"`
}

var (
	ErrWalletNotFound        = errors.New("wallet not found")
	ErrWalletNegativeBalance = errors.New("balance can't be negative")
)

// TODO: allocate a "repository" abstraction layer as the project grows
func GetWallets(ctx context.Context, db *pgxpool.Pool) ([]Wallet, error) {
	wallets := make([]Wallet, 0)

	rows, err := db.Query(ctx, `
		select id, balance, created_at
		from wallets
		order by id asc
	`)
	if err != nil {
		return nil, fmt.Errorf("select query error: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var wallet Wallet

		if err := rows.Scan(
			&wallet.ID,
			&wallet.Balance,
			&wallet.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("row scan error: %w", err)
		}

		wallets = append(wallets, wallet)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows scan error: %w", err)
	}

	return wallets, nil
}

func GetWallet(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*Wallet, error) {
	var wallet Wallet

	err := db.QueryRow(ctx, `
		select id, balance, created_at
		from wallets
		where id = ($1)
	`, id).Scan(
		&wallet.ID,
		&wallet.Balance,
		&wallet.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}

		return nil, fmt.Errorf("select query error: %w", err)
	}

	return &wallet, nil
}

func CreateWallet(ctx context.Context, db *pgxpool.Pool) (*Wallet, error) {
	var wallet Wallet

	ID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate uuid error: %w", err)
	}

	err = db.QueryRow(ctx, `
		insert into wallets (id)
		values ($1)
		returning id, balance, created_at;
	`, ID).Scan(
		&wallet.ID,
		&wallet.Balance,
		&wallet.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert query error: %w", err)
	}

	return &wallet, nil
}

func DoTransaction(
	ctx context.Context,
	db *pgxpool.Pool,
	id uuid.UUID,
	params *TransactionParams,
) error {
	change := params.Amount

	if params.OperationType == OperationTypeWithdraw {
		change = -change
	}

	result, err := db.Exec(ctx, `
		update wallets
		set balance = balance + $1
		where id = $2
	`, change, id)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23514" &&
			pgErr.ConstraintName == "balance_non_negative" {
			return ErrWalletNegativeBalance
		}

		return fmt.Errorf("update query error: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrWalletNotFound
	}

	return nil
}
