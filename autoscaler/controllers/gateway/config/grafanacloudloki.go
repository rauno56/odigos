package config

import (
	"fmt"
	"net/url"
	"strings"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	grafanaCloudLokiEndpointKey = "GRAFANA_CLOUD_LOKI_ENDPOINT"
	grafanaCloudLokiUsernameKey = "GRAFANA_CLOUD_LOKI_USERNAME"
	grafanaCloudLokiLabelsKey   = "GRAFANA_CLOUD_LOKI_LABELS"
)

type GrafanaCloudLoki struct{}

func (g *GrafanaCloudLoki) DestType() common.DestinationType {
	return common.GrafanaCloudLokiDestinationType
}

func (g *GrafanaCloudLoki) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {

	if !isLoggingEnabled(dest) {
		log.Log.V(0).Info("Logging not enabled, gateway will not be configured for grafana cloud Loki")
		return
	}

	lokiUrl, exists := dest.Spec.Data[grafanaCloudLokiEndpointKey]
	if !exists {
		log.Log.V(0).Info("Grafana Cloud Loki endpoint not specified, gateway will not be configured for Loki")
		return
	}

	lokiExporterEndpoint, err := grafanaLokiUrlFromInput(lokiUrl)
	if err != nil {
		log.Log.Error(err, "failed to parse grafana cloud loki endpoint, gateway will not be configured for Loki")
		return
	}

	lokiUsername, exists := dest.Spec.Data[grafanaCloudLokiUsernameKey]
	if !exists {
		log.Log.V(0).Info("Grafana Cloud Loki username not specified, gateway will not be configured for Loki")
		return
	}

	rawLokiLabels, exists := dest.Spec.Data[grafanaCloudLokiLabelsKey]
	lokiProcessors, err := lokiLabelsProcessors(rawLokiLabels, exists, dest.Name)
	if err != nil {
		log.Log.Error(err, "failed to parse grafana cloud loki labels, gateway will not be configured for Loki")
		return
	}

	authExtensionName := "basicauth/grafana" + dest.Name
	currentConfig.Extensions[authExtensionName] = commonconf.GenericMap{
		"client_auth": commonconf.GenericMap{
			"username": lokiUsername,
			"password": "${GRAFANA_CLOUD_LOKI_PASSWORD}",
		},
	}

	exporterName := "loki/grafana-" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"endpoint": lokiExporterEndpoint,
		"auth": commonconf.GenericMap{
			"authenticator": authExtensionName,
		},
	}

	processorNames := []string{}
	for k, v := range lokiProcessors {
		currentConfig.Processors[k] = v
		processorNames = append(processorNames, k)
	}

	logsPipelineName := "logs/grafana-" + dest.Name
	currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, authExtensionName)
	currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
		Processors: processorNames,
		Exporters:  []string{exporterName},
	}

}

// to send logs to grafana cloud we use the loki exporter, which uses a loki
// endpoint url like: "https://logs-prod-012.grafana.net/loki/api/v1/push"
// Unfortunately, the grafana cloud website does not provide this url in
// an easily parseable format, so we have to parse it ourselves.
//
// the grafana account page provides the url in the form:
//   - "https://logs-prod-012.grafana.net"
//     for data source, in which case we need to append the path
//   - "https://<User Id>:<Your Grafana.com API Token>@logs-prod-012.grafana.net/loki/api/v1/push"
//     for promtail, in which case we need error as we are expecting this info to be provided as input fields
//
// this function will attempt to parse and prepare the url for use with the
// otelcol loki exporter
func grafanaLokiUrlFromInput(rawUrl string) (string, error) {

	rawUrl = strings.TrimSpace(rawUrl)
	urlWithScheme := rawUrl

	// the user should provide the url with the scheme, but if they don't we add it ourselves
	if !strings.Contains(rawUrl, "://") {
		urlWithScheme = "https://" + rawUrl
	}

	parsedUrl, err := url.Parse(urlWithScheme)
	if err != nil {
		return "", err
	}

	if parsedUrl.Scheme != "https" {
		return "", fmt.Errorf("unexpected scheme %s, only https is supported", parsedUrl.Scheme)
	}

	if parsedUrl.Path == "" {
		parsedUrl.Path = "/loki/api/v1/push"
	}
	if parsedUrl.Path != "/loki/api/v1/push" {
		return "", fmt.Errorf("unexpected path for loki endpoint %s", parsedUrl.Path)
	}

	// the username and password should be givin as input fields, and not coded into the url
	if parsedUrl.User != nil {
		return "", fmt.Errorf("unexpected user info for loki endpoint url %s", parsedUrl.User)
	}

	return parsedUrl.String(), nil
}
