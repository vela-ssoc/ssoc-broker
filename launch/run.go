package launch

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/vela-ssoc/ssoc-broker/app/agtapi"
	"github.com/vela-ssoc/ssoc-broker/app/agtsvc"
	"github.com/vela-ssoc/ssoc-broker/app/mgtapi"
	"github.com/vela-ssoc/ssoc-broker/app/mgtsvc"
	"github.com/vela-ssoc/ssoc-broker/app/middle"
	"github.com/vela-ssoc/ssoc-broker/app/temporary"
	"github.com/vela-ssoc/ssoc-broker/app/temporary/linkhub"
	"github.com/vela-ssoc/ssoc-broker/appv2/manager/mrestapi"
	"github.com/vela-ssoc/ssoc-broker/appv2/manager/mservice"
	"github.com/vela-ssoc/ssoc-broker/bridge/gateway"
	"github.com/vela-ssoc/ssoc-broker/bridge/mlink"
	"github.com/vela-ssoc/ssoc-broker/bridge/telecom"
	"github.com/vela-ssoc/ssoc-broker/foreign/bytedance"
	"github.com/vela-ssoc/ssoc-broker/library/pipelog"
	"github.com/vela-ssoc/ssoc-common-mb/accord"
	"github.com/vela-ssoc/ssoc-common-mb/dal/gridfs"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"github.com/vela-ssoc/ssoc-common-mb/integration/alarm"
	"github.com/vela-ssoc/ssoc-common-mb/integration/cmdb"
	"github.com/vela-ssoc/ssoc-common-mb/integration/devops"
	"github.com/vela-ssoc/ssoc-common-mb/integration/dong/v2"
	"github.com/vela-ssoc/ssoc-common-mb/integration/elastic"
	"github.com/vela-ssoc/ssoc-common-mb/integration/ntfmatch"
	"github.com/vela-ssoc/ssoc-common-mb/integration/sonatype"
	"github.com/vela-ssoc/ssoc-common-mb/integration/vulnsync"
	"github.com/vela-ssoc/ssoc-common-mb/param/negotiate"
	"github.com/vela-ssoc/ssoc-common-mb/problem"
	"github.com/vela-ssoc/ssoc-common-mb/shipx"
	"github.com/vela-ssoc/ssoc-common-mb/sqldb"
	"github.com/vela-ssoc/ssoc-common-mb/storage/v2"
	"github.com/vela-ssoc/ssoc-common-mb/validation"
	"github.com/vela-ssoc/ssoc-common/logger"
	"github.com/vela-ssoc/vela-common-mba/netutil"
	"github.com/xgfone/ship/v5"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Run 运行服务
//
//goland:noinspection GoUnhandledErrorResult
func Run(parent context.Context, hide *negotiate.Hide) error {
	// 项目启动时默认初始化一个日志输出，方便启动前调试。
	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelDebug)
	initLogOption := &slog.HandlerOptions{AddSource: true, Level: logLevel}
	tint := logger.NewTint(os.Stdout, initLogOption) // 默认初始化日志输出。
	logHandler := logger.NewMultiHandler(tint)
	log := slog.New(logHandler)
	log.Info("日志组件初始化完毕")

	link, err := telecom.Dial(parent, hide, log) // 与中心端建立连接
	if err != nil {
		return err
	}

	ident := link.Ident()
	issue := link.Issue()
	log.Info("broker接入认证成功", slog.Any("ident", ident), slog.Any("issue", issue))

	logCfg := issue.Logger
	defer logCfg.Close()

	log.Info("日志组件初始化完毕")

	dbCfg := issue.Database

	gormLog := logger.NewGorm(logHandler, gormlogger.Config{LogLevel: gormlogger.Info})
	gormCfg := &gorm.Config{Logger: gormLog}
	db, err := sqldb.Open(dbCfg.DSN, gormCfg)
	if err != nil {
		return err
	}
	sdb, err := db.DB()
	if err != nil {
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer sdb.Close() // 程序结束时断开数据库连接。

	sdb.SetMaxOpenConns(dbCfg.MaxOpenConn)
	sdb.SetMaxIdleConns(dbCfg.MaxIdleConn)
	sdb.SetConnMaxLifetime(dbCfg.MaxLifeTime.Duration())
	sdb.SetConnMaxIdleTime(dbCfg.MaxIdleTime.Duration())
	log.Warn("当前数据库类型", slog.String("dialect", db.Dialector.Name()))

	qry := query.Use(db)
	gfs := gridfs.NewCache(qry, issue.Server.CDN)

	cli := netutil.NewClient()
	match := ntfmatch.NewMatch(qry)
	store := storage.NewStore(qry)

	tunCli := &http.Client{
		Transport: &http.Transport{
			DialContext: link.DialContext,
		},
	}
	dongCli := dong.NewTunnel(tunCli, log)
	devopsCfg := devops.NewConfig(store)
	devCli := devops.NewClient(devopsCfg, cli)
	alert := alarm.UnifyAlerter(store, match, log, dongCli, devCli, qry)

	// manager callback
	name := link.Name()
	pbh := problem.NewHandle(name)

	valid := validation.New()
	if err = valid.RegisterCustomValidations(validation.All()); err != nil {
		return err
	}

	agt := ship.Default()
	mgt := ship.Default()
	mgt.Logger = shipx.NewLog(log)
	mgt.NotFound = pbh.NotFound
	mgt.HandleError = pbh.HandleError
	mgt.Validator = valid
	agt.Logger = shipx.NewLog(log)
	agt.NotFound = pbh.NotFound
	agt.HandleError = pbh.HandleError
	agt.Validator = valid

	mv1 := mgt.Group(accord.PathPrefix).Use(middle.Oplog)
	av1 := agt.Group(accord.PathPrefix).Use(middle.Oplog)

	esCfg := elastic.NewConfigure(qry, name)
	esc := elastic.NewSearch(esCfg, cli)
	cmdbCfg := cmdb.NewConfigure(store)
	cmdbCli := cmdb.NewClient(qry, cmdbCfg, cli)

	sonaCfg := sonatype.HardConfig()
	sonaCli := sonatype.NewClient(sonaCfg, cli)
	vsync := vulnsync.New(db, sonaCli)
	_ = vsync

	nodeEventService := agtsvc.Phase(cmdbCli, alert, log)
	hub := mlink.LinkHub(qry, link, agt, nodeEventService, log)
	_ = hub.ResetDB()

	const consoleDir = "resources/agent/console"
	if err = os.MkdirAll(consoleDir, 0o777); err != nil {
		return err
	}
	pipeFS := pipelog.NewFS(consoleDir, 10*1024*1024, time.Minute)

	minionService := mgtsvc.Minion(qry)
	agentService := mgtsvc.Agent(qry, hub, minionService, store, log)
	nodeEventService.SetService(agentService)

	{
		agentREST := mgtapi.Agent(agentService)
		agentREST.Route(mv1)

		mgtapi.NewAgentConsole(pipeFS).Route(mv1)

		intoService := mgtsvc.Into(hub)
		intoREST := mgtapi.Into(intoService)
		intoREST.Route(mv1)

		resetREST := mgtapi.Reset(store, esCfg, match)
		resetREST.Route(mv1)

		pprofREST := mgtapi.Pprof(link)
		pprofREST.Route(mv1)

		systemSvc := mservice.NewSystem(link, qry, gfs, log)
		taskSvc := mservice.NewTask(qry, hub, log)
		routers := []shipx.RouteBinder{
			mrestapi.NewSystem(systemSvc),
			mrestapi.NewTask(taskSvc),
		}
		if err = shipx.BindRouters(mv1, routers); err != nil {
			return err
		}
	}

	{
		agentConsoleAPI := agtapi.NewAgentConsole(pipeFS)
		agentConsoleAPI.Route(av1)

		agentEmergencySnapshotSvc := agtsvc.NewAgentEmergencySnapshot(qry, log)
		agentEmergencySnapshotAPI := agtapi.NewAgentEmergencySnapshot(agentEmergencySnapshotSvc)
		agentEmergencySnapshotAPI.Route(av1)

		auditorREST := agtapi.Audit(alert)
		auditorREST.Route(av1)

		bpfREST := agtapi.BPF()
		bpfREST.Route(av1)

		collectService := agtsvc.NewCollect(qry)
		collectREST := agtapi.Collect(qry, collectService)
		collectREST.Route(av1)

		elasticREST := agtapi.Elastic(esc)
		elasticREST.Route(av1)

		elkeidFS := bytedance.ElkeidFS("resources/elkeid/", cli)
		agtapi.Reverse(elkeidFS).Route(av1)

		heartREST := agtapi.Heart(qry)
		heartREST.Route(av1)

		proxyAPI := agtapi.Proxy(link.DialContext)
		proxyAPI.Route(av1)

		securityREST := agtapi.Security(qry)
		securityREST.Route(av1)

		streamREST := agtapi.Stream(name, esc)
		streamREST.Route(av1)

		tagService := agtsvc.Tag(qry, agentService)
		tagREST := agtapi.Tag(tagService)
		tagREST.Route(av1)

		taskREST := agtapi.Task(qry)
		taskREST.Route(av1)

		thirdService := agtsvc.NewThird(qry, gfs)
		thirdREST := agtapi.Third(thirdService)
		thirdREST.Route(av1)

		bid := link.Ident().ID
		upgradeREST := agtapi.Upgrade(qry, bid, gfs)
		upgradeREST.Route(av1)

		sharedStringsService := agtsvc.SharedStrings(qry)
		sharedREST := agtapi.Shared(sharedStringsService)
		sharedREST.Route(av1)
	}

	oldHandler := linkhub.New(db, qry, link, log, gfs)
	temp := temporary.REST(oldHandler, valid, log)
	gw := gateway.New(hub, valid)
	deployService := agtsvc.Deploy(qry, store, gfs, ident.ID)
	deployAPI := agtapi.Deploy(deployService)

	mux := ship.Default()
	api := mux.Group("/")
	api.Route("/api/v1/minion").CONNECT(func(c *ship.Context) error {
		gw.ServeHTTP(c.ResponseWriter(), c.Request())
		return nil
	})
	api.Route("/v1/minion/endpoint").GET(temp.Endpoint)
	api.Route("/v1/edition/upgrade").GET(oldHandler.Upgrade)
	api.Route("/api/v1/deploy/minion").GET(deployAPI.Script)
	api.Route("/api/v1/deploy/minion/download").GET(deployAPI.MinionDownload)
	{
		routes := []shipx.RouteBinder{}
		baseAPI := mux.Group("/api/v1")
		if err = shipx.BindRouters(baseAPI, routes); err != nil {
			return err
		}
	}

	errCh := make(chan error, 1)
	// 监听本地端口用于 minion 节点连接
	ds := &daemonServer{issue: issue, hide: hide, handler: mux, errCh: errCh}
	go ds.Run()

	// 连接 manager 的客户端，保持在线与接受指令
	dc := &daemonClient{link: link, handler: mgt, errCh: errCh, log: log, parent: parent}
	go dc.Run()

	select {
	case err = <-errCh:
	case <-parent.Done():
	}

	_ = ds.Close()
	_ = dc.Close()
	_ = hub.ResetDB()

	return err
}
