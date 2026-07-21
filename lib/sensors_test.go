package mplmsensors

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseSensorOutput(t *testing.T) {
	input := []byte(`{
  "coretemp-isa-0000": {
    "Adapter": "ISA adapter",
    "Package id 0": {
      "temp1_input": 42.5,
      "temp1_max": 80.0,
      "temp1_crit": 100.0,
      "temp1_crit_alarm": 0.0
    }
  },
  "nct6798-isa-0290": {
    "Adapter": "ISA adapter",
    "Vcore": {"in0_input": 1.224, "in0_min": 0.0},
    "CPU Fan": {"fan1_input": 1247.0, "fan1_min": 0.0},
    "CPU Power": {"power1_average": 65.25, "power1_input": 67.0},
    "CPU Current": {"curr1_input": 12.5},
    "Humidity": {"humidity1_input": 44.0},
    "Intrusion": {"intrusion0_alarm": 1.0},
    "Unavailable": {"temp9_input": null},
    "Broken": {"temp8_input": "N/A"}
  },
  "energy-meter-acpi-0": {
    "Energy": {"energy1_input": 1234.5}
  }
}`)

	got, err := parseSensorOutput(input)
	if err != nil {
		t.Fatalf("parseSensorOutput() error = %v", err)
	}

	want := map[string]float64{
		"temperature.coretemp_isa_0000_package_id_0_temp1_input.value": 42.5,
		"voltage.nct6798_isa_0290_vcore_in0_input.value":               1.224,
		"fan.nct6798_isa_0290_cpu_fan_fan1_input.value":                1247,
		"power.nct6798_isa_0290_cpu_power_power1_average.value":        65.25,
		"power.nct6798_isa_0290_cpu_power_power1_input.value":          67,
		"current.nct6798_isa_0290_cpu_current_curr1_input.value":       12.5,
		"humidity.nct6798_isa_0290_humidity_humidity1_input.value":     44,
		"energy.energy_meter_acpi_0_energy_energy1_input.value":        1234.5,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseSensorOutput() = %#v, want %#v", got, want)
	}
}

func TestParseSensorOutputRejectsInvalidJSON(t *testing.T) {
	if _, err := parseSensorOutput([]byte(`{"chip":`)); err == nil {
		t.Fatal("parseSensorOutput() error = nil, want an error")
	}
}

func TestParseSensorOutputRejectsDuplicateMetricKeys(t *testing.T) {
	input := []byte(`{
  "chip-a": {"Feature": {"temp1_input": 1}},
  "chip_a": {"Feature": {"temp1_input": 2}}
}`)

	_, err := parseSensorOutput(input)
	if err == nil || !strings.Contains(err.Error(), "duplicate metric key") {
		t.Fatalf("parseSensorOutput() error = %v, want duplicate metric key error", err)
	}
}

func TestFetchMetrics(t *testing.T) {
	var gotCommand string
	var gotArgs []string
	plugin := LMSensorsPlugin{
		Command: "custom-sensors",
		Timeout: time.Second,
		run: func(_ context.Context, command string, args ...string) ([]byte, error) {
			gotCommand = command
			gotArgs = args
			return []byte(`{"chip":{"Temp":{"temp1_input":35.5}}}`), nil
		},
	}

	metrics, err := plugin.FetchMetrics()
	if err != nil {
		t.Fatalf("FetchMetrics() error = %v", err)
	}
	if gotCommand != "custom-sensors" || !reflect.DeepEqual(gotArgs, []string{"-j"}) {
		t.Errorf("runner called with %q, %v", gotCommand, gotArgs)
	}
	if got := metrics["temperature.chip_temp_temp1_input.value"]; got != 35.5 {
		t.Errorf("temperature metric = %v, want 35.5", got)
	}
}

func TestFetchMetricsIncludesCommandOutputInError(t *testing.T) {
	plugin := LMSensorsPlugin{
		run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte("permission denied\n"), errors.New("exit status 1")
		},
	}

	_, err := plugin.FetchMetrics()
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("FetchMetrics() error = %v, want command output", err)
	}
}

func TestMetricKeyPrefix(t *testing.T) {
	if got := (LMSensorsPlugin{}).MetricKeyPrefix(); got != "lm_sensors" {
		t.Errorf("default prefix = %q, want lm_sensors", got)
	}
	if got := (LMSensorsPlugin{Prefix: "hardware"}).MetricKeyPrefix(); got != "hardware" {
		t.Errorf("custom prefix = %q, want hardware", got)
	}
}

func TestSanitizeMetricPart(t *testing.T) {
	tests := map[string]string{
		"Core 0 / Temp": "core_0_temp",
		"  FAN---1  ":   "fan_1",
		"温度":            "sensor",
	}
	for input, want := range tests {
		if got := sanitizeMetricPart(input); got != want {
			t.Errorf("sanitizeMetricPart(%q) = %q, want %q", input, got, want)
		}
	}
}
