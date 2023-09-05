package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/showwin/speedtest-go/speedtest"
	"log"
	"net/http"
	"time"
)

type speedTestCollector struct {
	latencyMetric *prometheus.Desc
	dLSpeedMetric *prometheus.Desc
	uLSpeedMetric *prometheus.Desc
	latencyChan   *chan time.Duration
	dLSpeedChan   *chan float64
	uLSpeedChan   *chan float64
	latencyLast   time.Duration
	dLSpeedLast   float64
	uLSpeedLast   float64
}

func newSpeedTestCollector() *speedTestCollector {
	c1 := make(chan time.Duration, 1)
	c2 := make(chan float64, 1)
	c3 := make(chan float64, 1)

	return &speedTestCollector{
		latencyMetric: prometheus.NewDesc("latency", "measured latency", nil, nil),
		dLSpeedMetric: prometheus.NewDesc("download_speed", "download speed", nil, nil),
		uLSpeedMetric: prometheus.NewDesc("upload_speed", "upload speed", nil, nil),
		latencyChan:   &c1,
		dLSpeedChan:   &c2,
		uLSpeedChan:   &c3,
		latencyLast:   0,
		dLSpeedLast:   0.0,
		uLSpeedLast:   0.0,
	}
}

func (collector *speedTestCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.latencyMetric
	ch <- collector.dLSpeedMetric
	ch <- collector.uLSpeedMetric
}

func (collector *speedTestCollector) Collect(ch chan<- prometheus.Metric) {

	var latency time.Duration
	var DLSpeed float64
	var ULSpeed float64

	select {
	case latency = <-*collector.latencyChan:
	case <-time.After(time.Second / 10):
		latency = collector.latencyLast
	}
	collector.latencyLast = latency

	select {
	case DLSpeed = <-*collector.dLSpeedChan:
	case <-time.After(time.Second / 10):
		DLSpeed = collector.dLSpeedLast
	}
	collector.dLSpeedLast = DLSpeed

	select {
	case ULSpeed = <-*collector.uLSpeedChan:
	case <-time.After(time.Second / 10):
		ULSpeed = collector.uLSpeedLast
	}
	collector.uLSpeedLast = ULSpeed

	m1 := prometheus.MustNewConstMetric(collector.latencyMetric, prometheus.GaugeValue, float64(latency))
	m2 := prometheus.MustNewConstMetric(collector.dLSpeedMetric, prometheus.GaugeValue, DLSpeed)
	m3 := prometheus.MustNewConstMetric(collector.uLSpeedMetric, prometheus.GaugeValue, ULSpeed)

	timeNow := time.Now()
	m1 = prometheus.NewMetricWithTimestamp(timeNow, m1)
	m2 = prometheus.NewMetricWithTimestamp(timeNow, m2)
	m3 = prometheus.NewMetricWithTimestamp(timeNow, m3)

	ch <- m1
	ch <- m2
	ch <- m3
}

func main() {
	spdTst := newSpeedTestCollector()

	go func() {
		var latency time.Duration = 0
		var dLSpeed float64 = 0
		var uLSpeed float64 = 0

		for {
			var speedtestClient = speedtest.New()
			serverList, _ := speedtestClient.FetchServers()
			targets, _ := serverList.FindServer([]int{})
			// fmt.Println(targets)

			s := targets[0]
			err := s.PingTest(nil)
			if err != nil {
				continue
			}
			latency = s.Latency

			err = s.DownloadTest()
			if err != nil {
				continue
			}
			dLSpeed = s.DLSpeed

			err = s.UploadTest()
			if err != nil {
				continue
			}
			uLSpeed = s.ULSpeed

			*spdTst.latencyChan <- latency
			*spdTst.dLSpeedChan <- dLSpeed
			*spdTst.uLSpeedChan <- uLSpeed

			time.Sleep(5 * time.Minute)
		}
	}()

	prometheus.MustRegister(spdTst)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9101", nil))
}
