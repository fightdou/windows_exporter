package collector

import (
	"strconv"
	"strings"

	ps "github.com/bhendo/go-powershell"
	"github.com/bhendo/go-powershell/backend"
	"github.com/prometheus-community/windows_exporter/log"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerCollector("cpu_temperature", newCpuTemperature)
}

type CpuTemperatureCollector struct {
	CpuTemperature *prometheus.Desc
}

func newCpuTemperature() (Collector, error) {
	return &CpuTemperatureCollector{
		CpuTemperature: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "", "cpu_temperature"),
			"The CPU temperature",
			[]string{"virt"},
			nil,
		),
	}, nil
}

func (c *CpuTemperatureCollector) Collect(ctx *ScrapeContext, ch chan<- prometheus.Metric) error {
	if desc, err := c.collect(ctx.instance, ch); err != nil {
		log.Error("failed collecting cpu_temperature metrics:", desc, err)
		return err
	}
	return nil
}

func (c *CpuTemperatureCollector) collect(uuid string, ch chan<- prometheus.Metric) (*prometheus.Desc, error) {
	back := &backend.Local{}
	shell, err := ps.New(back)
	if err != nil {
		log.Error("open powershell process failed:", err)
		return nil, err
	}
	defer shell.Exit()

	stdout, _, err := shell.Execute("get-wmiobject -namespace root\\OpenHardwareMonitor -query 'select value,name,Parent from Sensor where SensorType=\"Temperature\" and Name LIKE \"%CPU Package%\"'")
	if err != nil {
		log.Error("Exec powershell command failed:", err)
		return nil, err
	}

	result := strings.TrimSpace(stdout)

	res := strings.Split(result, "\n")
	value := ""
	for _, line := range res {
		key := strings.Split(line, ":")
		if strings.TrimSpace(key[0]) == "Value" {
			value = strings.TrimSpace(key[1])
		}
	}

	v, _ := strconv.ParseFloat(value, 64)
	ch <- prometheus.MustNewConstMetric(
		c.CpuTemperature,
		prometheus.GaugeValue,
		v,
		uuid,
	)
	return nil, nil
}
