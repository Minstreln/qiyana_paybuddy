package routers

import (
	"net/http"
)

func MainRouter() *http.ServeMux {

	mux := http.NewServeMux()

	uRouter := usersRouter()
	mux.Handle("/users/", uRouter)

	gRouter := groupsRouter()
	mux.Handle("/groups/", gRouter)

	wRouter := walletRouter()
	mux.Handle("/wallet/", wRouter)

	eRouter := groupExpenseRouter()
	mux.Handle("/group-expense/", eRouter)

	return mux
}
