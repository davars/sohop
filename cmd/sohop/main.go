package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"bitbucket.org/davars/sohop"
)

var (
	configPath string
	httpAddr   string
	httpsAddr  string
	certFile   string
	certKey    string
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func checkFile(name string) {
	if _, err := os.Stat(name); err != nil {
		log.Print(err)
		flag.Usage()
		os.Exit(1)
	}
}

func newConfig() *sohop.Config {
	flag.StringVar(&configPath, "config", "config.json", "Config file")
	flag.StringVar(&httpAddr, "httpAddr", ":80", "Address to bind HTTP server")
	flag.StringVar(&httpsAddr, "httpsAddr", ":443", "Address to bind HTTPS server")
	flag.StringVar(&certFile, "certFile", "cert.pem", "Server certificate")
	flag.StringVar(&certKey, "certKey", "key.pem", "Server certificate private key")
	flag.Parse()

	checkFile(certFile)
	checkFile(certKey)

	configData, err := ioutil.ReadFile(configPath)
	check(err)

	c := &sohop.Config{}
	err = json.Unmarshal(configData, c)
	check(err)

	return c
}

func main() {
	conf := newConfig()
	handler, err := sohop.Handler(conf)
	check(err)
	http.Handle("/", ensuringHTTPS(handler))
	go func() {
		err := http.ListenAndServeTLS(httpsAddr, certFile, certKey, nil)
		check(err)
	}()
	go func() {
		err := http.ListenAndServe(httpAddr, nil)
		check(err)
	}()
	select {}
}

func ensuringHTTPS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			r.URL.Scheme = "https"
			r.URL.Host = r.Host + httpsAddr
			http.Redirect(w, r, r.URL.String(), http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}
