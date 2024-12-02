package main

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/caarlos0/domain_exporter/internal/client"
	"github.com/caarlos0/domain_exporter/internal/collector"
	promclient "github.com/caarlos0/domain_exporter/internal/prometheus"
	"github.com/caarlos0/domain_exporter/internal/rdap"
	"github.com/caarlos0/domain_exporter/internal/safeconfig"
	"github.com/caarlos0/domain_exporter/internal/whois"
	"github.com/castai/promwrite"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	timeout    = kingpin.Flag("timeout", "timeout for each domain").Default("10s").Duration()
	configFile = kingpin.Flag("config", "configuration file").Default("config.yml").String()
	version    = "dev"
)

func main() {
	kingpin.Version("domain_exporter version " + version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	cfg, err := safeconfig.New(*configFile)
	if err != nil {
		log.Println("failed to create config", err)
		os.Exit(1)
	}
	if len(cfg.Domains) == 0 {
		log.Println("no domains to probe --config must contain at least one domain")
		os.Exit(1)
	}

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	c := client.NewMultiClient(rdap.NewClient(), whois.NewClient())

	domainCollector := collector.NewDomainCollector(c, *timeout*time.Duration(len(cfg.Domains)), cfg.Domains...)
	prometheus.MustRegister(domainCollector)

	prometheusClient, err := promclient.NewClient(cfg)
	if err != nil {
		log.Println("failed to create prometheus client", err)
		os.Exit(1)
	}

	err = collectAndSendMetrics(prometheusClient)
	if err != nil {
		log.Println("failed to collect metrics", err)
		os.Exit(1)
	} else {
		log.Println("finished")
	}
}

func collectAndSendMetrics(promClient *promwrite.Client) error {
	gatherer := prometheus.DefaultGatherer
	metricFamilies, err := gatherer.Gather()
	if err != nil {
		return err
	}

	data := []promwrite.TimeSeries{}

	for _, mf := range metricFamilies {
		if !strings.Contains(mf.GetName(), "domain_") {
			continue
		}

		if len(mf.GetMetric()) == 0 {
			continue
		}

		for _, metric := range mf.GetMetric() {
			labels := metric.GetLabel()
			var domainLabelValue string
			for _, label := range labels {
				if label.GetName() == "domain" {
					domainLabelValue = label.GetValue()
					break
				}
			}
			if domainLabelValue == "" {
				continue
			}

			data = append(data, promwrite.TimeSeries{
				Labels: []promwrite.Label{
					{Name: "__name__", Value: mf.GetName()},
					{Name: "domain", Value: domainLabelValue},
				},
				Sample: promwrite.Sample{
					Time:  time.Now(),
					Value: metric.GetGauge().GetValue(),
				},
			})
		}
	}

	_, err = promClient.Write(context.TODO(), &promwrite.WriteRequest{TimeSeries: data})
	if err != nil {
		return err
	}

	return nil
}
