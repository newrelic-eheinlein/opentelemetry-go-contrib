// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelruntime_test

import (
	goruntime "runtime"
	"testing"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime/otelruntime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/internal/metric"
)

func TestRuntime(t *testing.T) {
	err := otelruntime.Start(
		otelruntime.WithMinimumReadMemStatsInterval(time.Second),
	)
	assert.NoError(t, err)
	time.Sleep(time.Second)
}

func getGCCount(impl *metric.MeterImpl) int {
	for _, b := range impl.MeasurementBatches {
		for _, m := range b.Measurements {
			if m.Instrument.Descriptor().Name() == "runtime.go.gc.count" {
				return int(m.Number.CoerceToInt64(m.Instrument.Descriptor().NumberKind()))
			}
		}
	}
	panic("Could not locate a runtime.go.gc.count metric in test output")
}

func testMinimumInterval(t *testing.T, shouldHappen bool, opts ...otelruntime.Option) {
	goruntime.GC()

	var mstats0 goruntime.MemStats
	goruntime.ReadMemStats(&mstats0)
	baseline := int(mstats0.NumGC)

	impl, provider := metric.NewProvider()

	err := otelruntime.Start(
		append(
			opts,
			otelruntime.WithMeterProvider(provider),
		)...,
	)
	assert.NoError(t, err)

	goruntime.GC()

	impl.RunAsyncInstruments()

	require.Equal(t, 1, getGCCount(impl)-baseline)

	impl.MeasurementBatches = nil

	extra := 0
	if shouldHappen {
		extra = 3
	}

	goruntime.GC()
	goruntime.GC()
	goruntime.GC()

	impl.RunAsyncInstruments()

	require.Equal(t, 1+extra, getGCCount(impl)-baseline)
}

func TestDefaultMinimumInterval(t *testing.T) {
	testMinimumInterval(t, false)
}

func TestNoMinimumInterval(t *testing.T) {
	testMinimumInterval(t, true, otelruntime.WithMinimumReadMemStatsInterval(0))
}

func TestExplicitMinimumInterval(t *testing.T) {
	testMinimumInterval(t, false, otelruntime.WithMinimumReadMemStatsInterval(time.Hour))
}
