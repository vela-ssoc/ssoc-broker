package service

import (
	"context"
	"crypto/tls"

	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Certificate struct {
	pdb polydb.Sourcer
}

func NewCertificate(pdb polydb.Sourcer) *Certificate {
	return &Certificate{pdb: pdb}
}

func (crt *Certificate) LoadCertificate(ctx context.Context) ([]*tls.Certificate, error) {
	coll := crt.pdb.MDB().Certificate()
	return coll.Certificates(ctx, bson.D{})
}
