package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var exporterUp = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "redfish_exporter_up",
	Help: "1, пока процесс экспортёра жив.",
})

func init() {
	prometheus.MustRegister(exporterUp)
}

func main() {
	exporterUp.Set(1)
	http.Handle("/metrics", promhttp.Handler())

	addr := ":9812"
	log.Printf("redfish-exporter listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
