package exception

import (
	"fmt"
	"strings"
)

type NoSessionException struct {
	SessionAddr string
}

func (n *NoSessionException) Addr() string {
	return strings.Split(n.SessionAddr, ":")[0]
}

func (n *NoSessionException) Error() string {
	return fmt.Sprintf("No Session for %s", n.SessionAddr)
}

type NoValidSessions struct{}

func (n *NoValidSessions) Error() string {
	return fmt.Sprintf("No valid sessions")
}
