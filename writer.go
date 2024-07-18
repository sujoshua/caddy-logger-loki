package caddy_logger_loki

import (
	"github.com/grafana/loki/v3/clients/pkg/promtail/api"
	"github.com/grafana/loki/v3/clients/pkg/promtail/client"
	"github.com/grafana/loki/v3/pkg/logproto"
	"github.com/prometheus/common/model"
	"time"
)

type LokiWriter struct {
	client client.Client
	logger logger
	send   chan<- api.Entry
	lbs    model.LabelSet
}

func newLokiWriter(client client.Client, logger logger, labels map[string]string) *LokiWriter {
	lbs := model.LabelSet{}
	for k, v := range labels {
		lbs[model.LabelName(k)] = model.LabelValue(v)
	}

	return &LokiWriter{
		client: client,
		logger: logger,
		send:   client.Chan(),
		lbs:    lbs,
	}
}

func (w *LokiWriter) Write(p []byte) (n int, err error) {
	entry := api.Entry{
		Labels: w.lbs.Clone(),
		Entry: logproto.Entry{
			Timestamp: time.Now(),
			Line:      string(p),
		},
	}
	w.send <- entry

	return len(p), nil
}

func (w *LokiWriter) Close() error {
	w.client.StopNow()
	return nil
}
