package serverd

type option struct {
	valid func(any) error
}
