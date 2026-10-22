package restapi

import (
	"io"
	"os/exec"

	"github.com/UserExistsError/conpty"
	_ "github.com/UserExistsError/conpty"
	"github.com/gorilla/websocket"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-common-mb/pterm"
	"github.com/xgfone/ship/v5"
)

type PTY struct {
	wsu *websocket.Upgrader
}

func NewPTY(wsu *websocket.Upgrader) *PTY {
	return &PTY{wsu: wsu}
}

func (pt *PTY) RegisterHTTP(g shipx.RouteGroup) error {
	g.ManagerV3.Route("/system/pty").GET(pt.open)

	return nil
}

func (pt *PTY) open(c *ship.Context) error {
	w, r := c.Response(), c.Request()
	ws, err := pt.wsu.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	var command string
	cmds := []string{"pwsh.exe", "C:\\Program Files\\PowerShell\\7\\pwsh.exe", "powershell.exe", "cmd.exe"}
	for _, cmd := range cmds {
		if command, _ = exec.LookPath(cmd); command != "" {
			break
		}
	}
	if command == "" {
		command = "C:\\WINDOWS\\system32\\cmd.exe"
	}
	cpty, err := conpty.Start(command)
	if err != nil {
		return err
	}
	defer cpty.Close()

	stdin, stdout := pterm.NewIO(ws)
	go func() {
		_, _ = io.Copy(stdout, cpty)
		_ = ws.Close()
	}()

	for {
		data, err1 := stdin.Read()
		if err1 != nil {
			return err1
		}

		switch data.Type {
		case "ping":
			_ = stdout.WriteType("pong", "")
			continue
		case "exit":
			break
		case "stdin":
			msg, err2 := data.AsString()
			if err2 == nil {
				_, err2 = cpty.Write([]byte(msg))
			}
			if err2 != nil {
				return err2
			}
		case "resize":
			msg, err2 := data.AsResize()
			if err2 == nil {
				err2 = cpty.Resize(msg.Cols, msg.Rows)
			}
			if err2 != nil {
				return err2
			}
		default:
		}
	}
}
