//go:build windows
// +build windows

package collector

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	var deps string
	// See below for 6.05 magic value
	if getWindowsVersion() > 6.05 {
		deps = "Processor Information"
	} else {
		deps = "Processor"
	}
	registerCollector("cpu", newCPUCollector, deps)
}

type cpuCollectorBasic struct {
	CStateSecondsTotal *prometheus.Desc
	TimeTotal          *prometheus.Desc
	InterruptsTotal    *prometheus.Desc
	DPCsTotal          *prometheus.Desc
}
type cpuCollectorFull struct {
	CStateSecondsTotal       *prometheus.Desc
	TimeTotal                *prometheus.Desc
	InterruptsTotal          *prometheus.Desc
	DPCsTotal                *prometheus.Desc
	ClockInterruptsTotal     *prometheus.Desc
	IdleBreakEventsTotal     *prometheus.Desc
	ParkingStatus            *prometheus.Desc
	ProcessorFrequencyMHz    *prometheus.Desc
	ProcessorMaxFrequencyMHz *prometheus.Desc
	ProcessorPerformance     *prometheus.Desc
}

// newCPUCollector constructs a new cpuCollector, appropriate for the running OS
func newCPUCollector() (Collector, error) {
	const subsystem = "cpu"

	version := getWindowsVersion()
	// For Windows 2008 (version 6.0) or earlier we only have the "Processor"
	// class. As of Windows 2008 R2 (version 6.1) the more detailed
	// "Processor Information" set is available (although some of the counters
	// are added in later versions, so we aren't guaranteed to get all of
	// them).
	// Value 6.05 was selected to split between Windows versions.
	if version < 6.05 {
		return &cpuCollectorBasic{
			CStateSecondsTotal: prometheus.NewDesc(
				prometheus.BuildFQName(Namespace, subsystem, "cstate_seconds_total"),
				"Time spent in low-power idle state",
				[]string{"core", "state", "virt"},
				nil,
			),
			TimeTotal: prometheus.NewDesc(
				prometheus.BuildFQName(Namespace, subsystem, "time_total"),
				"Time that processor spent in different modes (dpc, idle, interrupt, privileged, user)",
				[]string{"core", "mode", "virt"},
				nil,
			),
			InterruptsTotal: prometheus.NewDesc(
				prometheus.BuildFQName(Namespace, subsystem, "interrupts_total"),
				"Total number of received and serviced hardware interrupts",
				[]string{"core", "virt"},
				nil,
			),
			DPCsTotal: prometheus.NewDesc(
				prometheus.BuildFQName(Namespace, subsystem, "dpcs_total"),
				"Total number of received and serviced deferred procedure calls (DPCs)",
				[]string{"core", "virt"},
				nil,
			),
		}, nil
	}

	return &cpuCollectorFull{
		CStateSecondsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, subsystem, "cstate_seconds_total"),
			"Time spent in low-power idle state",
			[]string{"core", "state", "virt"},
			nil,
		),
		TimeTotal: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, subsystem, "time_total"),
			"Time that processor spent in different modes (dpc, idle, interrupt, privileged, user)",
			[]string{"core", "mode", "virt"},
			nil,
		),
		InterruptsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, subsystem, "interrupts_total"),
			"Total number of received and serviced hardware interrupts",
			[]string{"core", "virt"},
			nil,
		),
		DPCsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, subsystem, "dpcs_total"),
			"Total number of received and serviced deferred procedure calls (DPCs)",
			[]string{"core", "virt"},
			nil,
		),
		ClockInterruptsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, subsystem, "clock_interrupts_total"),
			"Total number of received and serviced clock tick interrupts",
			[]string{"core", "virt"},
			nil,
		),
		IdleBreakEventsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, subsystem, "idle_break_events_total"),
			"Total number of time processor was woken from idle",
			[]string{"core", "virt"},
			nil,
		),
		ParkingStatus: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, subsystem, "parking_status"),
			"Parking Status represents whether a processor is parked or not",
			[]string{"core", "virt"},
			nil,
		),
		ProcessorFrequencyMHz: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, subsystem, "core_frequency_mhz"),
			"Core frequency in megahertz",
			[]string{"core", "virt"},
			nil,
		),
		ProcessorPerformance: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, subsystem, "processor_performance"),
			"Processor Performance is the average performance of the processor while it is executing instructions, as a percentage of the nominal performance of the processor. On some processors, Processor Performance may exceed 100%",
			[]string{"core", "virt"},
			nil,
		),
	}, nil
}

type perflibProcessor struct {
	Name                  string
	C1Transitions         float64 `perflib:"C1 Transitions/sec"`
	C2Transitions         float64 `perflib:"C2 Transitions/sec"`
	C3Transitions         float64 `perflib:"C3 Transitions/sec"`
	DPCRate               float64 `perflib:"DPC Rate"`
	DPCsQueued            float64 `perflib:"DPCs Queued/sec"`
	Interrupts            float64 `perflib:"Interrupts/sec"`
	PercentC2Time         float64 `perflib:"% C1 Time"`
	PercentC3Time         float64 `perflib:"% C2 Time"`
	PercentC1Time         float64 `perflib:"% C3 Time"`
	PercentDPCTime        float64 `perflib:"% DPC Time"`
	PercentIdleTime       float64 `perflib:"% Idle Time"`
	PercentInterruptTime  float64 `perflib:"% Interrupt Time"`
	PercentPrivilegedTime float64 `perflib:"% Privileged Time"`
	PercentProcessorTime  float64 `perflib:"% Processor Time"`
	PercentUserTime       float64 `perflib:"% User Time"`
}

func (c *cpuCollectorBasic) Collect(ctx *ScrapeContext, ch chan<- prometheus.Metric) error {
	data := make([]perflibProcessor, 0)
	err := unmarshalObject(ctx.perfObjects["Processor"], &data)
	if err != nil {
		return err
	}
	uuid := ctx.instance
	for _, cpu := range data {
		if strings.Contains(strings.ToLower(cpu.Name), "_total") {
			continue
		}
		core := cpu.Name

		ch <- prometheus.MustNewConstMetric(
			c.CStateSecondsTotal,
			prometheus.CounterValue,
			cpu.PercentC1Time,
			core, "c1", uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.CStateSecondsTotal,
			prometheus.CounterValue,
			cpu.PercentC2Time,
			core, "c2", uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.CStateSecondsTotal,
			prometheus.CounterValue,
			cpu.PercentC3Time,
			core, "c3", uuid,
		)

		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.CounterValue,
			cpu.PercentIdleTime,
			core, "idle", uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.CounterValue,
			cpu.PercentInterruptTime,
			core, "interrupt", uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.CounterValue,
			cpu.PercentDPCTime,
			core, "dpc", uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.CounterValue,
			cpu.PercentPrivilegedTime,
			core, "privileged", uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.CounterValue,
			cpu.PercentUserTime,
			core, "user", uuid,
		)

		ch <- prometheus.MustNewConstMetric(
			c.InterruptsTotal,
			prometheus.CounterValue,
			cpu.Interrupts,
			core, uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.DPCsTotal,
			prometheus.CounterValue,
			cpu.DPCsQueued,
			core, uuid,
		)
	}

	return nil
}

type perflibProcessorInformation struct {
	Name                     string
	C1TimeSeconds            float64 `perflib:"% C1 Time"`
	C2TimeSeconds            float64 `perflib:"% C2 Time"`
	C3TimeSeconds            float64 `perflib:"% C3 Time"`
	C1TransitionsTotal       float64 `perflib:"C1 Transitions/sec"`
	C2TransitionsTotal       float64 `perflib:"C2 Transitions/sec"`
	C3TransitionsTotal       float64 `perflib:"C3 Transitions/sec"`
	ClockInterruptsTotal     float64 `perflib:"Clock Interrupts/sec"`
	DPCsQueuedTotal          float64 `perflib:"DPCs Queued/sec"`
	DPCTimeSeconds           float64 `perflib:"% DPC Time"`
	IdleBreakEventsTotal     float64 `perflib:"Idle Break Events/sec"`
	IdleTimeSeconds          float64 `perflib:"% Idle Time"`
	InterruptsTotal          float64 `perflib:"Interrupts/sec"`
	InterruptTimeSeconds     float64 `perflib:"% Interrupt Time"`
	ParkingStatus            float64 `perflib:"Parking Status"`
	PerformanceLimitPercent  float64 `perflib:"% Performance Limit"`
	PriorityTimeSeconds      float64 `perflib:"% Priority Time"`
	PrivilegedTimeSeconds    float64 `perflib:"% Privileged Time"`
	PrivilegedUtilitySeconds float64 `perflib:"% Privileged Utility"`
	ProcessorFrequencyMHz    float64 `perflib:"Processor Frequency"`
	ProcessorPerformance     float64 `perflib:"% Processor Performance"`
	ProcessorTimeSeconds     float64 `perflib:"% Processor Time"`
	ProcessorUtilityRate     float64 `perflib:"% Processor Utility"`
	UserTimeSeconds          float64 `perflib:"% User Time"`
}

func (c *cpuCollectorFull) Collect(ctx *ScrapeContext, ch chan<- prometheus.Metric) error {
	data := make([]perflibProcessorInformation, 0)
	err := unmarshalObject(ctx.perfObjects["Processor Information"], &data)
	if err != nil {
		return err
	}
	uuid := ctx.instance
	for _, cpu := range data {
		if strings.Contains(strings.ToLower(cpu.Name), "_total") {
			continue
		}
		core := cpu.Name

		ch <- prometheus.MustNewConstMetric(
			c.CStateSecondsTotal,
			prometheus.CounterValue,
			cpu.C1TimeSeconds,
			core, "c1", uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.CStateSecondsTotal,
			prometheus.CounterValue,
			cpu.C2TimeSeconds,
			core, "c2", uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.CStateSecondsTotal,
			prometheus.CounterValue,
			cpu.C3TimeSeconds,
			core, "c3", uuid,
		)

		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.CounterValue,
			cpu.IdleTimeSeconds,
			core, "idle", uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.CounterValue,
			cpu.InterruptTimeSeconds,
			core, "interrupt", uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.CounterValue,
			cpu.DPCTimeSeconds,
			core, "dpc", uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.CounterValue,
			cpu.PrivilegedTimeSeconds,
			core, "privileged", uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.CounterValue,
			cpu.UserTimeSeconds,
			core, "user", uuid,
		)

		ch <- prometheus.MustNewConstMetric(
			c.InterruptsTotal,
			prometheus.CounterValue,
			cpu.InterruptsTotal,
			core, uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.DPCsTotal,
			prometheus.CounterValue,
			cpu.DPCsQueuedTotal,
			core, uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.ClockInterruptsTotal,
			prometheus.CounterValue,
			cpu.ClockInterruptsTotal,
			core, uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.IdleBreakEventsTotal,
			prometheus.CounterValue,
			cpu.IdleBreakEventsTotal,
			core, uuid,
		)

		ch <- prometheus.MustNewConstMetric(
			c.ParkingStatus,
			prometheus.GaugeValue,
			cpu.ParkingStatus,
			core, uuid,
		)

		ch <- prometheus.MustNewConstMetric(
			c.ProcessorFrequencyMHz,
			prometheus.GaugeValue,
			cpu.ProcessorFrequencyMHz,
			core, uuid,
		)
		ch <- prometheus.MustNewConstMetric(
			c.ProcessorPerformance,
			prometheus.GaugeValue,
			cpu.ProcessorPerformance,
			core, uuid,
		)
	}

	return nil
}
