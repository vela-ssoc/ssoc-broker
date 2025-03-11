package launch

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"runtime"

	"github.com/vela-ssoc/vela-broker/app/agtapi"
	"github.com/vela-ssoc/vela-broker/app/agtsvc"
	"github.com/vela-ssoc/vela-broker/app/crontbl"
	"github.com/vela-ssoc/vela-broker/app/mgtapi"
	"github.com/vela-ssoc/vela-broker/app/mgtsvc"
	"github.com/vela-ssoc/vela-broker/app/middle"
	"github.com/vela-ssoc/vela-broker/app/temporary"
	"github.com/vela-ssoc/vela-broker/app/temporary/linkhub"
	"github.com/vela-ssoc/vela-broker/appv2/manager/mrestapi"
	"github.com/vela-ssoc/vela-broker/appv2/manager/mservice"
	"github.com/vela-ssoc/vela-broker/bridge/gateway"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-broker/bridge/telecom"
	"github.com/vela-ssoc/vela-broker/foreign/bytedance"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/vela-ssoc/vela-common-mb/dal/gridfs"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/integration/alarm"
	"github.com/vela-ssoc/vela-common-mb/integration/cmdb"
	"github.com/vela-ssoc/vela-common-mb/integration/devops"
	"github.com/vela-ssoc/vela-common-mb/integration/dong/v2"
	"github.com/vela-ssoc/vela-common-mb/integration/elastic"
	"github.com/vela-ssoc/vela-common-mb/integration/ntfmatch"
	"github.com/vela-ssoc/vela-common-mb/integration/sonatype"
	"github.com/vela-ssoc/vela-common-mb/integration/vulnsync"
	"github.com/vela-ssoc/vela-common-mb/param/negotiate"
	"github.com/vela-ssoc/vela-common-mb/problem"
	"github.com/vela-ssoc/vela-common-mb/profile"
	"github.com/vela-ssoc/vela-common-mb/shipx"
	"github.com/vela-ssoc/vela-common-mb/sqldb"
	"github.com/vela-ssoc/vela-common-mb/storage/v2"
	"github.com/vela-ssoc/vela-common-mb/validate"
	"github.com/vela-ssoc/vela-common-mba/netutil"
	"github.com/xgfone/ship/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Run 运行服务
func Run(parent context.Context, hide *negotiate.Hide) error {
	tempLogCfg := profile.Logger{Console: true}
	logWriter := tempLogCfg.LogWriter()
	logOption := &slog.HandlerOptions{AddSource: true, Level: logWriter.Level()}
	logHandler := slog.NewJSONHandler(logWriter, logOption)
	log := slog.New(logHandler)

	link, err := telecom.Dial(parent, hide, log) // 与中心端建立连接
	if err != nil {
		return err
	}

	ident := link.Ident()
	issue := link.Issue()
	log.Info("broker接入认证成功", slog.Any("ident", ident), slog.Any("issue", issue))

	logCfg := issue.Logger
	//goland:noinspection GoUnhandledErrorResult
	defer logCfg.Close()

	logWriter.Discard()
	if logCfg.Console {
		logWriter.Attach(os.Stdout)
	}
	if lumber := logCfg.Logger; lumber != nil && lumber.Filename != "" {
		logWriter.Attach(lumber)
	}
	_ = logWriter.Level().UnmarshalText([]byte(logCfg.Level))
	log.Info("日志组件初始化完毕")

	dbCfg := issue.Database
	gormLogLevel := sqldb.MappingGormLogLevel(dbCfg.Level)
	gormLog, _ := sqldb.NewLog(logWriter, logger.Config{LogLevel: gormLogLevel})
	gormCfg := &gorm.Config{Logger: gormLog}
	db, err := sqldb.Open(dbCfg.DSN, log, gormCfg)
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

	valid := validate.New()
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

	minionService := mgtsvc.Minion(qry)
	agentService := mgtsvc.Agent(qry, hub, minionService, store, log)
	nodeEventService.SetService(agentService)

	{
		agentREST := mgtapi.Agent(agentService)
		agentREST.Route(mv1)

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
		auditorREST := agtapi.Audit(alert)
		auditorREST.Route(av1)

		bpfREST := agtapi.BPF()
		bpfREST.Route(av1)

		collectService := agtsvc.Collect(qry)
		collectREST := agtapi.Collect(qry, collectService)
		collectREST.Route(av1)

		elasticREST := agtapi.Elastic(esc)
		elasticREST.Route(av1)

		elkeidFS := bytedance.ElkeidFS("resources/elkeid/", cli)
		agtapi.Reverse(elkeidFS).Route(av1)

		heartREST := agtapi.Heart()
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

		thirdService := agtsvc.Third(qry, gfs)
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
	gw := gateway.New(hub)
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
	if runtime.GOOS != "windows" {
		crontbl.Run(parent, qry, link.Ident().ID, link.Issue().Name, log)
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
