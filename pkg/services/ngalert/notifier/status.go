package notifier

import (
	apimodels "github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
)

func (am *Alertmanager) GetStatus() apimodels.GettableStatus {
	am.reloadConfigMtx.RLock()
	defer am.reloadConfigMtx.RUnlock()
	return *apimodels.NewGettableStatus(&am.config.AlertmanagerConfig)
}
