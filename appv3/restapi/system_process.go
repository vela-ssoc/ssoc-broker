package restapi

import (
	"net/http"

	psprocess "github.com/shirou/gopsutil/v4/process"
	"github.com/vela-ssoc/ssoc-broker/appv3/request"
	"github.com/vela-ssoc/ssoc-broker/appv3/response"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-common-mb/psutil"
	"github.com/xgfone/ship/v5"
)

type SystemProcess struct {
}

func NewSystemProcess() *SystemProcess {
	return &SystemProcess{}
}

func (sp *SystemProcess) RegisterHTTP(g shipx.RouteGroup) error {
	g.ManagerV3.Group.Route("/system/processes").GET(sp.list)                  // 进程列表
	g.ManagerV3.Group.Route("/system/process").GET(sp.info)                    // 进程信息
	g.ManagerV3.Group.Route("/system/process/connections").GET(sp.connections) // 进程打开的连接数
	g.ManagerV3.Group.Route("/system/process/openfiles").GET(sp.openfiles)     // 进程关联的文件描述符

	return nil
}

func (sp *SystemProcess) list(c *ship.Context) error {
	ctx := c.Request().Context()
	infos, err := psutil.ProcessesWithContext(ctx)
	if err != nil {
		return err
	}
	ret := response.NewRecords(infos)

	return c.JSON(http.StatusOK, ret)
}

func (sp *SystemProcess) info(c *ship.Context) error {
	var param request.SystemPID
	if err := c.BindQuery(&param); err != nil {
		return err
	}

	ctx := c.Request().Context()
	infos, err := psutil.ProcessesWithContext(ctx)
	if err != nil {
		return err
	}
	ret := response.NewRecords(infos)

	return c.JSON(http.StatusOK, ret)
}

func (sp *SystemProcess) connections(c *ship.Context) error {
	var param request.SystemPID
	if err := c.BindQuery(&param); err != nil {
		return err
	}

	ctx := c.Request().Context()
	proc, err := psprocess.NewProcessWithContext(ctx, param.PID)
	if err != nil {
		return err
	}
	stats, err := proc.ConnectionsWithContext(ctx)
	if err != nil {
		return err
	}
	ret := response.NewRecords(stats)

	return c.JSON(http.StatusOK, ret)
}

func (sp *SystemProcess) openfiles(c *ship.Context) error {
	var param request.SystemPID
	if err := c.BindQuery(&param); err != nil {
		return err
	}

	ctx := c.Request().Context()
	proc, err := psprocess.NewProcessWithContext(ctx, param.PID)
	if err != nil {
		return err
	}
	stats, err := proc.OpenFilesWithContext(ctx)
	if err != nil {
		return err
	}
	ret := response.NewRecords(stats)

	return c.JSON(http.StatusOK, ret)
}
