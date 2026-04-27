// Placeholder для Шага 2.0. Сервис биндится на порт, отдаёт 401 на всё, кроме
// /health (200) — нужен для compose-healthcheck. Постепенно вытесняется реальной
// логикой в Шаге 3 (TDD-цикл: модуль за модулем заменяет хендлеры).
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	addr := os.Getenv("SERVICE_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte(`{"code":"NOT_IMPLEMENTED","message":"endpoint not implemented yet, see Шаг 3 in backlog.md"}`))
	})

	log.Printf("placeholder service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
