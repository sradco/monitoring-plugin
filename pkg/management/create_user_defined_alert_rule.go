package management

import (
	"strings"

	"github.com/openshift/monitoring-plugin/pkg/managementlabels"
)

func filterBusinessLabels(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		if strings.HasPrefix(k, "openshift_io_") || k == managementlabels.AlertNameLabel {
			continue
		}
		out[k] = v
	}
	return out
}
