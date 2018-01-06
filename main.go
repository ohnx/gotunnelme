package main

import (
	"fmt"
	"github.com/ohnx/localtunnel/gotunnelme"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "gotunnelme <local port>")
		os.Exit(1)
	}
	i, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	t := gotunnelme.NewTunnel()
	url, err := t.GetUrl("")
	if err != nil {
		panic(err)
	}
	print("Tunnel now available at: ")
	println(url)

	err = t.CreateTunnel(i)
	if err != nil {
		panic(err)
	}
	t.StopTunnel()
}

