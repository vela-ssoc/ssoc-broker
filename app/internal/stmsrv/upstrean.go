package stmsrv

import (
	"bufio"
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vela-ssoc/ssoc-common-mba/netutil"
)

type Upstreamer interface {
	Proto() []string
	Dial(dest *url.URL, skip bool) (upstream net.Conn, err error)
	Serve(dest *url.URL, upstream net.Conn, ws *websocket.Conn) error
}

func Net() Upstreamer {
	return &netUpstream{proto: []string{"tcp", "udp", "tls"}}
}

func HTTP() Upstreamer {
	return &httpUpstream{}
}

type netUpstream struct {
	proto []string
}

func (up *netUpstream) Proto() []string {
	return up.proto
}

func (up *netUpstream) Dial(dest *url.URL, skip bool) (net.Conn, error) {
	scheme := dest.Scheme
	switch scheme {
	case "tls":
		dial := &tls.Dialer{Config: &tls.Config{InsecureSkipVerify: skip}}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return dial.DialContext(ctx, "tcp", dest.Host)
	default:
		return net.DialTimeout(scheme, dest.Host, 5*time.Second)
	}
}

func (up *netUpstream) Serve(_ *url.URL, upstream net.Conn, ws *websocket.Conn) error {
	defer func() {
		_ = upstream.Close()
		_ = ws.Close()
	}()
	_ = netutil.Pipe(upstream, ws)
	return nil
}

type httpUpstream struct{}

func (up *httpUpstream) Proto() []string {
	return []string{"http", "https"}
}

func (up *httpUpstream) Dial(dest *url.URL, skip bool) (upstream net.Conn, err error) {
	scheme := dest.Scheme
	host, _, _ := net.SplitHostPort(dest.Host)
	if scheme == "https" {
		addr := dest.Host
		if host == "" {
			addr += ":443"
		}
		dial := &tls.Dialer{Config: &tls.Config{InsecureSkipVerify: skip}}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return dial.DialContext(ctx, "tcp", addr)
	}
	addr := dest.Host
	if host == "" {
		addr += ":80"
	}
	return net.DialTimeout("tcp", addr, 5*time.Second)
}

func (up *httpUpstream) Serve(dest *url.URL, upstream net.Conn, ws *websocket.Conn) error {
	defer func() {
		_ = upstream.Close()
		_ = ws.Close()
	}()

	host, _, _ := net.SplitHostPort(dest.Host)
	if host == "" {
		host = dest.Host
	}

	bio := bufio.NewReader(websocket.JoinMessages(ws, ""))
	for {
		req, err := http.ReadRequest(bio)
		if err != nil {
			return err
		}
		req.URL.Scheme = dest.Scheme
		req.Host = host
		log.Printf("-------------------- %s", host)
		if err = req.Write(upstream); err != nil {
			return err
		}
		resp, err := http.ReadResponse(bufio.NewReader(upstream), req)
		if err != nil {
			return err
		}
		writer, err := ws.NextWriter(websocket.BinaryMessage)
		if err != nil {
			return err
		}
		err = resp.Write(writer)
		_ = writer.Close()
		if err != nil {
			return err
		}
	}
}
