package routers

import (
	"net/http"
	"qiyana_paybuddy/internal/api/handlers/wallet"
)

func walletRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/wallet/fund", wallet.FundWallet)

	mux.HandleFunc("/wallet/webhook", wallet.PaystackWebhook)

	return mux
}
