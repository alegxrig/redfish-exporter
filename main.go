package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stmcginnis/gofish"
)

type reading struct {
	labels []string
	value  float64
}

type snapshot struct {
	power  map[string]float64 // system -> 1=On, 0=Off
	health map[string]float64 // system -> 1=OK, 0=не OK
	temps  []reading
}

type RedfishCollector struct {
	mu   sync.RWMutex
	data snapshot

	powerDesc  *prometheus.Desc
	healthDesc *prometheus.Desc
	tempDesc   *prometheus.Desc
}

func NewRedfishCollector() *RedfishCollector {
	return &RedfishCollector{
		powerDesc: prometheus.NewDesc(
			"redfish_system_power_on",
			"1, если PowerState == On.",
			[]string{"system"}, nil,
		),
		healthDesc: prometheus.NewDesc(
			"redfish_system_health_ok",
			"1, если Health == OK.",
			[]string{"system"}, nil,
		),
		tempDesc: prometheus.NewDesc(
			"redfish_temperature_celsius",
			"Показание датчика температуры.",
			[]string{"chassis", "sensor"}, nil,
		),
	}
}

func (c *RedfishCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.powerDesc
	ch <- c.healthDesc
	ch <- c.tempDesc
}

// Collect читает кэш под RLock — сеть сюда не попадает никогда.
func (c *RedfishCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for name, v := range c.data.power {
		ch <- prometheus.MustNewConstMetric(c.powerDesc, prometheus.GaugeValue, v, name)
	}
	for name, v := range c.data.health {
		ch <- prometheus.MustNewConstMetric(c.healthDesc, prometheus.GaugeValue, v, name)
	}
	for _, r := range c.data.temps {
		ch <- prometheus.MustNewConstMetric(c.tempDesc, prometheus.GaugeValue, r.value, r.labels...)
	}
}

// poll — единственное место, где мы ходим в сеть. Пишет в кэш под Lock.
func (c *RedfishCollector) poll(endpoint, user, pass string) {
	config := gofish.ClientConfig{
		Endpoint:  endpoint,
		Username:  user,
		Password:  pass,
		Insecure:  true,
		BasicAuth: true, // на реальном BMC — false, там нормальные Redfish-сессии
	}

	client, err := gofish.Connect(config)
	if err != nil {
		log.Printf("redfish poll: connect: %v", err)
		return
	}
	defer client.Logout()

	service := client.Service
	newSnap := snapshot{power: make(map[string]float64), health: make(map[string]float64)}

	systems, err := service.Systems()
	if err != nil {
		log.Printf("redfish poll: systems: %v", err)
		return
	}
	for _, sys := range systems {
		if sys.PowerState == "On" {
			newSnap.power[sys.Name] = 1
		}
		if sys.Status.Health == "OK" {
			newSnap.health[sys.Name] = 1
		}
	}

	chassisList, err := service.Chassis()
	if err != nil {
		log.Printf("redfish poll: chassis: %v", err)
		return
	}
	for _, ch2 := range chassisList {
		thermal, err := ch2.Thermal()
		if err != nil {
			continue
		}
		for _, t := range thermal.Temperatures {
			if t.ReadingCelsius == nil {
				continue
			}
			newSnap.temps = append(newSnap.temps, reading{
				labels: []string{ch2.Name, t.Name},
				value:  *t.ReadingCelsius,
			})
		}
	}

	c.mu.Lock()
	c.data = newSnap
	c.mu.Unlock()
}

// Run делает первый опрос синхронно (чтобы /metrics не был пустым сразу после старта),
// затем продолжает по тикеру в фоне.
func (c *RedfishCollector) Run(endpoint, user, pass string, interval time.Duration) {
	c.poll(endpoint, user, pass)
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			c.poll(endpoint, user, pass)
		}
	}()
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var exporterUp = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "redfish_exporter_up",
	Help: "1, пока процесс экспортёра жив.",
})

func main() {
	exporterUp.Set(1)
	prometheus.MustRegister(exporterUp)

	collector := NewRedfishCollector()
	prometheus.MustRegister(collector)

	endpoint := getenv("REDFISH_ENDPOINT", "http://127.0.0.1:8000")
	user := getenv("REDFISH_USER", "root")
	pass := getenv("REDFISH_PASSWORD", "password")

	collector.Run(endpoint, user, pass, 15*time.Second)

	http.Handle("/metrics", promhttp.Handler())
	addr := ":9812"
	log.Printf("redfish-exporter listening on %s (polling %s every 15s)", addr, endpoint)
	log.Fatal(http.ListenAndServe(addr, nil))
}
