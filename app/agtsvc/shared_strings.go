package agtsvc

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-broker/bridge/mlink"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SharedStringsService interface {
	Get(ctx context.Context, keys []*param.SharedKey) ([]*model.KVData, error)
	Del(ctx context.Context, keys []*param.SharedKey) error
	Set(ctx context.Context, values []*param.SharedKeyValue, inf mlink.Infer) error
	Incr(ctx context.Context, values []*param.SharedKeyIncr, inf mlink.Infer) ([]*model.KVData, error)
}

func SharedStrings() SharedStringsService {
	return &sharedStringsService{}
}

type sharedStringsService struct{}

func (biz *sharedStringsService) Get(ctx context.Context, keys []*param.SharedKey) ([]*model.KVData, error) {
	size := len(keys)
	if size == 0 {
		return []*model.KVData{}, nil
	}

	// SELECT *
	// FROM kv_data
	// WHERE ((lifetime > 0 AND expired_at >= CURRENT_TIMESTAMP) OR lifetime <= 0)
	//  AND ((bucket = 'test' AND `key` = 'a') OR (bucket = 'test' AND `key` = 'b'));
	now := time.Now()
	tbl := query.KVData
	tblCtx := tbl.WithContext(ctx)
	survival := tblCtx.Or(tbl.Lifetime.Lte(0)).
		Or(tblCtx.Where(tbl.Lifetime.Gt(0), tbl.ExpiredAt.Gte(now)))

	filter := tblCtx.Or(tbl.Bucket.Eq(keys[0].Bucket), tbl.Key.Eq(keys[0].Key))
	for _, bk := range keys[1:] {
		filter.Or(tbl.Bucket.Eq(bk.Bucket), tbl.Key.Eq(bk.Key))
	}
	dats, err := tblCtx.Where(survival, filter).Find()

	return dats, err
}

func (biz *sharedStringsService) Del(ctx context.Context, keys []*param.SharedKey) error {
	size := len(keys)
	if size == 0 {
		return nil
	}

	tbl := query.KVData
	tblCtx := tbl.WithContext(ctx)
	head := keys[0]
	filter := tblCtx.Or(tbl.Bucket.Eq(head.Bucket), tbl.Key.Eq(head.Key))
	for _, bk := range keys[1:] {
		filter.Or(tbl.Bucket.Eq(bk.Bucket), tbl.Key.Eq(bk.Key))
	}
	_, err := tblCtx.Where(filter).Delete()

	return err
}

func (biz *sharedStringsService) Set(ctx context.Context, values []*param.SharedKeyValue, inf mlink.Infer) error {
	size := len(values)
	if size == 0 {
		return nil
	}

	return query.Q.Transaction(func(tx *query.Query) error {
		tbl := tx.KVData
		tblCtx := tbl.WithContext(ctx)
		now := time.Now()
		var err error
		for _, val := range values {
			lifetime := val.Lifetime
			dat := &model.KVData{
				Bucket:    val.Bucket,
				Key:       val.Key,
				Value:     val.Value,
				Lifetime:  lifetime,
				ExpiredAt: now.Add(val.Lifetime),
			}

			assigns := map[string]any{"value": val.Value}
			if lifetime > 0 {
				assigns["lifetime"] = lifetime
				assigns["expired_at"] = gorm.Expr("TIMESTAMPADD(MICROSECOND, ?, CURRENT_TIMESTAMP)", lifetime.Microseconds())
			} else {
				// 1us = 1000ns
				assigns["expired_at"] = gorm.Expr("TIMESTAMPADD(MICROSECOND, ?, CURRENT_TIMESTAMP)", gorm.Expr("lifetime/1000"))
			}
			conflict := clause.OnConflict{DoUpdates: clause.Assignments(assigns)}
			if err = tblCtx.UnderlyingDB().Clauses(conflict).Create(dat).Error; err != nil {
				break
			}
		}

		return err
	})
}

func (biz *sharedStringsService) Incr(ctx context.Context, values []*param.SharedKeyIncr, inf mlink.Infer) ([]*model.KVData, error) {
	size := len(values)
	if size == 0 {
		return []*model.KVData{}, nil
	}

	err := query.Q.Transaction(func(tx *query.Query) error {
		tbl := tx.KVData
		tblCtx := tbl.WithContext(ctx)
		now := time.Now()
		var err error
		for _, val := range values {
			n := val.N
			if n == 0 {
				n = 1
			}

			lifetime := val.Lifetime
			dat := &model.KVData{
				Bucket:    val.Bucket,
				Key:       val.Key,
				Count:     n,
				Lifetime:  lifetime,
				ExpiredAt: now.Add(val.Lifetime),
			}

			assigns := map[string]any{"count": gorm.Expr("`count` + ?", n)}
			if lifetime > 0 {
				assigns["lifetime"] = lifetime
				assigns["expired_at"] = gorm.Expr("TIMESTAMPADD(MICROSECOND, ?, CURRENT_TIMESTAMP)", lifetime.Microseconds())
			} else {
				// 1us = 1000ns
				assigns["expired_at"] = gorm.Expr("TIMESTAMPADD(MICROSECOND, ?, CURRENT_TIMESTAMP)", gorm.Expr("lifetime/1000"))
			}
			conflict := clause.OnConflict{DoUpdates: clause.Assignments(assigns)}
			if err = tblCtx.UnderlyingDB().Clauses(conflict).Create(dat).Error; err != nil {
				break
			}
		}

		return err
	})
	if err != nil {
		return nil, err
	}

	// 查询回显数据
	tbl := query.KVData
	tblCtx := tbl.WithContext(ctx)
	head := values[0]
	filter := tblCtx.Or(tbl.Bucket.Eq(head.Bucket), tbl.Key.Eq(head.Key))
	for _, bk := range values[1:] {
		filter.Or(tbl.Bucket.Eq(bk.Bucket), tbl.Key.Eq(bk.Key))
	}
	dats, err := tblCtx.Where(filter).Find()

	return dats, err
}
