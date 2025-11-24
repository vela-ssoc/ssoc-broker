package agtsvc

import (
	"context"
	"path"

	"github.com/vela-ssoc/ssoc-common-mb/dal/gridfs"
	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"gorm.io/gen"
)

func NewThird(qry *query.Query, gfs gridfs.FS) *Third {
	return &Third{
		qry: qry,
		gfs: gfs,
	}
}

type Third struct {
	qry *query.Query
	gfs gridfs.FS
}

func (thr *Third) Open(ctx context.Context, name string) (*model.Third, gridfs.File, error) {
	tbl := thr.qry.Third
	th, err := tbl.WithContext(ctx).Where(tbl.Name.Eq(name)).First()
	if err != nil {
		return nil, nil, err
	}

	file, err := thr.gfs.OpenID(th.FileID)
	if err != nil {
		return nil, nil, err
	}

	return th, file, nil
}

func (thr *Third) List(ctx context.Context, customized, pattern string) ([]*model.Third, error) {
	tbl := thr.qry.Third
	dao := tbl.WithContext(ctx)
	var wheres []gen.Condition
	if customized != "" {
		wheres = append(wheres, tbl.Customized.Eq(customized))
	}

	thirds, err := dao.Where(wheres...).Find()
	if err != nil || pattern == "" {
		return thirds, err
	}

	ret := make([]*model.Third, 0, len(thirds))
	for _, third := range thirds {
		name := third.Name
		if matched, _ := path.Match(pattern, name); matched {
			ret = append(ret, third)
		}
	}

	return ret, nil
}
