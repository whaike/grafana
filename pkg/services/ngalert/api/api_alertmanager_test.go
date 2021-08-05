package api

import (
	"net/http"
	"testing"

	api "github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
	"github.com/grafana/grafana/pkg/services/ngalert/notifier"
	"github.com/stretchr/testify/require"
)

func TestStatusForTestReceivers(t *testing.T) {
	t.Run("assert 400 Bad Request has precedence over 408 Request Timeout", func(t *testing.T) {
		require.Equal(t, http.StatusBadRequest, statusForTestReceivers([]api.TestReceiverResult{{
			Name: "test1",
			Configs: []api.TestReceiverConfigResult{{
				Name:   "test1",
				UID:    "uid1",
				Status: "failed",
				Error:  notifier.InvalidReceiverError{},
			}},
		}, {
			Name: "test2",
			Configs: []api.TestReceiverConfigResult{{
				Name:   "test2",
				UID:    "uid2",
				Status: "failed",
				Error:  notifier.ReceiverTimeoutError{},
			}},
		}}))
	})
}
