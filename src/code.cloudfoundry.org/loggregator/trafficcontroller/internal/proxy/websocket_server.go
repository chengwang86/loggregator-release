package proxy

import (
	"log"
	"net/http"
	"time"

	"code.cloudfoundry.org/loggregator/metricemitter"
)

const websocketKeepAliveDuration = 30 * time.Second

type WebSocketServer struct {
	slowConsumerMetric  *metricemitter.Counter
	slowConsumerTimeout time.Duration
}

func NewWebSocketServer(slowConsumerTimeout time.Duration, m MetricClient) *WebSocketServer {
	// metric-documentation-v2: (doppler_proxy.slow_consumer) Counter
	// indicating occurrences of slow consumers.
	slowConsumerMetric := m.NewCounter("doppler_proxy.slow_consumer",
		metricemitter.WithVersion(2, 0),
	)

	return &WebSocketServer{
		slowConsumerMetric:  slowConsumerMetric,
		slowConsumerTimeout: slowConsumerTimeout,
	}
}

func (s *WebSocketServer) ServeWS(
	w http.ResponseWriter,
	r *http.Request,
	recv func() ([]byte, error),
	egressMetric *metricemitter.Counter,
) {
	data := make(chan []byte)

	handler := NewWebsocketHandler(
		data,
		websocketKeepAliveDuration,
		egressMetric,
	)

	go func() {
		defer close(data)
		timer := time.NewTimer(s.slowConsumerTimeout)
		timer.Stop()
		for {
			resp, err := recv()
			if err != nil {
				log.Printf("error receiving from doppler via gRPC %s", err)
				return
			}

			if resp == nil {
				continue
			}

			timer.Reset(s.slowConsumerTimeout)
			select {
			case data <- resp:
				if !timer.Stop() {
					<-timer.C
				}
			case <-timer.C:
				s.slowConsumerMetric.Increment(1)
				log.Print("Doppler Proxy: Slow Consumer")
				return
			}
		}
	}()

	handler.ServeHTTP(w, r)
}
