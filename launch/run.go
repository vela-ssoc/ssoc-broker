package launch

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/vela-ssoc/ssoc-broker/app/agtapi"
	"github.com/vela-ssoc/ssoc-broker/app/agtsvc"
	"github.com/vela-ssoc/ssoc-broker/app/mgtapi"
	"github.com/vela-ssoc/ssoc-broker/app/mgtsvc"
	"github.com/vela-ssoc/ssoc-broker/app/middle"
	"github.com/vela-ssoc/ssoc-broker/app/temporary"
	"github.com/vela-ssoc/ssoc-broker/app/temporary/linkhub"
	manapi "github.com/vela-ssoc/ssoc-broker/application/manager/restapi"
	mansvc "github.com/vela-ssoc/ssoc-broker/application/manager/service"
	pgmcron "github.com/vela-ssoc/ssoc-broker/application/program/cronjob"
	pgmsvc "github.com/vela-ssoc/ssoc-broker/application/program/service"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	shipxv3 "github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-broker/bridge/gateway"
	"github.com/vela-ssoc/ssoc-broker/bridge/mlink"
	"github.com/vela-ssoc/ssoc-broker/bridge/telecom"
	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-broker/foreign/bytedance"
	"github.com/vela-ssoc/ssoc-broker/library/pipelog"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokclient"
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
	"github.com/vela-ssoc/ssoc-common-mb/sqldb"
	"github.com/vela-ssoc/ssoc-common-mb/storage/v2"
	"github.com/vela-ssoc/ssoc-common-mb/validation"
	"github.com/vela-ssoc/ssoc-common/banner"
	"github.com/vela-ssoc/ssoc-common/cronv3"
	"github.com/vela-ssoc/ssoc-common/logger"
	"github.com/vela-ssoc/ssoc-common/vmetric"
	"github.com/vela-ssoc/vela-common-mba/netutil"
	"github.com/xgfone/ship/v5"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Run 运行服务
//
//goland:noinspection GoUnhandledErrorResult
func Run(parent context.Context, parse config.Parser) error {
	// 项目启动时默认初始化一个日志输出，方便启动前调试。
	initLogOption := &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}
	logHandler := logger.NewMultiHandler(logger.NewTint(os.Stdout, initLogOption))
	log := slog.New(logHandler)
	log.Info("临时日志组件初始化完毕")

	valid := validation.New()
	if err := valid.RegisterCustomValidations(validation.All()); err != nil {
		return err
	}

	// 读取配置文件
	hidecfg := new(config.HideConfig)
	if err := parse.Parse(parent, hidecfg); err != nil {
		log.Error("读取启动配置文件出错", "err", err)
		return err
	}

	agt := ship.Default()
	mgt := ship.Default()

	version := banner.Version()
	brokopts := brokclient.Options{
		Addresses:  hidecfg.Addresses,
		Secret:     hidecfg.Secret,
		Version:    version,
		Handler:    mgt,
		Validator:  valid.Validate,
		DialConfig: nil,
	}
	hide := &negotiate.Hide{Secret: hidecfg.Secret}
	for _, addr := range hidecfg.Addresses {
		hide.Servers = append(hide.Servers, &netutil.Address{Addr: addr, TLS: true})
		hide.Servers = append(hide.Servers, &netutil.Address{Addr: addr})
	}

	// 兼容老版本接口
	link, err := telecom.Dial(parent, hide, version, log) // 与中心端建立连接
	if err != nil {
		return err
	}

	ident, issue := link.Ident(), link.Issue()
	log.Info("broker接入认证成功", "ident", ident, "issue", issue)

	muxv3, err := brokclient.Open(parent, brokopts)
	if err != nil {
		return err
	}
	defer muxv3.Close()

	dbCfg := issue.Database
	logCfg := issue.Logger
	logHandler.Replace() // 清空初始化日志
	logLevel := new(slog.LevelVar)
	_ = logLevel.UnmarshalText([]byte(dbCfg.Level))
	logOpt := &slog.HandlerOptions{AddSource: true, Level: logLevel}
	if logCfg.Console {
		h := logger.NewTint(os.Stdout, logOpt)
		logHandler.Append(h)
	}
	if lum := logCfg.Lumber(); lum != nil {
		defer lum.Close()
		h := slog.NewJSONHandler(lum, logOpt)
		logHandler.Append(h)
	}

	log.Info("日志组件初始化完毕")
	gormLog := sqldb.NewGormLog(logHandler, gormlogger.Config{LogLevel: gormlogger.Warn})
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

	crontab := cronv3.New(log)
	crontab.Start()
	defer crontab.Stop()

	strID := strconv.FormatInt(issue.ID, 10)
	extraLabels := "instance=" + strconv.Quote(strID) +
		",instance_type=" + strconv.Quote("broker") +
		",instance_name=" + strconv.Quote(issue.Name)

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
	agtv3 := agt.Group("/api/v3").Use(middle.Oplog)
	mgtv3 := mgt.Group("/api/v3").Use(middle.Oplog)

	esCfg := elastic.NewConfigure(qry, name)
	esc := elastic.NewSearch(esCfg, cli)
	cmdbCfg := cmdb.NewConfigure(store)
	cmdbCli := cmdb.NewClient(qry, cmdbCfg, cli)

	metricWriters := []vmetric.MetricWriter{vmetric.NewPsutil()}
	pgmVictoriaMetricsConfigSvc := pgmsvc.NewVictoriaMetricsConfig(qry, extraLabels, log)
	pgmMetricsTask := pgmcron.NewMetrics(pgmVictoriaMetricsConfigSvc, metricWriters)
	_ = pgmMetricsTask
	// crontab.AddTask(pgmMetricsTask)

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

		systemSvc := mansvc.NewSystem(link, qry, gfs, log)
		taskSvc := mansvc.NewTask(qry, hub, log)
		routers := []shipx.RouteBinder{
			manapi.NewSystem(systemSvc),
			manapi.NewTask(taskSvc),
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

		bid := issue.ID
		upgradeREST := agtapi.Upgrade(qry, bid, gfs)
		upgradeREST.Route(av1)

		sharedStringsService := agtsvc.SharedStrings(qry)
		sharedREST := agtapi.Shared(sharedStringsService)
		sharedREST.Route(av1)
	}

	oldHandler := linkhub.New(db, qry, link, log, gfs)
	temp := temporary.REST(oldHandler, valid, log)
	gw := gateway.New(hub, valid)
	deployService := agtsvc.Deploy(qry, store, gfs, issue.ID)
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
		var routes []shipx.RouteBinder
		baseAPI := mux.Group("/api/v1")
		if err = shipx.BindRouters(baseAPI, routes); err != nil {
			return err
		}
	}

	routev3 := shipxv3.RouteGroup{
		AgentV3:   agtv3,
		ManagerV3: mgtv3,
		Expose:    mux,
	}
	input := v3Input{
		route: routev3,
		issue: issue,
		db:    db,
		mux:   muxv3,
		hide:  *hidecfg,
		cron:  crontab,
		log:   log,
	}
	log.Info("新版配置文件", "v3", muxv3.Config())
	if err = runV3(parent, input); err != nil {
		return err
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
