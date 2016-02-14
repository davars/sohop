package sohop

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

var healthClient = createInsecureClient()

func createInsecureClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	return client
}

type healthStatus struct {
	Response  string        `json:"response"`
	LatencyMS time.Duration `json:"latency_ms"`
}

func health(conf *Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allOk := true
		responses := make(map[string]healthStatus)

		var lock sync.Mutex // responses
		var wg sync.WaitGroup

		for _k, _v := range conf.Upstreams {
			k := _k
			v := _v

			wg.Add(1)
			go func() {
				defer wg.Done()

				healthCheck := v.HealthCheck
				if healthCheck == "" {
					healthCheck = v.URL
				}

				start := time.Now()
				resp, err := healthClient.Get(healthCheck)
				elapsed := time.Since(start) / time.Millisecond

				lock.Lock()
				defer lock.Unlock()
				if err == nil {
					responses[k] = healthStatus{Response: resp.Status, LatencyMS: elapsed}
					if resp.StatusCode != 200 {
						allOk = false
					}
				} else {
					responses[k] = healthStatus{Response: err.Error(), LatencyMS: elapsed}
					allOk = false
				}
			}()
		}

		wg.Wait()

		res, err := json.MarshalIndent(responses, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !allOk {
			w.WriteHeader(503)
		}
		w.Write(res)
	})
}
