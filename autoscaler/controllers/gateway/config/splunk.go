package config

import (
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	splunkRealm = "SPLUNK_REALM"
)

type Splunk struct{}

func (s *Splunk) DestType() common.DestinationType {
	return common.SplunkDestinationType
}

func (s *Splunk) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	realm, exists := dest.Spec.Data[splunkRealm]
	if !exists {
		log.Log.V(0).Info("Splunk realm not specified, gateway will not be configured for Splunk")
		return
	}

	if isTracingEnabled(dest) {
		exporterName := "sapm/" + dest.Name
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"access_token": "${SPLUNK_ACCESS_TOKEN}",
			"endpoint":     fmt.Sprintf("https://ingest.%s.signalfx.com/v2/trace", realm),
		}

		tracesPipelineName := "traces/splunk-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}
}
