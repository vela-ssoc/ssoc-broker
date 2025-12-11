package current

import (
	"context"
	"crypto/tls"
	"log/slog"
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
)

func NewBroker(secret string, qry *query.Query, log *slog.Logger) *Broker {
	return &Broker{
		secret: secret,
		qry:    qry,
		log:    log,
	}
}

type Broker struct {
	secret string
	qry    *query.Query
	log    *slog.Logger
}

func (brok *Broker) Load(ctx context.Context) (*model.Broker, error) {
	tbl := brok.qry.Broker
	dao := tbl.WithContext(ctx)
	this, err := dao.Where(tbl.Secret.Eq(brok.secret)).First()
	if err != nil {
		return nil, err
	}

	return this, nil
}

func (brok *Broker) ResetAgents(parent context.Context) error {
	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithTimeout(parent, time.Minute)
	defer cancel()

	this, err := brok.Load(ctx)
	if err != nil {
		brok.log.Error("查询当前 broker 信息错误", "error", err)
		return err
	}

	tbl := brok.qry.Minion
	dao := tbl.WithContext(ctx)
	_, err = dao.Where(tbl.Status.Eq(uint8(model.MSOnline)), tbl.BrokerID.Eq(this.ID)).
		UpdateColumn(tbl.Status, model.MSOffline)

	return err
}

func (brok *Broker) Certificates(ctx context.Context) ([]*tls.Certificate, error) {
	this, err := brok.Load(ctx)
	if err != nil {
		brok.log.Error("查询当前 broker 信息错误", "error", err)
		return nil, err
	}

	certID := this.CertID
	if certID == 0 {
		brok.log.Error("当前 broker 没有挂载 TLS 证书")
		return nil, nil
	}

	tbl := brok.qry.Certificate
	dao := tbl.WithContext(ctx)
	cert, err := dao.Where(tbl.ID.Eq(certID)).First()
	if err != nil {
		brok.log.Error("从数据库查询证书出错", "error", err, "cert_id", certID)
		return nil, err
	}
	pair, err := tls.X509KeyPair([]byte(cert.Certificate), []byte(cert.PrivateKey))
	if err != nil {
		brok.log.Error("parse 证书出错", "error", err, "cert_id", certID, "cert_name", cert.Name)
		return nil, err
	}
	certs := []*tls.Certificate{&pair}

	return certs, nil
}
