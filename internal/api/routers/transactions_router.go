package routers

import (
	"net/http"
	"qiyana_paybuddy/internal/api/handlers/transactions"
)

func transactionsRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/transactions/user", transactions.GetAllUserTransactions)

	mux.HandleFunc("/transactions/{id}/user", transactions.GetTransactionById)

	return mux
}
