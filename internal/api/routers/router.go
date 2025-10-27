package routers

import (
	"net/http"
)

func MainRouter() *http.ServeMux {
	mux := http.NewServeMux()

	apiMux := http.NewServeMux()

	apiMux.Handle("/users/", usersRouter())

	apiMux.Handle("/groups/", groupsRouter())

	apiMux.Handle("/wallet/", walletRouter())

	apiMux.Handle("/group-expense/", groupExpenseRouter())

	apiMux.Handle("/transactions/", transactionsRouter())

	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", apiMux))

	return mux
}
