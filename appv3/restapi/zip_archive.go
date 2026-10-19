package restapi

import (
	"mime"
	"net/http"
	"strconv"

	"github.com/vela-ssoc/ssoc-broker/appv3/request"
	"github.com/vela-ssoc/ssoc-broker/appv3/service"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-common-mb/yekazip"
	"github.com/xgfone/ship/v5"
	"github.com/yeka/zip"
)

type ZipArchive struct {
	svc *service.ZipArchive
}

func NewZipArchive(svc *service.ZipArchive) *ZipArchive {
	return &ZipArchive{svc: svc}
}

func (za *ZipArchive) RegisterHTTP(g shipx.RouteGroup) error {
	g.AgentV3.Route("/zip-archive/download").GET(za.download)
	return nil
}

// download agent 下载虚拟文件系统，响应规范如下：
//   - 文件不存在响应 404
//   - 文件找到且 version 匹配，响应 304 和密码 Header。
//   - 文件找到但 version 不匹配，响应 2xx 文件流与密码 Header。
func (za *ZipArchive) download(c *ship.Context) error {
	var param request.ZipArchiveDownload
	if err := c.BindQuery(&param); err != nil {
		return err
	}

	ctx := c.Request().Context()
	archive, zfs, err := za.svc.Open(ctx, param.Name)
	if err != nil {
		return ship.ErrNotFound
	}

	params := map[string]string{
		"filename": archive.Name + ".zip",
		"version":  strconv.FormatInt(archive.Version, 10), // 最新版本
		"method":   strconv.Itoa(archive.Method),
		"password": archive.Password,
	}
	mediaType := mime.FormatMediaType("attachment", params)
	c.SetRespHeader(ship.HeaderContentDisposition, mediaType)
	if archive.Version == param.Version {
		return c.NoContent(http.StatusNotModified)
	}
	c.SetRespHeader(ship.HeaderContentType, "application/zip")

	method := zip.EncryptionMethod(archive.Method)
	zw := zip.NewWriter(c)
	defer zw.Close()

	return yekazip.AddFS(zw, zfs, method, archive.Password)
}
