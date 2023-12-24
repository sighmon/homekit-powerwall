package promexporter

import (
	"flag"
	"net/http"
	"text/template"

	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	batteryGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "battery",
			Help: "Charge (%)",
		},
	)

	index = template.Must(template.New("index").Parse(
		`<!doctype html>
	 <title>Tesla Powerwall Prometheus Exporter</title>
	 <h1>Tesla Powerwall Prometheus Exporter</h1>
	 <a href="/metrics">Metrics</a>
	 <p>
	 `))
)

type Exporter struct {
	address string
}

func New(address string) *Exporter {
	return &Exporter{address: address}
}

func (e *Exporter) Start() {
	flag.Parse()
	log.Printf("Prometheus Exporter starting on port %s\n", e.address)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		index.Execute(w, "")
	})
	if err := http.ListenAndServe(e.address, nil); err != http.ErrServerClosed {
		panic(err)
	}
}

func (e *Exporter) UpdateReadings(battery float64) {
	batteryGauge.Set(battery)
}
