package main

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestCollect_PowerOn(t *testing.T) {
	c := NewRedfishCollector()
	c.data = snapshot{
		power:  map[string]float64{"srv1": 1},
		health: map[string]float64{"srv1": 1},
	}

	expected := `
		# HELP redfish_system_power_on 1, если PowerState == On.
		# TYPE redfish_system_power_on gauge
		redfish_system_power_on{system="srv1"} 1
		# HELP redfish_system_health_ok 1, если Health == OK.
		# TYPE redfish_system_health_ok gauge
		redfish_system_health_ok{system="srv1"} 1
	`
	if err := testutil.CollectAndCompare(c, strings.NewReader(expected),
		"redfish_system_power_on", "redfish_system_health_ok"); err != nil {
		t.Fatalf("unexpected metrics: %v", err)
	}
}

func TestCollect_PowerOff(t *testing.T) {
	c := NewRedfishCollector()
	c.data = snapshot{
		power: map[string]float64{"srv1": 0},
	}

	expected := `
		# HELP redfish_system_power_on 1, если PowerState == On.
		# TYPE redfish_system_power_on gauge
		redfish_system_power_on{system="srv1"} 0
	`
	if err := testutil.CollectAndCompare(c, strings.NewReader(expected), "redfish_system_power_on"); err != nil {
		t.Fatalf("unexpected metrics: %v", err)
	}
}

func TestCollect_Temperature(t *testing.T) {
	c := NewRedfishCollector()
	c.data = snapshot{
		temps: []reading{
			{labels: []string{"Chassis1", "CPU1 Temp"}, value: 41},
		},
	}

	expected := `
		# HELP redfish_temperature_celsius Показание датчика температуры.
		# TYPE redfish_temperature_celsius gauge
		redfish_temperature_celsius{chassis="Chassis1",sensor="CPU1 Temp"} 41
	`
	if err := testutil.CollectAndCompare(c, strings.NewReader(expected), "redfish_temperature_celsius"); err != nil {
		t.Fatalf("unexpected metrics: %v", err)
	}
}
