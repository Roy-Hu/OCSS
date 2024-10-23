package ocss

import "github.com/comp590/ocss/pkg/app"

type RdcOCSS interface {
	app.App
}

type Rdc struct {
	RdcOCSS
}

func NewRdc(ocss RdcOCSS) (*Rdc, error) {
	r := &Rdc{
		RdcOCSS: ocss,
	}

	return r, nil
}

func (r *Rdc) GetPorts(switchName string) []int {
	return []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
}
