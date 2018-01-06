package gotunnelme

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var Debug = false

type TunnelConn struct {
	remoteHost  	string
	remotePort  	int
	proxyHost		string
	proxyPort   	int
	remoteConn  	net.Conn
	localConn   	net.Conn
	errorChannel	chan error
}

func NewTunnelConn(remoteHost, proxyHost string, remotePort, proxyPort int) *TunnelConn {
	tunnelConn := &TunnelConn{}
	tunnelConn.remoteHost = remoteHost
	tunnelConn.remotePort = remotePort
	tunnelConn.proxyHost = proxyHost
	tunnelConn.proxyPort = proxyPort
	return tunnelConn
}

func (self *TunnelConn) Tunnel(replyCh chan<- int) error {
	self.errorChannel = make(chan error, 1) // clear previous channel's message
	remoteConn, remoteErr := self.connectRemote()
	if remoteErr != nil {
		if Debug {
			fmt.Printf("Remote[%s:%d]: Connect error: %s!\n", self.remoteHost, self.remotePort, remoteErr.Error())
		}
		replyCh <- -1
		return remoteErr
	}

	if Debug {
		fmt.Printf("Remote[%s:%d]: Connect successful!\n", self.remoteHost, self.remotePort)
	}

	localConn, localErr := self.connectLocal()
	if localErr != nil {
		if Debug {
			fmt.Printf("Local[%s:%d]: Connect error: %s!\n", self.proxyHost, self.proxyPort, localErr.Error())
		}
		replyCh <- -1
		return localErr
	}
	if Debug {
		fmt.Printf("Local[%s:%d]: Connect successful!\n", self.proxyHost, self.proxyPort)
	}

	self.remoteConn = remoteConn
	self.localConn = localConn
	go func() {
		var err error
		for {
			_, err = io.Copy(remoteConn, localConn)
			if err != nil {
				if Debug {
					fmt.Printf("Stop copy from local to remote! error=[%v]\n", err)
				}
				break
			}
		}
		self.errorChannel <- err
	}()
	go func() {
		var err error
		for {
			_, err = io.Copy(localConn, remoteConn)
			if err != nil {
				if Debug {
					fmt.Printf("Stop copy from remote to local! error=[%v]\n", err)
				}
				break
			}

		}
		self.errorChannel <- err
	}()
	err := <-self.errorChannel
	replyCh <- 1
	return err
}

func (self *TunnelConn) StopTunnel() error {
	if self.remoteConn != nil {
		self.remoteConn.Close()
	}
	if self.localConn != nil {
		self.localConn.Close()
	}
	return nil
}

func (self *TunnelConn) connectRemote() (net.Conn, error) {
	remoteAddr := fmt.Sprintf("%s:%d", self.remoteHost, self.remotePort)
	addr := remoteAddr
	proxy := os.Getenv("HTTP_PROXY")
	if proxy == "" {
		proxy = os.Getenv("http_proxy")
	}
	if len(proxy) > 0 {
		url, err := url.Parse(proxy)
		if err == nil {
			addr = url.Host
		}
	}
	remoteConn, remoteErr := net.Dial("tcp", addr)
	if remoteErr != nil {
		return nil, remoteErr
	}

	if len(proxy) > 0 {
		fmt.Fprintf(remoteConn, "CONNECT %s HTTP/1.1\r\n", remoteAddr)
		fmt.Fprintf(remoteConn, "Host: %s\r\n", remoteAddr)
		fmt.Fprintf(remoteConn, "\r\n")
		br := bufio.NewReader(remoteConn)
		req, _ := http.NewRequest("CONNECT", remoteAddr, nil)
		resp, readRespErr := http.ReadResponse(br, req)
		if readRespErr != nil {
			return nil, readRespErr
		}
		if resp.StatusCode != 200 {
			f := strings.SplitN(resp.Status, " ", 2)
			return nil, errors.New(f[1])
		}

		if Debug {
			fmt.Printf("Connect %s by proxy[%s].\n", remoteAddr, proxy)
		}
	}
	return remoteConn, nil
}

func (self *TunnelConn) connectLocal() (net.Conn, error) {
	localAddr := fmt.Sprintf("%s:%d", self.proxyHost, self.proxyPort)
	return net.Dial("tcp", localAddr)
}

type TunnelCommand int

const (
	stopTunnelCmd TunnelCommand = 1
)

type Tunnel struct {
	assignedUrlInfo *AssignedUrlInfo
	proxyPort       int
	proxyHost		string
	tunnelServer	string
	tunnelConns     []*TunnelConn
	cmdChan         chan TunnelCommand
}

func NewTunnel(tunnelServer string) *Tunnel {
	tunnel := &Tunnel{}
	tunnel.cmdChan = make(chan TunnelCommand, 1)
	tunnel.tunnelServer = tunnelServer
	return tunnel
}

func (self *Tunnel) StartTunnel() error {
	if err := self.checkArgs(); err != nil {
		return err
	}
	url, parseErr := url.Parse(self.tunnelServer)
	if parseErr != nil {
		return parseErr
	}
	replyCh := make(chan int, self.assignedUrlInfo.MaxConnCount)
	remoteHost := url.Host
	for i := 0; i < self.assignedUrlInfo.MaxConnCount; i++ {
		tunnelConn := NewTunnelConn(remoteHost, self.proxyHost, self.assignedUrlInfo.Port, self.proxyPort)
		self.tunnelConns[i] = tunnelConn
		go tunnelConn.Tunnel(replyCh)
	}
L:
	for i := 0; i < self.assignedUrlInfo.MaxConnCount; i++ {
		select {
		case <-replyCh:
		case cmd := <-self.cmdChan:
			switch cmd {
			case stopTunnelCmd:
				break L
			}
		}
	}

	return nil
}

func (self *Tunnel) checkArgs() error {
	localAddr := fmt.Sprintf("%s:%d", self.proxyHost, self.proxyPort)
	c, err := net.Dial("tcp", localAddr)
	if err != nil {
		return errors.New("Can't connect to port!")
	}
	c.Close()
	return nil
}

func (self *Tunnel) StopTunnel() {
	if Debug {
		fmt.Printf("Stop tunnel for [%s:%d]!\n", self.proxyHost, self.proxyPort)
	}
	self.cmdChan <- stopTunnelCmd
	for _, tunnelCon := range self.tunnelConns {
		tunnelCon.StopTunnel()
	}
}

func (self *Tunnel) GetUrl(assignedDomain string) (string, error) {
	if len(assignedDomain) == 0 {
		assignedDomain = "?new"
	}
	assignedUrlInfo, err := GetAssignedUrl(self.tunnelServer, assignedDomain)
	if err != nil {
		return "", err
	}
	self.assignedUrlInfo = assignedUrlInfo
	self.tunnelConns = make([]*TunnelConn, assignedUrlInfo.MaxConnCount)
	return assignedUrlInfo.Url, nil
}

func (self *Tunnel) CreateTunnel(proxyHost string, proxyPort int) error {
	self.proxyHost = proxyHost
	self.proxyPort = proxyPort
	return self.checkArgs()
}
