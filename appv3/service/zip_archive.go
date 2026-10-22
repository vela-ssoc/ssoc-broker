package service

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/ssoc-common-mb/datalayer/dbzipfs"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/model"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
)

type ZipArchive struct {
	db  polydb.Sourcer
	log *slog.Logger
}

func NewZipArchive(db polydb.Sourcer, log *slog.Logger) *ZipArchive {
	return &ZipArchive{
		db:  db,
		log: log,
	}
}

func (za *ZipArchive) Open(ctx context.Context, name string) (*model.ZipArchive, *dbzipfs.Zip, error) {
	db := za.db.MDB()
	coll := db.ZipArchive()
	archive, err := coll.FindByName(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	zfs := dbzipfs.NewZip(ctx, db, archive.ID)

	return archive, zfs, nil
}
