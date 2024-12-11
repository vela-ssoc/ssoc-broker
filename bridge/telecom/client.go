package telecom

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"time"

	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mb/problem"
	"github.com/vela-ssoc/vela-common-mba/netutil"
	"github.com/vela-ssoc/vela-common-mba/smux"
)

type brokerClient struct {
	hide   Hide
	ident  Ident
	issue  Issue
	slog   logback.Logger
	client netutil.HTTPClient
	dialer *iterDial
	mux    *smux.Session
	joinAt time.Time
	parent context.Context
	ctx    context.Context
	cancel context.CancelFunc
}

func (bc *brokerClient) Client() netutil.HTTPClient { return bc.client }
func (bc *brokerClient) Hide() Hide                 { return bc.hide }
func (bc *brokerClient) Ident() Ident               { return bc.ident }
func (bc *brokerClient) Issue() Issue               { return bc.issue }
func (bc *brokerClient) Listen() net.Listener       { return bc.mux }

func (bc *brokerClient) JoinAt() time.Time {
	return bc.joinAt
}

func (bc *brokerClient) Name() string {
	return fmt.Sprintf("broker-%s-%d", bc.ident.Inet, bc.ident.ID)
}

func (bc *brokerClient) Reconnect(parent context.Context) error {
	_ = bc.close()
	return bc.dial(parent)
}

func (bc *brokerClient) close() error {
	bc.cancel()
	return bc.mux.Close()
}

func (bc *brokerClient) dial(parent context.Context) error {
	bc.parent = parent
	bc.ctx, bc.cancel = context.WithCancel(parent)
	start := time.Now()

	for {
		conn, addr, err := bc.dialer.iterDial(bc.ctx, 5*time.Second)
		if err != nil {
			if ce := bc.ctx.Err(); ce != nil {
				return ce
			}
			bc.slog.Warnf("dial %s 失败：%s", addr, err)
			bc.dialSleep(bc.ctx, start)
			continue
		}

		bc.slog.Infof("dial %s 成功，准备握手协商。", addr)
		ident, issue, err := bc.consult(bc.ctx, conn, addr)
		if err == nil {
			cfg := smux.DefaultConfig()
			cfg.KeepAliveDisabled = true
			mux := smux.Client(conn, cfg)
			// mux := spdy.Client(conn, spdy.WithEncrypt(issue.Passwd))
			bc.ident, bc.issue, bc.mux, bc.joinAt = ident, issue, mux, time.Now()
			return nil
		}

		_ = conn.Close()
		if pe := parent.Err(); pe != nil {
			return pe
		}

		if he, ok := err.(*netutil.HTTPError); ok && he.NotAcceptable() {
			return he
		} else if pde, ok := err.(problem.Detail); ok && pde.Status == http.StatusNotAcceptable {
			return pde
		} else if pe, ok := err.(*problem.Detail); ok && pe.Status == http.StatusNotAcceptable {
			return pe
		}

		bc.slog.Warnf("与 %s 协商失败：%s", addr, err)
		bc.dialSleep(parent, start)
	}
}

// consult 当建立好 TCP 连接后进行应用层协商
func (bc *brokerClient) consult(parent context.Context, conn net.Conn, addr *netutil.Address) (Ident, Issue, error) {
	ip := conn.LocalAddr().(*net.TCPAddr).IP
	mac := bc.dialer.lookupMAC(ip)

	ident := Ident{
		ID:     bc.hide.ID,
		Secret: bc.hide.Secret,
		Semver: bc.hide.Semver,
		Inet:   ip,
		MAC:    mac.String(),
		Goos:   runtime.GOOS,
		Arch:   runtime.GOARCH,
		TimeAt: time.Now(),
	}
	ident.Hostname, _ = os.Hostname()
	ident.PID = os.Getpid()
	ident.Workdir, _ = os.Getwd()
	executable, _ := os.Executable()
	ident.Executable = executable
	ident.CPU = runtime.NumCPU()
	if cu, _ := user.Current(); cu != nil {
		ident.Username = cu.Username
	}

	var issue Issue
	enc, err := ident.encrypt()
	if err != nil {
		return ident, issue, err
	}
	buf := bytes.NewReader(enc)

	const endpoint = "http://vtun/api/v1/broker"
	req, err := bc.client.NewRequest(parent, http.MethodConnect, endpoint, buf, nil)
	if err != nil {
		return ident, issue, err
	}

	host := addr.Name
	req.Host = host
	req.URL.Host = host
	if err = req.Write(conn); err != nil {
		return ident, issue, err
	}

	res, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return ident, issue, err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer res.Body.Close()

	code := res.StatusCode
	if code != http.StatusAccepted {
		resp := make([]byte, 10*1024)
		n, _ := io.ReadFull(res.Body, resp)
		pd := new(problem.Detail)
		if err = json.Unmarshal(resp[:n], pd); err == nil {
			return ident, issue, pd
		}
		exr := &netutil.HTTPError{Code: code, Body: resp[:n]}
		return ident, issue, exr
	}

	resp := make([]byte, 100*1024) // 100KiB 缓冲区
	n, err := io.ReadFull(res.Body, resp)
	if err == nil || err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
		err = issue.decrypt(resp[:n])
	}

	return ident, issue, err
}

func (bc *brokerClient) dialSleep(ctx context.Context, start time.Time) {
	since := time.Since(start)
	du := time.Second

	switch {
	case since > 12*time.Hour:
		du = 10 * time.Minute
	case since > time.Hour:
		du = time.Minute
	case since > 30*time.Minute:
		du = 30 * time.Second
	case since > 10*time.Minute:
		du = 10 * time.Second
	case since > 3*time.Minute:
		du = 3 * time.Second
	}

	log.Printf("%s 后进行重试", du)
	// 非阻塞休眠
	select {
	case <-ctx.Done():
	case <-time.After(du):
	}
}

func (bc *brokerClient) dialContext(_ context.Context, _, _ string) (net.Conn, error) {
	mux := bc.mux
	if mux == nil {
		return nil, io.ErrNoProgress
	}

	if stream, err := mux.OpenStream(); err != nil {
		return nil, err
	} else {
		return stream, nil
	}
}

func (bc *brokerClient) heartbeat(du time.Duration) {
	ticker := time.NewTicker(du)
	defer ticker.Stop()

	var over bool
	for !over {
		select {
		case <-ticker.C:
		case <-bc.parent.Done():
			over = true
		}
	}
}
