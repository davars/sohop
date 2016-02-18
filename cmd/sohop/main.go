package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
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
	sohop.Server{
		Config:    newConfig(),
		CertFile:  certFile,
		CertKey:   certKey,
		HTTPAddr:  httpAddr,
		HTTPSAddr: httpsAddr,
	}.Run()
}
