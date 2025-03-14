package mrequest

import "github.com/vela-ssoc/ssoc-common-mb/dal/model"

type SystemUpdate struct {
	Semver model.Semver `json:"semver" query:"semver"`
}
