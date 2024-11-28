package main

import (
	"context"
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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg, err := safeconfig.New(*configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("設定の作成中にエラーが発生しました")
	}
	if len(cfg.Domains) == 0 {
		log.Error().Msg("プローブするドメインがありません --config は少なくとも1つのドメインを含む必要があります")
		os.Exit(1)
	}

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	c := client.NewMultiClient(rdap.NewClient(), whois.NewClient())

	domainCollector := collector.NewDomainCollector(c, *timeout*time.Duration(len(cfg.Domains)), cfg.Domains...)
	prometheus.MustRegister(domainCollector)

	prometheusClient, err := promclient.NewClient(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create prometheus client")
	}

	err = collectAndSendMetrics(prometheusClient)
	if err != nil {
		log.Error().Err(err).Msg("failed to collect metrics")
	} else {
		log.Info().Msg("successfully output metrics to stdout")
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
