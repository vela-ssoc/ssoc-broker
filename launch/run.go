package launch

import (
	"context"
	"net/http"

	"github.com/vela-ssoc/vela-broker/app/agtapi"
	"github.com/vela-ssoc/vela-broker/app/mgtapi"
	"github.com/vela-ssoc/vela-broker/app/service"
	"github.com/vela-ssoc/vela-broker/app/subtask"
	"github.com/vela-ssoc/vela-broker/bridge/gateway"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-broker/bridge/telecom"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/vela-ssoc/vela-common-mb/audit"
	"github.com/vela-ssoc/vela-common-mb/dal/gridfs"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/dbms"
	"github.com/vela-ssoc/vela-common-mb/integration/alarm"
	"github.com/vela-ssoc/vela-common-mb/integration/cmdb"
	"github.com/vela-ssoc/vela-common-mb/integration/devops"
	"github.com/vela-ssoc/vela-common-mb/integration/dong"
	"github.com/vela-ssoc/vela-common-mb/integration/elastic"
	"github.com/vela-ssoc/vela-common-mb/integration/formwork"
	"github.com/vela-ssoc/vela-common-mb/integration/ntfmatch"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mb/problem"
	"github.com/vela-ssoc/vela-common-mb/taskpool"
	"github.com/vela-ssoc/vela-common-mb/validate"
	"github.com/vela-ssoc/vela-common-mba/netutil"
	"github.com/xgfone/ship/v5"
)

// Run 运行服务
func Run(parent context.Context, hide telecom.Hide, slog logback.Logger) error {
	link, err := telecom.Dial(parent, hide, slog) // 与中心端建立连接
	if err != nil {
		return err
	}

	ident := link.Ident()
	issue := link.Issue()
	slog.Infof("broker 接入认证成功，上报认证信息如下：\n%s\n下发的配置如下：\n%s", ident, issue)

	logCfg := issue.Logger
	zlg := logCfg.Zap() // 根据配置文件初始化日志
	slog.Replace(zlg)   // 替换日志输出内核
	gormLog := logback.Gorm(zlg, logCfg.Level)

	dbCfg := issue.Database
	db, sdb, err := dbms.Open(dbCfg, gormLog)
	if err != nil {
		return err
	}
	query.SetDefault(db)
	gfs := gridfs.NewCDN(sdb, "", 60*1024)

	// minionHandler := handler.Minion()
	cli := netutil.NewClient()
	pool := taskpool.NewPool(256, 1024)
	rend := formwork.NewRend(slog)
	match := ntfmatch.NewMatch()

	dongCfg := dong.NewConfigure()
	dongCli := dong.NewClient(dongCfg, cli, slog)
	devCli := devops.NewClient(cli)

	alert := alarm.UnifyAlerter(rend, pool, match, slog, dongCli, devCli)

	// manager callback
	name := link.Name()
	pbh := problem.NewHandle(name)

	agt := ship.Default()
	mgt := ship.Default()
	mgt.Logger = slog
	mgt.NotFound = pbh.NotFound
	mgt.HandleError = pbh.HandleError
	mgt.Validator = validate.New()
	agt.Logger = slog
	agt.NotFound = pbh.NotFound
	agt.HandleError = pbh.HandleError
	agt.Validator = validate.New()

	mv1 := mgt.Group(accord.PathPrefix)
	av1 := agt.Group(accord.PathPrefix)

	thirdService := service.Third(gfs)
	thirdREST := agtapi.Third(thirdService)
	thirdREST.Route(av1)

	esCfg := elastic.NewSearchConfigure()
	esc := elastic.NewSearch(esCfg, cli)

	agtapi.Stream(name, esc).Route(av1)
	agtapi.Forward(esc).Route(av1)
	agtapi.Heart().Route(av1)
	agtapi.Operate().Route(av1)
	agtapi.Collect().Route(av1)
	agtapi.Security().Route(av1)

	auditor := audit.NewAuditor(slog)
	agtapi.Audit(auditor).Route(av1)
	agtapi.BPF().Route(av1)

	cmdbCfg := cmdb.NewConfigure(slog)
	cmdbCli := cmdb.NewClient(cmdbCfg, cli, slog)
	compare := subtask.Compare()
	nodeEventService := service.NodeEvent(compare, cmdbCli, pool, alert, slog)
	hub := mlink.LinkHub(link, agt, nodeEventService, pool)
	_ = hub.ResetDB()

	taskService := service.Task(hub, compare, pool, slog)
	taskREST := mgtapi.Task(taskService)
	taskREST.Route(mv1)

	mgtapi.Reset(esCfg, cmdbCfg).Route(mv1)
	mgtapi.Third(hub, pool).Route(mv1)

	intoService := service.Into(hub)
	intoREST := mgtapi.Into(intoService)
	intoREST.Route(mv1)

	gw := gateway.New(hub)
	mux := http.NewServeMux()
	mux.Handle("/api/v1/minion", gw)

	errCh := make(chan error, 1)

	// 监听本地端口用于 minion 节点连接
	ds := &daemonServer{listen: issue.Listen, handler: gw, errCh: errCh}
	go ds.Run()

	// 连接 manager 的客户端，保持在线与接受指令
	dc := &daemonClient{link: link, handler: mgt, errCh: errCh, slog: slog, parent: parent}
	go dc.Run()

	select {
	case err = <-errCh:
	case <-parent.Done():
	}

	_ = ds.Close()
	_ = dc.Close()
	_ = hub.ResetDB()
	_ = zlg.Sync()

	return err
}
