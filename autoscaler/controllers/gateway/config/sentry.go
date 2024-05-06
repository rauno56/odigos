package config

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Sentry struct{}

func (s *Sentry) DestType() common.DestinationType {
	return common.SentryDestinationType
}

func (s *Sentry) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if !isTracingEnabled(dest) {
		log.Log.V(0).Info("Sentry is not enabled for any supported signals, skipping")
		return
	}

	if isTracingEnabled(dest) {
		exporterName := "sentry/" + dest.Name
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"dsn": "${DSN}",
		}

		tracesPipelineName := "traces/sentry-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}
}
