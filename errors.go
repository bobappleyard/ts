package ts

import (
	"errors"
	"fmt"
)

var (
	Undefined = errors.New("undefined")
)

func ArgError(c int) error {
	return fmt.Errorf("wrong number of arguments %d", c)
}
