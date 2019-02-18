package exporter

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/camptocamp/prometheus-puppetdb-exporter/internal/puppetdb"
)

// Exporter implements the prometheus.Exporter interface, and exports PuppetDB metrics
type Exporter struct {
	client    *puppetdb.PuppetDB
	namespace string
	metrics   map[string]*prometheus.GaugeVec
}

var (
	metricMap = map[string]string{
		"node_status_count": "node_status_count",
	}
)

// NewPuppetDBExporter returns a new exporter of PuppetDB metrics.
func NewPuppetDBExporter(url, certPath, caPath, keyPath string, sslVerify bool) (e *Exporter, err error) {
	e = &Exporter{
		namespace: "puppetdb",
	}

	opts := &puppetdb.Options{
		URL:        url,
		CertPath:   certPath,
		CACertPath: caPath,
		KeyPath:    keyPath,
		SSLVerify:  sslVerify,
	}

	e.client, err = puppetdb.NewClient(opts)
	if err != nil {
		log.Fatalf("failed to create new client: %s", err)
		return
	}

	e.initGauges()

	return
}

// Describe outputs PuppetDB metric descriptions
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.metrics {
		m.Describe(ch)
	}
}

// Collect fetches new metrics from the PuppetDB and updates the appropriate metrics
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	for _, m := range e.metrics {
		m.Collect(ch)
	}
}

// Scrape scrapes PuppetDB and update metrics
func (e *Exporter) Scrape(interval time.Duration) {
	var statuses map[string]int
	for {
		statuses = make(map[string]int)

		nodes, err := e.client.Nodes()
		if err != nil {
			log.Errorf("failed to get nodes: %s", err)
		}
		for _, node := range nodes {
			statuses[node.LatestReportStatus]++
		}
		for statusName, statusValue := range statuses {
			e.metrics["node_report_status_count"].With(prometheus.Labels{"status": statusName}).Set(float64(statusValue))
		}

		time.Sleep(interval)
	}
	return
}

func (e *Exporter) initGauges() {
	e.metrics = map[string]*prometheus.GaugeVec{}

	e.metrics["node_report_status_count"] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: e.namespace,
		Name:      "node_report_status_count",
		Help:      "Total count of reports status by type",
	}, []string{"status"})

	for _, m := range e.metrics {
		prometheus.MustRegister(m)
	}
}