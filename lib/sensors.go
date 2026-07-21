package mplmsensors

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

const defaultTimeout = 10 * time.Second

var readingNamePattern = regexp.MustCompile(`^(temp|fan|in|power|energy|curr|humidity)[0-9]+_(input|average)$`)

type commandRunner func(context.Context, string, ...string) ([]byte, error)

// LMSensorsPlugin collects current hardware sensor readings from sensors(1).
type LMSensorsPlugin struct {
	Prefix  string
	Command string
	Timeout time.Duration
	run     commandRunner
}

// MetricKeyPrefix implements mackerelplugin.PluginWithPrefix.
func (p LMSensorsPlugin) MetricKeyPrefix() string {
	if p.Prefix == "" {
		return "lm_sensors"
	}
	return p.Prefix
}

// GraphDefinition implements mackerelplugin.PluginWithPrefix.
func (p LMSensorsPlugin) GraphDefinition() map[string]mp.Graphs {
	return map[string]mp.Graphs{
		"temperature.#": graph("Temperature", mp.UnitFloat, "Temperature (Celsius)"),
		"fan.#":         graph("Fan speed", mp.UnitInteger, "Speed (RPM)"),
		"voltage.#":     graph("Voltage", mp.UnitFloat, "Voltage (V)"),
		"power.#":       graph("Power", mp.UnitFloat, "Power (W)"),
		"energy.#":      graph("Energy", mp.UnitFloat, "Energy (J)"),
		"current.#":     graph("Current", mp.UnitFloat, "Current (A)"),
		"humidity.#":    graph("Humidity", mp.UnitPercentage, "Humidity (%)"),
	}
}

func graph(label, unit, metricLabel string) mp.Graphs {
	return mp.Graphs{
		Label: label,
		Unit:  unit,
		Metrics: []mp.Metrics{
			{Name: "value", Label: metricLabel},
		},
	}
}

// FetchMetrics implements mackerelplugin.PluginWithPrefix.
func (p LMSensorsPlugin) FetchMetrics() (map[string]float64, error) {
	command := p.Command
	if command == "" {
		command = "sensors"
	}

	timeout := p.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	runner := p.run
	if runner == nil {
		runner = runCommand
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	output, err := runner(ctx, command, "-j")
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("sensors command timed out after %s", timeout)
		}
		message := strings.TrimSpace(string(output))
		if message == "" {
			return nil, fmt.Errorf("run sensors command: %w", err)
		}
		return nil, fmt.Errorf("run sensors command: %w: %s", err, message)
	}

	metrics, err := parseSensorOutput(output)
	if err != nil {
		return nil, fmt.Errorf("parse sensors JSON output: %w", err)
	}
	return metrics, nil
}

func runCommand(ctx context.Context, command string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, command, args...).CombinedOutput()
}

func parseSensorOutput(data []byte) (map[string]float64, error) {
	var chips map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &chips); err != nil {
		return nil, err
	}

	metrics := make(map[string]float64)
	for chipName, chip := range chips {
		for featureName, rawFeature := range chip {
			var readings map[string]json.RawMessage
			if err := json.Unmarshal(rawFeature, &readings); err != nil {
				// Entries such as "Adapter" are strings rather than sensor readings.
				continue
			}

			for readingName, rawValue := range readings {
				category, ok := readingCategory(readingName)
				if !ok || strings.EqualFold(strings.TrimSpace(string(rawValue)), "null") {
					continue
				}

				var value float64
				if err := json.Unmarshal(rawValue, &value); err != nil {
					continue
				}

				identifier := sanitizeMetricPart(strings.Join([]string{chipName, featureName, readingName}, "_"))
				key := category + "." + identifier + ".value"
				if _, exists := metrics[key]; exists {
					return nil, fmt.Errorf("sensor names produce duplicate metric key %q", key)
				}
				metrics[key] = value
			}
		}
	}
	return metrics, nil
}

func readingCategory(name string) (string, bool) {
	match := readingNamePattern.FindStringSubmatch(name)
	if match == nil {
		return "", false
	}

	switch match[1] {
	case "temp":
		return "temperature", true
	case "fan":
		return "fan", true
	case "in":
		return "voltage", true
	case "power":
		return "power", true
	case "energy":
		return "energy", true
	case "curr":
		return "current", true
	case "humidity":
		return "humidity", true
	default:
		return "", false
	}
}

func sanitizeMetricPart(value string) string {
	var b strings.Builder
	previousUnderscore := false
	for _, r := range strings.ToLower(value) {
		valid := r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		if valid {
			b.WriteRune(r)
			previousUnderscore = false
			continue
		}
		if !previousUnderscore {
			b.WriteByte('_')
			previousUnderscore = true
		}
	}

	result := strings.Trim(b.String(), "_")
	if result == "" {
		return "sensor"
	}
	return result
}

// Do parses command-line options and runs the plugin.
func Do() {
	optPrefix := flag.String("metric-key-prefix", "lm_sensors", "Metric key prefix")
	optCommand := flag.String("sensors-command", "sensors", "Path to the sensors command")
	optTimeout := flag.Duration("timeout", defaultTimeout, "Timeout for the sensors command")
	optVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *optVersion {
		fmt.Printf("Version: mackerel-plugin-lm-sensors %s\nCompiler: %s %s\n", Version, runtime.Compiler, runtime.Version())
		return
	}

	plugin := LMSensorsPlugin{
		Prefix:  *optPrefix,
		Command: *optCommand,
		Timeout: *optTimeout,
	}
	mp.NewMackerelPlugin(plugin).Run()
}
