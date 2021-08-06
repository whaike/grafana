package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/grafana/pkg/services/ngalert/notifier"
	"github.com/stretchr/testify/require"
)

func TestContextWithTimeoutFromRequest(t *testing.T) {
	t.Run("assert context has default timeout when header is absent", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "https://grafana.net", nil)
		require.NoError(t, err)

		now := time.Now()
		ctx := context.Background()
		ctx, cancelFunc, err := contextWithTimeoutFromRequest(
			ctx,
			req,
			15*time.Second,
			30*time.Second)
		require.NoError(t, err)
		require.NotNil(t, cancelFunc)
		require.NotNil(t, ctx)

		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.True(t, deadline.After(now))
		require.Less(t, deadline.Sub(now).Seconds(), 30.0)
		require.GreaterOrEqual(t, deadline.Sub(now).Seconds(), 15.0)
	})

	t.Run("assert context has timeout in request header", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "https://grafana.net", nil)
		require.NoError(t, err)
		req.Header.Set("Request-Timeout", "5")

		now := time.Now()
		ctx := context.Background()
		ctx, cancelFunc, err := contextWithTimeoutFromRequest(
			ctx,
			req,
			15*time.Second,
			30*time.Second)
		require.NoError(t, err)
		require.NotNil(t, cancelFunc)
		require.NotNil(t, ctx)

		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.True(t, deadline.After(now))
		require.Less(t, deadline.Sub(now).Seconds(), 15.0)
		require.GreaterOrEqual(t, deadline.Sub(now).Seconds(), 5.0)
	})

	t.Run("assert timeout in request header cannot exceed max timeout", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "https://grafana.net", nil)
		require.NoError(t, err)
		req.Header.Set("Request-Timeout", "60")

		ctx := context.Background()
		ctx, cancelFunc, err := contextWithTimeoutFromRequest(
			ctx,
			req,
			15*time.Second,
			30*time.Second)
		require.Error(t, err, "exceeded maximum timeout")
		require.Nil(t, cancelFunc)
		require.Nil(t, ctx)
	})
}

func TestStatusForTestReceivers(t *testing.T) {
	t.Run("assert 400 Bad Request has precedence over 408 Request Timeout", func(t *testing.T) {
		require.Equal(t, http.StatusBadRequest, statusForTestReceivers([]notifier.TestReceiverResult{{
			Name: "test1",
			Configs: []notifier.TestReceiverConfigResult{{
				Name:   "test1",
				UID:    "uid1",
				Status: "failed",
				Error:  notifier.InvalidReceiverError{},
			}},
		}, {
			Name: "test2",
			Configs: []notifier.TestReceiverConfigResult{{
				Name:   "test2",
				UID:    "uid2",
				Status: "failed",
				Error:  notifier.ReceiverTimeoutError{},
			}},
		}}))
	})
}
