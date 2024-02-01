package agtsvc

import (
	"context"
	"fmt"
	"time"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SharedStringsService interface {
	Get(ctx context.Context, req *param.SharedKey) (*model.KVData, error)
	Set(ctx context.Context, req *param.SharedKeyValue) (*model.KVData, error)
	Store(ctx context.Context, req *param.SharedKeyValue) (*model.KVData, error)
	Incr(ctx context.Context, req *param.SharedKeyIncr, inf mlink.Infer) (*model.KVData, error)
	Del(ctx context.Context, req *param.SharedKey) error
}

func SharedStrings() SharedStringsService {
	return &sharedStringsService{}
}

type sharedStringsService struct{}

func (biz *sharedStringsService) Get(ctx context.Context, req *param.SharedKey) (*model.KVData, error) {
	// SELECT *
	// FROM kv_data
	// WHERE bucket = ?
	//   AND `key` = ?
	//   AND (lifetime <= 0 OR (lifetime > 0 AND expired_at >= ?));

	now := time.Now()
	tbl := query.KVData
	tblCtx := tbl.WithContext(ctx)

	survival := tblCtx.Or(tbl.Lifetime.Gte(0)).
		Or(tbl.Lifetime.Gt(0), tbl.ExpiredAt.Gte(now))

	return tbl.WithContext(ctx).
		Where(tbl.Bucket.Eq(req.Bucket), tbl.Key.Eq(req.Key)).
		Where(survival).
		First()
}

func (biz *sharedStringsService) Set(ctx context.Context, req *param.SharedKeyValue) (*model.KVData, error) {
	// 1. 当 req.lifetime <= 0 时
	//    ├─ 如果没有数据：直接插入一条不过期的数据。
	//    └─ 如果存在数据（old）：
	//        ├─ 如果 old.lifetime <= 0，不更新过期时间。
	//        └─ 如果 old.lifetime > 0，按照 old.lifetime 续期。
	// 2. 当 req.lifetime > 0 时，按照 req.lifetime 续期

	now := time.Now()
	bucket, key, lifetime := req.Bucket, req.Key, req.Lifetime
	if lifetime < 0 {
		lifetime = 0
	}

	tbl := query.KVData
	old := biz.find(ctx, bucket, key) // 查询数据库中的数据

	if old == nil || old.Expired(now) {
		if old != nil {
			if _, err := tbl.WithContext(ctx).
				Where(tbl.Bucket.Eq(bucket), tbl.Key.Eq(key), tbl.Version.Eq(old.Version)).
				Delete(); err != nil {
				return nil, err
			}
		}
		dat := &model.KVData{
			Bucket:    bucket,
			Key:       key,
			Lifetime:  lifetime,
			ExpiredAt: now.Add(lifetime),
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}
		if err := tbl.WithContext(ctx).Create(dat); err != nil {
			return nil, err
		}
	} else {
		if lifetime <= 0 {
			lifetime = old.Lifetime
		}

		if _, err := tbl.WithContext(ctx).
			Where(tbl.Bucket.Eq(bucket), tbl.Key.Eq(key), tbl.Version.Eq(old.Version)).
			UpdateSimple(
				tbl.Value.Value(req.Value),
				tbl.ExpiredAt.Value(now.Add(lifetime)),
				tbl.UpdatedAt.Value(now),
				tbl.Version.Value(old.Version+1),
			); err != nil {
			return nil, err
		}
	}

	if req.Reply {
		return biz.Get(ctx, &param.SharedKey{Bucket: bucket, Key: key})
	}

	return nil, nil
}

func (biz *sharedStringsService) Store(ctx context.Context, req *param.SharedKeyValue) (*model.KVData, error) {
	now := time.Now()
	bucket, key, lifetime := req.Bucket, req.Key, req.Lifetime
	if lifetime < 0 {
		lifetime = 0
	}

	tbl := query.KVData
	old := biz.find(ctx, bucket, key) // 查询数据库中的数据

	if old == nil || old.Expired(now) {
		if old != nil {
			if _, err := tbl.WithContext(ctx).
				Where(tbl.Bucket.Eq(bucket), tbl.Key.Eq(key), tbl.Version.Eq(old.Version)).
				Delete(); err != nil {
				return nil, err
			}
		}
		dat := &model.KVData{
			Bucket:    bucket,
			Key:       key,
			Lifetime:  lifetime,
			ExpiredAt: now.Add(lifetime),
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}
		if err := tbl.WithContext(ctx).Create(dat); err != nil {
			return nil, err
		}
	} else {
		if _, err := tbl.WithContext(ctx).
			Where(tbl.Bucket.Eq(bucket), tbl.Key.Eq(key), tbl.Version.Eq(old.Version)).
			UpdateSimple(
				tbl.Value.Value(req.Value),
				tbl.UpdatedAt.Value(now),
				tbl.Version.Value(old.Version+1),
			); err != nil {
			return nil, err
		}
	}

	if req.Reply {
		return biz.Get(ctx, &param.SharedKey{Bucket: bucket, Key: key})
	}

	return nil, nil
}

func (biz *sharedStringsService) Set1(ctx context.Context, values []*param.SharedKeyValue, inf mlink.Infer) error {
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

func (biz *sharedStringsService) Incr(ctx context.Context, req *param.SharedKeyIncr, _ mlink.Infer) (*model.KVData, error) {
	now := time.Now()
	bucket, key := req.Bucket, req.Key
	old := biz.find(ctx, bucket, key)

	n := req.N
	if n == 0 {
		n = 1
	}

	tbl := query.KVData
	if old == nil || old.Expired(now) { // 如果不存在老数据则直接插入
		if old != nil {
			if _, err := tbl.WithContext(ctx).
				Where(tbl.Bucket.Eq(bucket), tbl.Key.Eq(key), tbl.Version.Eq(old.Version)).
				Delete(); err != nil {
				return nil, err
			}
		}
		dat := &model.KVData{
			Bucket:    bucket,
			Key:       key,
			Count:     n,
			ExpiredAt: now,
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}
		if err := tbl.WithContext(ctx).Create(dat); err != nil {
			return nil, err
		}
	} else {
		if _, err := tbl.WithContext(ctx).
			Where(tbl.Bucket.Eq(bucket), tbl.Key.Eq(key), tbl.Version.Eq(old.Version)).
			UpdateSimple(
				tbl.Count.Value(old.Count+n),
				tbl.UpdatedAt.Value(now),
				tbl.Version.Value(old.Version+1),
			); err != nil {
			return nil, err
		}
	}

	return biz.Get(ctx, &param.SharedKey{Bucket: bucket, Key: key})
}

func (biz *sharedStringsService) Del(ctx context.Context, req *param.SharedKey) error {
	tbl := query.KVData
	_, err := tbl.WithContext(ctx).
		Where(tbl.Bucket.Eq(req.Bucket), tbl.Key.Eq(req.Key)).
		Delete()

	return err
}

func (biz *sharedStringsService) find(ctx context.Context, bucket, key string) *model.KVData {
	tbl := query.KVData
	dat, _ := tbl.WithContext(ctx).
		Where(tbl.Bucket.Eq(bucket), tbl.Key.Eq(key)).
		First()

	return dat
}

func (biz *sharedStringsService) Testing() {
	_, err := biz.Get(context.Background(), &param.SharedKey{Bucket: "bucket", Key: "key"})
	fmt.Print(err)
}
