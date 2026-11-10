package agentsync

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/ssoc-broker/muxtunnel/apiclient"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
)

type Client struct {
	pdb polydb.Sourcer
	cli apiclient.Client
	log *slog.Logger
}

func (c Client) PushStartup(ctx context.Context, id int64) error {
	return nil
}

func (c Client) ResyncTask(ctx context.Context, id int64) error {
	return nil
}
