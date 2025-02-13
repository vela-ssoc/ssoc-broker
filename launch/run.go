package launch

import (
	"context"
	"log/slog"
	"net/http"
	"runtime"

	"github.com/vela-ssoc/vela-broker/appv2/manager/mrestapi"
	"github.com/vela-ssoc/vela-broker/appv2/manager/mservice"

	"github.com/vela-ssoc/vela-broker/app/agtapi"
	"github.com/vela-ssoc/vela-broker/app/agtsvc"
	"github.com/vela-ssoc/vela-broker/app/crontbl"
	"github.com/vela-ssoc/vela-broker/app/mgtapi"
	"github.com/vela-ssoc/vela-broker/app/mgtsvc"
	"github.com/vela-ssoc/vela-broker/app/middle"
	"github.com/vela-ssoc/vela-broker/app/temporary"
	"github.com/vela-ssoc/vela-broker/app/temporary/linkhub"
	"github.com/vela-ssoc/vela-broker/bridge/gateway"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-broker/bridge/telecom"
	"github.com/vela-ssoc/vela-broker/foreign/bytedance"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/vela-ssoc/vela-common-mb/dal/gridfs"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/dbms"
	"github.com/vela-ssoc/vela-common-mb/integration/alarm"
	"github.com/vela-ssoc/vela-common-mb/integration/cmdb"
	"github.com/vela-ssoc/vela-common-mb/integration/devops"
	"github.com/vela-ssoc/vela-common-mb/integration/dong/v2"
	"github.com/vela-ssoc/vela-common-mb/integration/elastic"
	"github.com/vela-ssoc/vela-common-mb/integration/ntfmatch"
	"github.com/vela-ssoc/vela-common-mb/integration/sonatype"
	"github.com/vela-ssoc/vela-common-mb/integration/vulnsync"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mb/problem"
	"github.com/vela-ssoc/vela-common-mb/storage/v2"
	"github.com/vela-ssoc/vela-common-mb/validate"
	"github.com/vela-ssoc/vela-common-mba/netutil"
	"github.com/xgfone/ship/v5"
)

// Run 运行服务
func Run(parent context.Context, hide telecom.Hide, oldLog logback.Logger) error {
	link, err := telecom.Dial(parent, hide, oldLog) // 与中心端建立连接
	if err != nil {
		return err
	}

	ident := link.Ident()
	issue := link.Issue()
	oldLog.Infof("broker 接入认证成功，上报认证信息如下：\n%s\n下发的配置如下：\n%s", ident, issue)

	logCfg := issue.Logger
	zlg := logCfg.Zap() // 根据配置文件初始化日志
	oldLog.Replace(zlg) // 替换日志输出内核
	gormLog := logback.Gorm(zlg, logCfg.Level)

	dbCfg := issue.Database
	db, sdb, err := dbms.Open(dbCfg, gormLog)
	if err != nil {
		return err
	}
	qry := query.Use(db)
	gfs := gridfs.NewCache(sdb, issue.Section.CDN)

	cli := netutil.NewClient()
	match := ntfmatch.NewMatch(qry)
	store := storage.NewStore(qry)

	tunCli := &http.Client{
		Transport: &http.Transport{
			DialContext: link.DialContext,
		},
	}
	dongCli := dong.NewTunnel(tunCli, oldLog)
	devopsCfg := devops.NewConfig(store)
	devCli := devops.NewClient(devopsCfg, cli)
	alert := alarm.UnifyAlerter(store, match, oldLog, dongCli, devCli)

	// manager callback
	name := link.Name()
	pbh := problem.NewHandle(name)

	valid := validate.New()
	agt := ship.Default()
	mgt := ship.Default()
	mgt.Logger = oldLog
	mgt.NotFound = pbh.NotFound
	mgt.HandleError = pbh.HandleError
	mgt.Validator = valid
	agt.Logger = oldLog
	agt.NotFound = pbh.NotFound
	agt.HandleError = pbh.HandleError
	agt.Validator = valid

	mv1 := mgt.Group(accord.PathPrefix).Use(middle.Oplog)
	av1 := agt.Group(accord.PathPrefix).Use(middle.Oplog)

	esCfg := elastic.NewConfigure(qry, name)
	esc := elastic.NewSearch(esCfg, cli)
	cmdbCfg := cmdb.NewConfigure(store)
	cmdbCli := cmdb.NewClient(qry, cmdbCfg, cli, oldLog)

	sonaCfg := sonatype.HardConfig()
	sonaCli := sonatype.NewClient(sonaCfg, cli)
	vsync := vulnsync.New(db, sonaCli)
	_ = vsync

	nodeEventService := agtsvc.Phase(cmdbCli, alert, oldLog)
	hub := mlink.LinkHub(qry, link, agt, nodeEventService, oldLog)
	_ = hub.ResetDB()

	minionService := mgtsvc.Minion(qry)
	agentService := mgtsvc.Agent(qry, hub, minionService, store, oldLog)
	nodeEventService.SetService(agentService)

	// manager api
	log := slog.New(slog.DiscardHandler) // 暂时这么弄
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

		taskSvc := mservice.NewTask(qry, hub, log)
		taskAPI := mrestapi.NewTask(taskSvc)
		taskAPI.Route(mv1)
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

	oldHandler := linkhub.New(db, qry, link, oldLog, gfs)
	temp := temporary.REST(oldHandler, valid, oldLog)
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
		crontbl.Run(parent, qry, link.Ident().ID, link.Issue().Name, oldLog)
	}

	errCh := make(chan error, 1)
	// 监听本地端口用于 minion 节点连接
	ds := &daemonServer{listen: issue.Listen, hide: hide, handler: mux, errCh: errCh}
	go ds.Run()

	// 连接 manager 的客户端，保持在线与接受指令
	dc := &daemonClient{link: link, handler: mgt, errCh: errCh, slog: oldLog, parent: parent}
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
