package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"itk-tt/config"
	"itk-tt/db"
	"itk-tt/services/wallets"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testDB     *pgxpool.Pool
	testRouter *gin.Engine
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	os.Setenv("APP_ENV", "test")

	conf, err := config.Init("../config.env")
	if err != nil {
		panic(err)
	}

	testDB, err = db.Init(ctx, conf)
	if err != nil {
		panic(err)
	}

	_, err = testDB.Exec(ctx, `truncate table wallets`)
	if err != nil {
		panic(err)
	}

	gin.SetMode(gin.TestMode)
	testRouter = Init(testDB)

	code := m.Run()

	testDB.Close()
	os.Exit(code)
}

func TestCreateWallet(t *testing.T) {
	// create wallet
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/wallets",
		nil,
	)
	rec := httptest.NewRecorder()

	testRouter.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var wallet wallets.Wallet

	err := json.Unmarshal(rec.Body.Bytes(), &wallet)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, wallet.ID)
	assert.Equal(t, int64(0), wallet.Balance)

	// check in db
	var balance int64

	err = testDB.QueryRow(
		t.Context(),
		`select balance from wallets where id = $1`,
		wallet.ID,
	).Scan(&balance)

	require.NoError(t, err)
	assert.Equal(t, int64(0), balance)
}

func TestGetWallet(t *testing.T) {
	// create wallet
	id, err := uuid.NewV7()
	require.NoError(t, err)

	_, err = testDB.Exec(
		t.Context(),
		`insert into wallets (id, balance) values ($1, $2)`,
		id,
		1000,
	)
	require.NoError(t, err)

	// get wallet
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/wallets/"+id.String(),
		nil,
	)
	rec := httptest.NewRecorder()

	testRouter.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var wallet wallets.Wallet
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wallet))

	assert.Equal(t, id, wallet.ID)
	assert.Equal(t, int64(1000), wallet.Balance)
}

func TestTransaction(t *testing.T) {
	// create wallet
	id, err := uuid.NewV7()
	require.NoError(t, err)

	_, err = testDB.Exec(
		t.Context(),
		`insert into wallets (id, balance) values ($1, $2)`,
		id,
		1000,
	)
	require.NoError(t, err)

	var balance int64

	// deposit
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/wallets/"+id.String()+"/transaction",
		strings.NewReader(`{"operationType": "deposit", "amount": 500}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	testRouter.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)

	// check balance in db
	err = testDB.QueryRow(
		t.Context(),
		`select balance from wallets where id = $1`,
		id,
	).Scan(&balance)

	require.NoError(t, err)
	assert.Equal(t, int64(1500), balance)

	// withdraw
	req = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/wallets/"+id.String()+"/transaction",
		strings.NewReader(`{"operationType": "withdraw", "amount": 300}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec = httptest.NewRecorder()

	testRouter.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)

	// check balance in db
	err = testDB.QueryRow(
		t.Context(),
		`select balance from wallets where id = $1`,
		id,
	).Scan(&balance)

	require.NoError(t, err)
	assert.Equal(t, int64(1200), balance)
}

func TestNegativeBalance(t *testing.T) {
	// create wallet
	id, err := uuid.NewV7()
	require.NoError(t, err)

	_, err = testDB.Exec(
		t.Context(),
		`insert into wallets (id, balance) values ($1, $2)`,
		id,
		1000,
	)
	require.NoError(t, err)

	// try to withdraw
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/wallets/"+id.String()+"/transaction",
		strings.NewReader(`{"operationType": "withdraw", "amount": 1500}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	testRouter.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	// check balance in db
	var balance int64

	err = testDB.QueryRow(
		t.Context(),
		`select balance from wallets where id = $1`,
		id,
	).Scan(&balance)

	require.NoError(t, err)
	assert.Equal(t, int64(1000), balance)
}

func TestNotFound(t *testing.T) {
	id, err := uuid.NewV7()
	require.NoError(t, err)

	// try to get
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/wallets/"+id.String(),
		nil,
	)
	rec := httptest.NewRecorder()

	testRouter.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)

	// try to deposit
	body := `{"operationType": "deposit", "amount": 100}`

	req = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/wallets/"+id.String()+"/transaction",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rec = httptest.NewRecorder()

	testRouter.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTransactionValidation(t *testing.T) {
	id, err := uuid.NewV7()
	require.NoError(t, err)

	cases := []struct {
		name string
		body string
	}{
		{
			name: "invalid json",
			body: `not a json`,
		},
		{
			name: "invalid operation type",
			body: `{"operationType": "foobar", "amount": 100}`,
		},
		{
			name: "amount is zero",
			body: `{"operationType": "deposit", "amount": 0}`,
		},
		{
			name: "amount is too large",
			body: `{"operationType": "deposit", "amount": 1000001}`,
		},
	}

	// check
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/wallets/"+id.String()+"/transaction",
				strings.NewReader(testCase.body),
			)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			testRouter.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestConcurrentTransactions(t *testing.T) {
	id, err := uuid.NewV7()
	require.NoError(t, err)

	// create wallet
	_, err = testDB.Exec(
		t.Context(),
		`insert into wallets (id, balance) values ($1, 0)`,
		id,
	)
	require.NoError(t, err)

	// check
	const count = 1000

	var wg sync.WaitGroup
	wg.Add(count)

	for range count {
		go func() {
			defer wg.Done()

			body := `{"operationType": "deposit", "amount": 1}`

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/wallets/"+id.String()+"/transaction",
				strings.NewReader(body),
			)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			testRouter.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusNoContent, rec.Code)
		}()
	}

	wg.Wait()

	// check balance in db
	var balance int64

	err = testDB.QueryRow(
		t.Context(),
		`select balance from wallets where id = $1`,
		id,
	).Scan(&balance)

	require.NoError(t, err)
	assert.Equal(t, int64(count), balance)
}
