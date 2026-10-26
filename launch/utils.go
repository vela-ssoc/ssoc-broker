package launch

import (
	"context"
	"net/http"
	"time"

	"github.com/NVIDIA/gontainer/v2"
	"github.com/vela-ssoc/ssoc-common/cronv3"
	"github.com/vela-ssoc/vela-tunnel/stegano"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"gorm.io/gorm"
)

type steganoFile string

func (s steganoFile) Load(_ context.Context, v any) error {
	return stegano.ReadManifest(string(s), v)
}

func closeMDB(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli := db.Client()

	return cli.Disconnect(ctx)
}

func closeRDB(db *gorm.DB) error {
	sdb, err := db.DB()
	if err != nil {
		return err
	}

	return sdb.Close()
}

func startHTTPS(errch chan error, srv *http.Server) {
	errch <- srv.ListenAndServeTLS("", "")
}

func closeHTTP(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return srv.Shutdown(ctx)
}

func registerCronJob(c *cronv3.Crontab, jobs gontainer.Multiple[cronv3.Tasker]) error {
	for _, job := range jobs {
		if err := c.AddTask(job); err != nil {
			return err
		}
	}

	return nil
}
