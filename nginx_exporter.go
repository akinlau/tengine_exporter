package main

import (
	"crypto/tls"
	"flag"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/log"
)

const (
	namespace = "nginx" // For Prometheus metrics.
	exporter  = "exporter"
)

var (
	listeningAddress = flag.String("telemetry.address", ":9113", "Address on which to expose metrics.")
	metricsEndpoint  = flag.String("telemetry.endpoint", "/metrics", "Path under which to expose metrics.")
	nginxScrapeURI   = flag.String("nginx.scrape_uri", "http://localhost/nginx_status", "URI to nginx stub status page")
	insecure         = flag.Bool("insecure", true, "Ignore server certificate if using https")
)

var landingPage = []byte(`<html>
<head><title>Nginx Exporter</title></head>
<body>
<h1>Nginx Exporter</h1>
<p><a href="` + *metricsEndpoint + `">Metrics</a></p>
</body>
</html>`)

// Exporter collects nginx stats from the given URI and exports them using
// the prometheus metrics package.
type Exporter struct {
	URI    string
	mutex  sync.RWMutex
	client *http.Client

	error        prometheus.Gauge
	scrapeErrors *prometheus.CounterVec
	nginxUp      prometheus.Gauge
	raise        *prometheus.GaugeVec
	fail         *prometheus.GaugeVec
}

// NewExporter returns an initialized Exporter.
func NewExporter(uri string) *Exporter {
	return &Exporter{
		URI: uri,
		scrapeErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: exporter,
			Name:      "scrape_errors_total",
			Help: "Number 	of errors while scraping nginx.",
		}, []string{"collector"}),
		nginxUp: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Whether the Nginx server is up.",
		}),
		raise: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "raise",
			Help:      "Number of raise status.",
		}, []string{"upstream", "name", "status"}),
		fail: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "fail",
			Help:      "Number of fail status.",
		}, []string{"upstream", "name", "status"}),
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: *insecure},
			},
		},
	}
}

// Describe describes all the metrics ever exported by the nginx exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.nginxUp.Describe(ch)
	e.raise.Describe(ch)
	e.fail.Describe(ch)
	e.scrapeErrors.Describe(ch)
}

// Collect fetches the stats from configured nginx location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.scrape(ch)
	e.raise.Collect(ch)
	e.fail.Collect(ch)
	e.scrapeErrors.Collect(ch)
	ch <- e.nginxUp
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) {
	resp, err := e.client.Get(e.URI)
	if err != nil {
		log.Errorln("Error calling nginx status API: ", err)
		e.nginxUp.Set(0)
	}

	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		if err != nil {
			data = []byte(err.Error())
		}
		log.Warnf("Status %s (%d): %s", resp.Status, resp.StatusCode, data)
		e.nginxUp.Set(0)
	}

	e.nginxUp.Set(1)

	// Parsing results
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if len(line) <= 0 {
			continue
		}
		cols := strings.Split(line, ",")
		upstream := cols[1]
		name := cols[2]
		status := cols[3]
		raise := cols[4]
		fail := cols[5]
		raiseCount, err := strconv.Atoi(raise)
		if err != nil {
			log.Errorln("Error parsing raise count: ", err)
			e.scrapeErrors.WithLabelValues("raise").Inc()
		} else {
			e.raise.WithLabelValues(upstream, name, status).Set(float64(raiseCount))
		}

		failCount, err := strconv.Atoi(fail)
		if err != nil {
			log.Errorln("Error parsing fail count: ", err)
			e.scrapeErrors.WithLabelValues("fail").Inc()
		} else if failCount != 0 {
			e.raise.WithLabelValues(upstream, name, status).Set(float64(failCount))
		}
	}
}

func main() {
	flag.Parse()

	exporter := NewExporter(*nginxScrapeURI)
	prometheus.MustRegister(exporter)

	http.Handle(*metricsEndpoint, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(landingPage)
	})

	log.Infoln("Listening on", *listeningAddress)
	log.Fatal(http.ListenAndServe(*listeningAddress, nil))
}
