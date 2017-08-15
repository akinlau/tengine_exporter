package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	nginxStatus = `0,us1,10.1.0.1:80,up,8247,0,tcp,0
1,us1,10.1.0.2:80,up,8251,0,tcp,0
2,us2,10.1.0.3:80,up,8251,0,tcp,0
3,us2,10.1.0.4:80,up,8247,0,tcp,0
4,us2,10.1.0.5:80,up,7918,0,tcp,0
`
	// 5 status and 1 up
	metricCount = 6
)

func TestNginxStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(nginxStatus))
	})
	server := httptest.NewServer(handler)

	e := NewExporter(server.URL)
	ch := make(chan prometheus.Metric)

	go func() {
		defer close(ch)
		e.Collect(ch)
	}()

	for i := 1; i <= metricCount; i++ {
		m := <-ch
		if m == nil {
			t.Error("expected metric but got nil")
		}
	}
	if <-ch != nil {
		t.Error("expected closed channel")
	}
}
