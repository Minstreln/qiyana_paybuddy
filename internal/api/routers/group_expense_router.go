package routers

import (
	"net/http"
	"qiyana_paybuddy/internal/api/handlers/groups"
)

func groupExpenseRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/group-expense/create", groups.CreateGroupExpenseHandler)

	mux.HandleFunc("/group-expense/{id}/expenses", groups.GetGroupExpensesHandler)

	mux.HandleFunc("/group-expense/details/{id}/expense", groups.GetExpenseByIdHandler)

	mux.HandleFunc("/group-expense/{id}/update", groups.UpdateGroupExpensesHandler)

	mux.HandleFunc("/group-expense/member/balance", groups.GetUserBalanceSummaryHandler)

	mux.HandleFunc("/group-expense/{id}/balance", groups.GetUserBalanceSummaryHandler)

	mux.HandleFunc("/group-expense/{split_id}/settle", groups.SettleExpenseSplitHandler)

	mux.HandleFunc("/group-expense/delete/{expense_id}/expense", groups.DeleteExpenseHandler)

	return mux
}
