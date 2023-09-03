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
}

func newSpeedTestCollector() *speedTestCollector {
	return &speedTestCollector{
		latencyMetric: prometheus.NewDesc("latency", "measured latency", nil, nil),
		dLSpeedMetric: prometheus.NewDesc("download_speed", "download speed", nil, nil),
		uLSpeedMetric: prometheus.NewDesc("upload_speed", "upload speed", nil, nil),
	}
}

func (collector *speedTestCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.latencyMetric
	ch <- collector.dLSpeedMetric
	ch <- collector.uLSpeedMetric
}

func (collector *speedTestCollector) Collect(ch chan<- prometheus.Metric) {
	latency, DLSpeed, ULSpeed, err := speedTest()
	if err != nil {
		log.Fatalf("Error: %v\n", err.Error())
	}

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

func speedTest() (time.Duration, float64, float64, error) {
	var latency time.Duration = 0
	var dLSpeed float64 = 0
	var uLSpeed float64 = 0

	var speedtestClient = speedtest.New()
	serverList, _ := speedtestClient.FetchServers()
	targets, _ := serverList.FindServer([]int{})
	// fmt.Println(targets)

	s := targets[0]
	err := s.PingTest(nil)
	if err != nil {
		return latency, dLSpeed, uLSpeed, err
	}
	latency = s.Latency

	err = s.DownloadTest()
	if err != nil {
		return latency, dLSpeed, uLSpeed, err
	}
	dLSpeed = s.DLSpeed

	err = s.UploadTest()
	if err != nil {
		return latency, dLSpeed, uLSpeed, err
	}
	uLSpeed = s.ULSpeed

	return latency, dLSpeed, uLSpeed, nil
}

func main() {
	spdTst := newSpeedTestCollector()
	prometheus.MustRegister(spdTst)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9101", nil))
}
