package agtsvc

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"gorm.io/gen"
	"gorm.io/gorm/clause"
)

type SharedStringsService interface {
	Get(ctx context.Context, req *param.SharedKey) (*model.KVData, error)
	Set(ctx context.Context, inf mlink.Infer, req *param.SharedKeyValue) (*model.KVData, error)
	Store(ctx context.Context, inf mlink.Infer, req *param.SharedKeyValue) (*model.KVData, error)
	Incr(ctx context.Context, inf mlink.Infer, req *param.SharedKeyIncr) (*model.KVData, error)
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

func (biz *sharedStringsService) Set(ctx context.Context, inf mlink.Infer, req *param.SharedKeyValue) (*model.KVData, error) {
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
			Value:     req.Value,
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

	if req.Audit { // 审计功能
		ident := inf.Ident()
		issue := inf.Issue()
		audit := &model.KVAudit{
			MinionID: issue.ID,
			Inet:     ident.Inet.String(),
			Bucket:   bucket,
			Key:      key,
		}
		_ = query.KVAudit.WithContext(ctx).
			Clauses(clause.OnConflict{UpdateAll: true}).
			Create(audit)
	}
	if req.Reply {
		return biz.Get(ctx, &param.SharedKey{Bucket: bucket, Key: key})
	}

	return nil, nil
}

func (biz *sharedStringsService) Store(ctx context.Context, inf mlink.Infer, req *param.SharedKeyValue) (*model.KVData, error) {
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
			Value:     req.Value,
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

	if req.Audit { // 审计功能
		ident := inf.Ident()
		issue := inf.Issue()
		audit := &model.KVAudit{
			MinionID: issue.ID,
			Inet:     ident.Inet.String(),
			Bucket:   bucket,
			Key:      key,
		}
		_ = query.KVAudit.WithContext(ctx).
			Clauses(clause.OnConflict{UpdateAll: true}).
			Create(audit)
	}
	if req.Reply {
		return biz.Get(ctx, &param.SharedKey{Bucket: bucket, Key: key})
	}

	return nil, nil
}

func (biz *sharedStringsService) Incr(ctx context.Context, inf mlink.Infer, req *param.SharedKeyIncr) (*model.KVData, error) {
	now := time.Now()
	n, lifetime := req.N, req.Lifetime
	if n == 0 {
		n = 1
	}
	bucket, key := req.Bucket, req.Key

	old := biz.find(ctx, bucket, key)
	if lifetime <= 0 {
		if old != nil {
			lifetime = old.Lifetime
		} else {
			lifetime = 0
		}
	}

	expiredAt := now.Add(lifetime)
	tbl := query.KVData
	if old == nil || old.Expired(now) { // 如果不存在老数据则直接插入
		if old != nil {
			if _, err := tbl.WithContext(ctx).
				Where(tbl.Bucket.Eq(bucket), tbl.Key.Eq(key)).
				Delete(); err != nil {
				return nil, err
			}
		}
		dat := &model.KVData{
			Bucket:    bucket,
			Key:       key,
			Count:     n,
			ExpiredAt: expiredAt,
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
				tbl.Lifetime.Value(int64(lifetime)),
				tbl.ExpiredAt.Value(expiredAt),
				tbl.UpdatedAt.Value(now),
				tbl.Version.Value(old.Version+1),
			); err != nil {
			return nil, err
		}
	}

	if req.Audit { // 审计功能
		ident := inf.Ident()
		issue := inf.Issue()
		audit := &model.KVAudit{
			MinionID: issue.ID,
			Inet:     ident.Inet.String(),
			Bucket:   bucket,
			Key:      key,
		}
		_ = query.KVAudit.WithContext(ctx).
			Clauses(clause.OnConflict{UpdateAll: true}).
			Create(audit)
	}

	return biz.Get(ctx, &param.SharedKey{Bucket: bucket, Key: key})
}

func (biz *sharedStringsService) Del(ctx context.Context, req *param.SharedKey) error {
	tbl := query.KVData
	cond := []gen.Condition{tbl.Bucket.Eq(req.Bucket)}
	if key := req.Key; key != "" {
		cond = append(cond, tbl.Key.Eq(key))
	}

	_, err := tbl.WithContext(ctx).
		Where(cond...).
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
