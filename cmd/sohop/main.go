// Package main implements the CLI for sohop.
package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

	"github.com/davars/sohop"
)

var (
	configPath string
	httpAddr   string
	httpsAddr  string
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func newConfig() *sohop.Config {
	flag.StringVar(&configPath, "config", "config.json", "Config file")
	flag.StringVar(&httpAddr, "httpAddr", ":80", "Address to bind HTTP server")
	flag.StringVar(&httpsAddr, "httpsAddr", ":443", "Address to bind HTTPS server")
	flag.Parse()

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
