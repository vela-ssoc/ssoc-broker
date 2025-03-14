package bservice

import (
	"log/slog"

	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
)

func NewDeploy(qry *query.Query) *Deploy {
	return &Deploy{
		qry: qry,
	}
}

type Deploy struct {
	qry *query.Query
	log *slog.Logger
}
