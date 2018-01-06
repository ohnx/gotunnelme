# localtunnel

golang version of http://localtunnel.me client

See the original client [here](https://github.com/localtunnel/localtunnel).

## Why Go?

Because <3 a single binary.

## Installation

On systems with go set up:
```
go get github.com/ohnx/localtunnel
```

If you don't have go installed,  you can download a pre-compiled release from [here](https://github.com/ohnx/localtunnel/releases).

## Usage and arguments

Assuming you have `GOPATH/bin` in your `PATH`, you can run:

```
localtunnel
```

For help, see the output of `localtunnel -h`:

```
Usage of localtunnel:
  -local-host string
        Proxy to a hostname (default is localhost) (default "localhost")
  -port int
        Port to connect tunnel to (default is 80) (default 80)
  -remote string
        Remote localtunnel server (default is localtunnel.me) (default "https://localtunnel.me/")
  -subdomain string
        Request named subdomain (default is random characters)
```

## API

In progress.
