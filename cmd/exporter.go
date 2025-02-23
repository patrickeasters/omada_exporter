package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/charlie-haley/omada_exporter/pkg/api"
	"github.com/charlie-haley/omada_exporter/pkg/collector"
	"github.com/charlie-haley/omada_exporter/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var version = "development"

var conf = config.Config{}

func Run() {
	app := cli.NewApp()
	app.Name = "omada_exporter"
	app.Version = version
	app.Usage = "Prometheus Exporter for TP-Link Omada Controller SDN."
	app.EnableBashCompletion = true
	app.Authors = []*cli.Author{
		{Name: "Charlie Haley", Email: "charlie-haley@users.noreply.github.com"},
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{Destination: &conf.Host, Required: true, Name: "host", Value: "", Usage: "The hostname of the Omada Controller, including protocol.", EnvVars: []string{"OMADA_HOST"}},
		&cli.StringFlag{Destination: &conf.Username, Required: true, Name: "username", Value: "", Usage: "Username of the Omada user you'd like to use to fetch metrics.", EnvVars: []string{"OMADA_USER"}},
		&cli.StringFlag{Destination: &conf.Password, Required: true, Name: "password", Value: "", Usage: "Password for your Omada user.", EnvVars: []string{"OMADA_PASS"}},
		&cli.StringFlag{Destination: &conf.Port, Name: "port", Value: "9202", Usage: "Port on which to expose the Prometheus metrics.", EnvVars: []string{"OMADA_PORT"}},
		&cli.StringFlag{Destination: &conf.Site, Name: "site", Value: "Default", Usage: "Omada site to scrape metrics from.", EnvVars: []string{"OMADA_SITE"}},
		&cli.IntFlag{Destination: &conf.Timeout, Name: "timeout", Value: 15, Usage: "Timeout when making requests to the Omada Controller.", EnvVars: []string{"OMADA_REQUEST_TIMEOUT"}},
		&cli.BoolFlag{Destination: &conf.Insecure, Name: "insecure", Value: false, Usage: "Whether to skip verifying the SSL certificate on the controller.", EnvVars: []string{"OMADA_INSECURE"}},
		&cli.BoolFlag{Destination: &conf.GoCollectorDisabled, Name: "disable-go-collector", Value: true, Usage: "Disable Go collector metrics.", EnvVars: []string{"OMADA_DISABLE_GO_COLLECTOR"}},
		&cli.BoolFlag{Destination: &conf.ProcessCollectorDisabled, Name: "disable-process-collector", Value: true, Usage: "Disable process collector metrics.", EnvVars: []string{"OMADA_DISABLE_PROCESS_COLLECTOR"}},
	}
	app.Commands = []*cli.Command{
		{Name: "version", Aliases: []string{"v"}, Usage: "prints the current version.",
			Action: func(c *cli.Context) error {
				fmt.Println(version)
				os.Exit(0)
				return nil
			}},
	}
	app.Action = run

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	if conf.GoCollectorDisabled {
		// remove Go collector
		prometheus.Unregister(prometheus.NewGoCollector())
	}

	if conf.ProcessCollectorDisabled {
		// remove Process collector
		prometheus.Unregister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}

	// check if host is properly formatted
	if strings.HasSuffix(conf.Host, "/") {
		// remove trailing slash if it exists
		conf.Host = strings.TrimRight(conf.Host, "/")
	}

	client, err := api.Configure(&conf)
	if err != nil {
		return err
	}

	// register omada collectors
	prometheus.MustRegister(collector.NewClientCollector(client))
	prometheus.MustRegister(collector.NewControllerCollector(client))
	prometheus.MustRegister(collector.NewDeviceCollector(client))
	prometheus.MustRegister(collector.NewPortCollector(client))

	log.Info(fmt.Sprintf("listening on :%s", conf.Port))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
    <head>
	<title>omada_exporter</title>
	</head>
    	<body>
			<h1>omada_exporter</h1>
			<p>
				<a href="/metrics">Metrics</a>
			</p>
    	</body>
    </html>`))
	})

	http.Handle("/metrics", promhttp.Handler())
	err = http.ListenAndServe(fmt.Sprintf(":%s", conf.Port), nil)
	if err != nil {
		return err
	}

	return nil
}
