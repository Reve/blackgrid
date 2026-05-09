package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "blackgrid_http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	HttpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "blackgrid_http_request_duration_seconds",
		Help:    "Duration of HTTP requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	MonitorChecksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "blackgrid_monitor_checks_total",
		Help: "Total number of monitor checks",
	}, []string{"monitor_type", "status"})

	MonitorCheckDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "blackgrid_monitor_check_duration_seconds",
		Help:    "Duration of monitor checks",
		Buckets: prometheus.DefBuckets,
	}, []string{"monitor_type"})

	IncidentsOpen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "blackgrid_incidents_open",
		Help: "Current number of open incidents",
	})

	DiscoveryScansTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "blackgrid_discovery_scans_total",
		Help: "Total number of discovery scans",
	}, []string{"status"})

	NotificationDeliveriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "blackgrid_notification_deliveries_total",
		Help: "Total number of notification deliveries",
	}, []string{"channel_type", "status"})

	SseClientsCurrent = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "blackgrid_sse_clients_current",
		Help: "Current number of active SSE clients",
	})

	EventBusEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "blackgrid_event_bus_events_total",
		Help: "Total number of events published on the event bus",
	}, []string{"type"})
)
