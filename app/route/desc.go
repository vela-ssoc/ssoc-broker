package route

import "time"

type Describer interface {
	Ignore(du time.Duration) bool
	Name() string
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
	if r.ignore {
		return true
	}
	if r.timeout > 0 {
		return du > r.timeout
	}
	return false
}
