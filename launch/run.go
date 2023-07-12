package launch

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-broker/app/agtapi"
	"github.com/vela-ssoc/vela-broker/app/mgtapi"
	"github.com/vela-ssoc/vela-broker/app/middle"
	"github.com/vela-ssoc/vela-broker/app/service"
	"github.com/vela-ssoc/vela-broker/app/temporary"
	"github.com/vela-ssoc/vela-broker/app/temporary/linkhub"
	"github.com/vela-ssoc/vela-broker/bridge/gateway"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-broker/bridge/telecom"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/vela-ssoc/vela-common-mb/audit"
	"github.com/vela-ssoc/vela-common-mb/dal/gridfs"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/dbms"
	"github.com/vela-ssoc/vela-common-mb/gopool"
	"github.com/vela-ssoc/vela-common-mb/integration/alarm"
	"github.com/vela-ssoc/vela-common-mb/integration/cmdb"
	"github.com/vela-ssoc/vela-common-mb/integration/devops"
	"github.com/vela-ssoc/vela-common-mb/integration/dong"
	"github.com/vela-ssoc/vela-common-mb/integration/elastic"
	"github.com/vela-ssoc/vela-common-mb/integration/ntfmatch"
	"github.com/vela-ssoc/vela-common-mb/integration/sonatype"
	"github.com/vela-ssoc/vela-common-mb/integration/vulnsync"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mb/problem"
	"github.com/vela-ssoc/vela-common-mb/storage"
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
	pool := gopool.New(4096, 2048, 10*time.Minute)
	match := ntfmatch.NewMatch()
	store := storage.NewStore()

	dongCfg := dong.NewConfigure()
	dongCli := dong.NewClient(dongCfg, cli, slog)
	devopsCfg := devops.NewConfig(store)
	devCli := devops.NewClient(devopsCfg, cli)

	alert := alarm.UnifyAlerter(store, pool, match, slog, dongCli, devCli)

	// manager callback
	name := link.Name()
	pbh := problem.NewHandle(name)

	valid := validate.New()
	agt := ship.Default()
	mgt := ship.Default()
	mgt.Logger = slog
	mgt.NotFound = pbh.NotFound
	mgt.HandleError = pbh.HandleError
	mgt.Validator = valid
	agt.Logger = slog
	agt.NotFound = pbh.NotFound
	agt.HandleError = pbh.HandleError
	agt.Validator = valid

	mv1 := mgt.Group(accord.PathPrefix).Use(middle.Oplog)
	av1 := agt.Group(accord.PathPrefix).Use(middle.Oplog)

	thirdService := service.Third(gfs)
	thirdREST := agtapi.Third(thirdService)
	thirdREST.Route(av1)
	bid := link.Ident().ID
	agtapi.Upgrade(bid, gfs).Route(av1)
	agtapi.Task().Route(av1)

	esCfg := elastic.NewConfigure(name)
	esc := elastic.NewSearch(esCfg, cli)

	agtapi.Stream(name, esc).Route(av1)
	agtapi.Elastic(esc).Route(av1)
	agtapi.Heart().Route(av1)
	agtapi.Collect().Route(av1)
	agtapi.Security().Route(av1)

	auditor := audit.NewAuditor(slog)
	agtapi.Audit(auditor).Route(av1)
	agtapi.BPF().Route(av1)

	ntfMatch := ntfmatch.NewMatch()
	cmdbCfg := cmdb.NewConfigure(store)
	cmdbCli := cmdb.NewClient(cmdbCfg, cli, slog)
	nodeEventService := service.NodeEvent(cmdbCli, pool, alert, slog)

	linkpool := gopool.New(1024, 1024, 10*time.Minute)
	hub := mlink.LinkHub(link, agt, nodeEventService, linkpool)
	_ = hub.ResetDB()

	taskService := service.Task(hub, pool, slog)
	taskREST := mgtapi.Task(taskService)
	taskREST.Route(mv1)

	operateService := service.Operate(hub, pool, slog)
	agtapi.Operate(operateService).Route(av1)
	mgtapi.Reset(store, esCfg, ntfMatch).Route(mv1)
	mgtapi.Third(hub, pool).Route(mv1)

	sonaCfg := sonatype.HardConfig()
	sonaCli := sonatype.NewClient(sonaCfg, cli)
	vsync := vulnsync.New(db, sonaCli)
	_ = vsync

	intoService := service.Into(hub)
	intoREST := mgtapi.Into(intoService)
	intoREST.Route(mv1)
	agentService := service.Agent(hub, pool, slog)
	agentREST := mgtapi.Agent(agentService)
	agentREST.Route(mv1)
	mgtapi.Pprof(link).Route(mv1)

	oldHandler := linkhub.New(db, link, slog, gfs)
	temp := temporary.REST(oldHandler, valid, slog)
	gw := gateway.New(hub)

	mux := ship.Default()
	api := mux.Group("/")
	api.Route("/api/v1/minion").CONNECT(func(c *ship.Context) error {
		gw.ServeHTTP(c.ResponseWriter(), c.Request())
		return nil
	})
	api.Route("/v1/minion/endpoint").GET(temp.Endpoint)
	api.Route("/v1/edition/upgrade").GET(oldHandler.Upgrade)

	// mux := http.NewServeMux()
	// mux.Handle("/api/v1/minion", gw)
	// mux.Handle("/api/minion/endpoint", nil)
	// mux.Handle("/api/edition/upgrade", nil)

	errCh := make(chan error, 1)

	// 监听本地端口用于 minion 节点连接
	ds := &daemonServer{listen: issue.Listen, hide: hide, handler: mux, errCh: errCh}
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
