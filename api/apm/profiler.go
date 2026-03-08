package apm

import (
	"os"

	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

// StartProfiling starts Datadog continuous profiling (CPU, heap, goroutine,
// mutex profiles) and returns a stop function that must be deferred in main().
//
// Profiling is only activated when DD_PROFILING_ENABLED=true.
// Profiles are shipped to the Datadog agent at DD_AGENT_HOST:8126, which
// defaults to the "dd-agent" service name on the shared docker network.
//
// Profiling is independent of tracing — it works whether or not APM_ENABLED
// is set, and uses the same service/env/version tags for correlation.
func StartProfiling(cfg Config) (func(), error) {
	if os.Getenv("DD_PROFILING_ENABLED") != "true" {
		return func() {}, nil
	}

	agentHost := os.Getenv("DD_AGENT_HOST")
	if agentHost == "" {
		agentHost = "dd-agent"
	}

	err := profiler.Start(
		profiler.WithService(cfg.ServiceName),
		profiler.WithEnv(cfg.Environment),
		profiler.WithVersion(cfg.Version),
		profiler.WithAgentAddr(agentHost+":8126"),
		profiler.WithProfileTypes(
			profiler.CPUProfile,
			profiler.HeapProfile,
			profiler.GoroutineProfile,
			profiler.MutexProfile,
		),
	)
	if err != nil {
		return func() {}, err
	}

	return profiler.Stop, nil
}
