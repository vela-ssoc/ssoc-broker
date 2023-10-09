package crontbl

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mb/sigar"
)

func Run(ctx context.Context, id int64, name string, log logback.Logger) {
	bs := &brokerStat{
		id:   id,
		name: name,
	}

	ct := cronTask{
		ctx:  ctx,
		idle: 10 * time.Second,
		log:  log,
		fn:   bs.Func,
	}

	go ct.Run()
}

type brokerStat struct {
	id   int64
	name string
}

func (bs *brokerStat) Func(parent context.Context, at time.Time) error {
	ctx, cancel := context.WithTimeout(parent, time.Minute)
	defer cancel()

	mem, err := sigar.Memory()
	if err != nil {
		return err
	}
	percent, err := sigar.CPUPercent(ctx, time.Second)
	if err != nil {
		return err
	}

	data := &model.BrokerStat{
		ID:         bs.id,
		Name:       bs.name,
		MemUsed:    mem.Used(),
		MemTotal:   mem.MemTotal,
		CPUPercent: percent,
	}

	return query.BrokerStat.WithContext(ctx).Save(data)
}
