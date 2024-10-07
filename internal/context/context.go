package context

import (
	"github.com/comp590/ocss/pkg/factory"
)

var ocssContext OCSSContext

// Contains all the hardware information
type OCSSContext struct {
}

func Init(cfg *factory.Configuration) {

}

func GetSelf() *OCSSContext {
	return &ocssContext
}
