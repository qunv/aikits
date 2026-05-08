package lang

import "fmt"

// ErrToolNotFound is returned when a required external tool (e.g. gopls, jdtls) is not in PATH.
type ErrToolNotFound string

func (e ErrToolNotFound) Error() string {
	return fmt.Sprintf("required tool %q not found in PATH", string(e))
}
