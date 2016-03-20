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

	certFile string
	certKey  string
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func checkFile(name string) {
	if name != "" {
		log.Printf("deprecated option %q provided", name)
		flag.Usage()
		os.Exit(1)
	}
}

func newConfig() *sohop.Config {
	flag.StringVar(&configPath, "config", "config.json", "Config file")
	flag.StringVar(&httpAddr, "httpAddr", ":80", "Address to bind HTTP server")
	flag.StringVar(&httpsAddr, "httpsAddr", ":443", "Address to bind HTTPS server")
	flag.StringVar(&certFile, "certFile", "", "(deprecated) Now set in config file.  Originally, server certificate.  ")
	flag.StringVar(&certKey, "certKey", "", "(deprecated) Now set in config file.  Originally, server certificate private key. ")
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
		HTTPAddr:  httpAddr,
		HTTPSAddr: httpsAddr,
	}.Run()
}
