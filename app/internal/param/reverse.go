package param

type ReverseElkeid struct {
	Version string `query:"version" validate:"required,excludes=/,printascii"`
	Uname   string `query:"uname"   validate:"required,excludes=/,printascii"`
	Arch    string `query:"arch"    validate:"omitempty,oneof=amd64 arm64 arm"`
}
