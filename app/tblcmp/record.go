package tblcmp

import (
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
)

type Record struct {
	id   int64
	inet string
	clam bool
	subs []*model.Substance
}

func (r Record) ID() int64 {
	return r.id
}

func (r Record) Inet() string {
	return r.inet
}

func (r Record) Clam() bool {
	return r.clam
}

func (r Record) Compare(dat *param.TaskReport) *param.TaskDiff {
	hm := dat.IDMap()
	removes := make([]int64, 0, 10)
	updates := make([]*param.TaskChunk, 0, 10)
	for _, sub := range r.subs {
		id := sub.ID
		rp, ok := hm[id]
		delete(hm, id)
		if ok && rp.Hash == sub.Hash {
			continue
		}

		chk := &param.TaskChunk{
			ID:      id,
			Name:    sub.Name,
			Dialect: sub.MinionID == r.id,
			Hash:    sub.Hash,
			Chunk:   sub.Chunk,
		}
		updates = append(updates, chk)
	}

	for _, rp := range hm {
		if rp.From == "tunnel" {
			removes = append(removes, rp.ID)
		}
	}

	diff := &param.TaskDiff{
		Removes: removes,
		Updates: updates,
	}

	return diff
}
