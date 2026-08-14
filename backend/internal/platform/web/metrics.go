package web

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var requestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{Name: "http_requests_total", Help: "HTTP requests by method and status."},
	[]string{"method", "status"},
)

var moneyOpsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{Name: "money_operations_total", Help: "Money-moving operations by type and outcome."},
	[]string{"operation", "outcome"},
)

func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(moneyOpsTotal)
}

// Metrics counts every request by method and status code.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		requestsTotal.WithLabelValues(r.Method, strconv.Itoa(ww.Status())).Inc()
	})
}

// MetricsHandler serves the Prometheus exposition endpoint.
func MetricsHandler() http.Handler { return promhttp.Handler() }

// RecordMoneyOp counts a money-moving operation (deposit, withdraw, transfer, loan_disbursement,
// loan_repayment) by its outcome ("success" or "failure"), alongside the audit log entry for the same
// event.
func RecordMoneyOp(operation, outcome string) {
	moneyOpsTotal.WithLabelValues(operation, outcome).Inc()
}
