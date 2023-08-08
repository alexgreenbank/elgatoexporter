package main

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	prefix = "elgato_keylight" // TODO
)

type Recorder interface {
	measurePolls(state string)
	measureStatusCode(code int)
	measureLastGoodPoll(ts time.Time)
	measureLastPoll(ts time.Time) // Time (UTC) of last poll
	measurePollDur(duration time.Duration)
	measureParseDur(duration time.Duration)
	measureLastError(ts time.Time) // Time (UTC) of last errored poll
	measureOnOff(onoff int)
	measureBrightness(val int)
	measureTemperature(temp int)
}

func NewRecorder(reg prometheus.Registerer) Recorder {
	r := &prometheusRecorder{
		elgatoPolls: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "polls",
			Help:      "Number of polls we have attempted",
		}, []string{"state"}),
		elgatoStatusCodeCount: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "status_code_count",
			Help:      "A count of each status code encountered",
		}, []string{"statusCode"}),
		elgatoLastGoodPollSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: prefix,
			Name:      "last_good_poll_seconds",
			Help:      "The UNIX timestamp in seconds of the last good poll",
		}),
		elgatoLastPollSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: prefix,
			Name:      "last_poll_seconds",
			Help:      "The UNIX timestamp in seconds of the last poll",
		}),
		elgatoPollDurSeconds: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "poll_duration",
			Help:      "The total duration of polling",
		}),
		elgatoParseDurSeconds: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "parse_duration",
			Help:      "The total duration of parsing",
		}),
		elgatoLastErrorTimeSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: prefix,
			Name:      "last_error_time_seconds",
			Help:      "The UNIX timestamp in seconds of the last error",
		}),
		elgatoOnOff: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: prefix,
			Name:      "onoff",
			Help:      "Whether the keylight is on or off",
		}),
		elgatoBrightness: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: prefix,
			Name:      "brightness",
			Help:      "The brightness of the keylight",
		}),
		elgatoTemperature: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: prefix,
			Name:      "temperature",
			Help:      "The temperature of the keylight",
		}),
	}

	reg.MustRegister(r.elgatoPolls, r.elgatoStatusCodeCount, r.elgatoLastGoodPollSeconds,
		r.elgatoLastPollSeconds, r.elgatoPollDurSeconds, r.elgatoParseDurSeconds,
		r.elgatoLastErrorTimeSeconds, r.elgatoOnOff, r.elgatoBrightness, r.elgatoTemperature)
	return r
}

type prometheusRecorder struct {
	elgatoPolls                *prometheus.CounterVec
	elgatoStatusCodeCount      *prometheus.CounterVec
	elgatoLastGoodPollSeconds  prometheus.Gauge
	elgatoLastPollSeconds      prometheus.Gauge
	elgatoPollDurSeconds       prometheus.Counter
	elgatoParseDurSeconds      prometheus.Counter
	elgatoLastErrorTimeSeconds prometheus.Gauge
	elgatoOnOff                prometheus.Gauge
	elgatoBrightness           prometheus.Gauge
	elgatoTemperature          prometheus.Gauge
}

// Count the number of polls we have made
func (r prometheusRecorder) measurePolls(state string) {
	r.elgatoPolls.WithLabelValues(state).Inc()
}

// A count of each status code encountered
func (r prometheusRecorder) measureStatusCode(code int) {
	codeStr := fmt.Sprintf("%d", code)
	r.elgatoStatusCodeCount.WithLabelValues(codeStr).Add(1)
}

// The UNIX timestamp in seconds of the last good poll
func (r prometheusRecorder) measureLastGoodPoll(ts time.Time) {
	r.elgatoLastGoodPollSeconds.Set(float64(ts.Unix()))
}

// The UNIX timestamp in seconds of the last poll
func (r prometheusRecorder) measureLastPoll(ts time.Time) {
	r.elgatoLastPollSeconds.Set(float64(ts.Unix()))
}

// The duration of the poll
func (r prometheusRecorder) measurePollDur(duration time.Duration) {
	r.elgatoPollDurSeconds.Add(duration.Seconds())
}

// The duration of the parse
func (r prometheusRecorder) measureParseDur(duration time.Duration) {
	r.elgatoParseDurSeconds.Add(duration.Seconds())
}

// The UNIX timestamp in seconds of the last error
func (r prometheusRecorder) measureLastError(ts time.Time) {
	r.elgatoLastErrorTimeSeconds.Set(float64(ts.Unix()))
}

// Whether the keylight is on or off
func (r prometheusRecorder) measureOnOff(onoff int) {
	r.elgatoOnOff.Set(float64(onoff))
}

// The brightness of the keylight
func (r prometheusRecorder) measureBrightness(bright int) {
	r.elgatoBrightness.Set(float64(bright))
}

// The temperature of the keylight
func (r prometheusRecorder) measureTemperature(temp int) {
	r.elgatoTemperature.Set(float64(temp))
}
