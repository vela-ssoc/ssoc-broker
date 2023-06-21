package route

import "time"

type Describer interface {
	Ignore(du time.Duration) bool
	Name() string
}

func Slow(name string, du time.Duration) Describer {
	return routeDesc{name: name}
}

func Named(name string) Describer {
	return routeDesc{name: name}
}

func Ignore() Describer {
	return routeDesc{ignore: true}
}

type routeDesc struct {
	ignore  bool
	name    string
	timeout time.Duration
}

func (r routeDesc) Name() string { return r.name }
func (r routeDesc) Ignore(du time.Duration) bool {
	return !r.ignore && r.timeout > 0 && du >= r.timeout
}
