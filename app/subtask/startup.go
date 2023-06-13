package subtask

//type startupTask struct {
//	id      int64
//	slog    logback.Logger
//	cycle   int
//	timeout time.Duration
//}
//
//func (st *startupTask) Run() {
//	ctx, cancel := context.WithTimeout(context.Background(), st.timeout)
//	defer cancel()
//
//	dat, err := st.find(ctx)
//	if err != nil {
//		st.slog.Warnf("查询 startup 配置出错：%s", err)
//		return
//	}
//
//	st.slog
//}
//
//func (st *startupTask) find(ctx context.Context) (*definition.Startup, error) {
//	tbl := query.Startup
//	dat, err := tbl.WithContext(ctx).Where(tbl.ID.Eq(st.id)).First()
//	if err == nil {
//		ret := st.convert(dat)
//		return ret, nil
//	}
//	if err != gorm.ErrRecordNotFound {
//		return nil, err
//	}
//	// 查询通用配置
//}
//
//func (st *startupTask) convert(dat *model.Startup) *definition.Startup {
//	node := dat.Node
//	cons := dat.Console
//	logg := dat.Logger
//
//	exts := make([]*definition.StartupExtend, 0, 8)
//	for _, e := range dat.Extends {
//		exts = append(exts, &definition.StartupExtend{
//			Name:  e.Name,
//			Type:  e.Type,
//			Value: e.Value,
//		})
//	}
//
//	return &definition.Startup{
//		Node: definition.StartupNode{DNS: node.DNS, Prefix: node.Prefix},
//		Logger: definition.StartupLogger{
//			Level:    logg.Level,
//			Filename: logg.Filename,
//			Console:  logg.Console,
//			Format:   logg.Format,
//			Caller:   logg.Caller,
//			Skip:     logg.Skip,
//		},
//		Console: definition.StartupConsole{
//			Enable:  cons.Enable,
//			Network: cons.Network,
//			Address: cons.Address,
//			Script:  cons.Script,
//		},
//		Extends: exts,
//	}
//}
