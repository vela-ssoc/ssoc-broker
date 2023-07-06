package mgtapi

import (
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/app/service"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/dynsql"
	"github.com/xgfone/ship/v5"
	"gorm.io/gen/field"
)

func Agent(svc service.AgentService) route.Router {
	const (
		idKey         = "minion.id"
		tagKey        = "minion_tag.tag"
		inetKey       = "minion.inet"
		editionKey    = "minion.edition"
		idcKey        = "minion.idc"
		ibuKey        = "minion.ibu"
		commentKey    = "minion.`comment`"
		statusKey     = "minion.status"
		goosKey       = "minion.goos"
		archKey       = "minion.arch"
		brokerNameKey = "minion.broker_name"
		opDutyKey     = "minion.op_duty"
		createdAtKey  = "minion.created_at"
		uptimeKey     = "minion.uptime"
	)

	idCol := dynsql.IntColumn(idKey, "ID").Build()
	tagCol := dynsql.StringColumn(tagKey, "标签").
		Operators([]dynsql.Operator{dynsql.Eq, dynsql.Like, dynsql.In}).
		Build()
	inetCol := dynsql.StringColumn(inetKey, "终端IP").Build()
	verCol := dynsql.StringColumn(editionKey, "版本").Build()
	idcCol := dynsql.StringColumn(idcKey, "机房").Build()
	ibuCol := dynsql.StringColumn(ibuKey, "部门").Build()
	commentCol := dynsql.StringColumn(commentKey, "描述").Build()
	statusEnums := dynsql.IntEnum().Set(1, "未激活").Set(2, "离线").
		Set(3, "在线").Set(4, "已删除")
	statusCol := dynsql.IntColumn(statusKey, "状态").
		Enums(statusEnums).
		Operators([]dynsql.Operator{dynsql.Eq, dynsql.Ne, dynsql.In, dynsql.NotIn}).
		Build()
	goosEnums := dynsql.StringEnum().Sames([]string{"linux", "windows", "darwin"})
	goosCol := dynsql.StringColumn(goosKey, "操作系统").
		Enums(goosEnums).
		Operators([]dynsql.Operator{dynsql.Eq, dynsql.Ne, dynsql.In, dynsql.NotIn}).
		Build()
	archEnums := dynsql.StringEnum().Sames([]string{"amd64", "386", "arm64", "arm"})
	archCol := dynsql.StringColumn(archKey, "系统架构").
		Enums(archEnums).
		Operators([]dynsql.Operator{dynsql.Eq, dynsql.Ne, dynsql.In, dynsql.NotIn}).
		Build()
	brkCol := dynsql.StringColumn(brokerNameKey, "代理节点").Build()
	dutyCol := dynsql.StringColumn(opDutyKey, "运维负责人").Build()
	catCol := dynsql.TimeColumn(createdAtKey, "创建时间").Build()
	upCol := dynsql.TimeColumn(uptimeKey, "上线时间").Build()

	tbl := dynsql.Builder().
		Filters(tagCol, inetCol, goosCol, archCol, statusCol, verCol, idcCol, ibuCol, commentCol,
			brkCol, dutyCol, catCol, upCol, idCol).
		Build()

	monTbl := query.Minion
	likes := map[string]field.String{
		tagKey:        query.MinionTag.Tag,
		inetKey:       monTbl.Inet,
		editionKey:    monTbl.Edition,
		idcKey:        monTbl.IDC,
		ibuKey:        monTbl.IBu,
		commentKey:    monTbl.Comment,
		goosKey:       monTbl.Goos,
		archKey:       monTbl.Arch,
		brokerNameKey: monTbl.BrokerName,
		opDutyKey:     monTbl.OpDuty,
	}

	return &agentREST{
		svc:   svc,
		tbl:   tbl,
		likes: likes,
	}
}

type agentREST struct {
	svc   service.AgentService
	tbl   dynsql.Table
	likes map[string]field.String
}

func (rest *agentREST) Route(r *ship.RouteGroupBuilder) {
	r.Route(accord.PathUpgrade).Data(route.Named("通知节点二进制升级")).POST(rest.Upgrade)
	r.Route(accord.PathStartup).Data(route.Named("通知节点 startup 更新")).POST(rest.Startup)
	r.Route(accord.PathCommand).Data(route.Named("通知节点执行命令")).POST(rest.Command)
}

func (rest *agentREST) Upgrade(c *ship.Context) error {
	var req accord.Upgrade
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	return rest.svc.UpgradeID(ctx, req.ID, req.Semver)
}

func (rest *agentREST) Startup(c *ship.Context) error {
	var req accord.Startup
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	return rest.svc.Startup(ctx, req.ID)
}

func (rest *agentREST) Command(c *ship.Context) error {
	var req accord.Command
	if err := c.Bind(&req); err != nil {
		return err
	}
	ctx := c.Request().Context()

	cmd := req.Cmd
	if cmd == "offline" {
		return rest.svc.Offline(ctx, req.ID)
	}

	return rest.svc.Command(ctx, req.ID, cmd)
}

func (rest *agentREST) Batch(c *ship.Context) error {
	var req param.MinionBatchRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	return nil
}
