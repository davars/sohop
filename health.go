package sohop

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

var healthClient = createHealthClient()

const certWarning = 72 * time.Hour

func createHealthClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   5 * time.Second,
	}
	return client
}

type healthStatus struct {
	Response  string        `json:"response"`
	LatencyMS time.Duration `json:"latency_ms"`
}

func (s Server) HealthHandler() http.Handler {
	data, err := ioutil.ReadFile(s.Config.TLS.CertFile)
	check(err)
	notBefore, notAfter, err := CertValidity(data)
	check(err)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allOk := true
		responses := make(map[string]healthStatus)

		var lock sync.Mutex // responses
		var wg sync.WaitGroup

		for _k, _v := range s.Config.Upstreams {
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

		certResponse := map[string]interface{}{
			"expires_at": notAfter,
		}
		now := time.Now()
		if !notBefore.Before(now) {
			certResponse["error"] = "not yet valid"
			certResponse["valid_at"] = notBefore
			certResponse["ok"] = false
		} else if !notAfter.After(now) {
			certResponse["error"] = "expired"
			certResponse["ok"] = false
		} else if !notAfter.Add(-1 * certWarning).After(now) {
			certResponse["expires_in"] = notAfter.Sub(now).String()
			certResponse["error"] = "expires soon"
			certResponse["ok"] = false
		} else {
			certResponse["ok"] = true
		}

		allOk = allOk && certResponse["ok"].(bool)

		res, err := json.MarshalIndent(struct {
			Upstreams map[string]healthStatus `json:"upstreams"`
			Cert      map[string]interface{}  `json:"cert"`
		}{
			Upstreams: responses,
			Cert:      certResponse,
		}, "", "  ")
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
