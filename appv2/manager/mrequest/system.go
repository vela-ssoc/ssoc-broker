package mrequest

import "github.com/vela-ssoc/vela-common-mb/dal/model"

type SystemUpdate struct {
	Semver model.Semver `json:"semver" query:"semver"`
}
