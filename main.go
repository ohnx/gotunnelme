package main

import (
	// standard library
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	// localtunnel-specific
	"github.com/ohnx/localtunnel/gotunnelme"
)

func main() {
	var port int
	var subdomain string
	var localHost string
	var tunnelServer string

	// Set up flags
	flag.IntVar(&port, "port", 80, "Port to connect tunnel to (default is 80)")
	flag.StringVar(&subdomain, "subdomain", "", "Request named subdomain (default is random characters)")
	flag.StringVar(&localHost, "local-host", "localhost", "Proxy to a hostname (default is localhost)")
	flag.StringVar(&tunnelServer, "remote", "https://localtunnel.me/", "Remote localtunnel server (default is localtunnel.me)")

	// Parse flags
	flag.Parse()

	// Get tunnel handle
	t := gotunnelme.NewTunnel(tunnelServer)

	// custom host
	url, err := t.GetUrl(subdomain)

	// failed to get custom host
	if err != nil {
		log.Fatal(err)
	}

	// info message
	log.Printf("Tunneling %s -> %s:%d", url, localHost, port)

	// close handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(){
    	for sig := range c {
    		fmt.Println()
    		log.Printf("Caught signal %s; closing tunnel...", sig)
        	t.StopTunnel()
    	}
	}()

	// create tunnel
	err = t.CreateTunnel(localHost, port)
	if err != nil {
		log.Fatal(err)
	}

	// Start the tunnel
	err = t.StartTunnel()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Goodbye...")
}

