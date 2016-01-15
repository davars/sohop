package sohop

import (
	"net/http"
	"os"

	"github.com/gorilla/handlers"
)

func notFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not found", http.StatusNotFound)
}

func logging(next http.Handler) http.Handler {
	return handlers.CombinedLoggingHandler(os.Stdout, next)
}
