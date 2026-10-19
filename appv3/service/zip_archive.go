package service

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/ssoc-common-mb/datalayer/dbzipfs"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/model"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/repository"
)

type ZipArchive struct {
	db  *repository.BaseDB
	log *slog.Logger
}

func NewZipArchive(db *repository.BaseDB, log *slog.Logger) *ZipArchive {
	return &ZipArchive{
		db:  db,
		log: log,
	}
}

func (za *ZipArchive) Open(ctx context.Context, name string) (*model.ZipArchive, *dbzipfs.Zip, error) {
	coll := za.db.ZipArchive()
	archive, err := coll.FindByName(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	zfs := dbzipfs.NewZip(ctx, za.db, archive.ID)

	return archive, zfs, nil
}
