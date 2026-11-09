package access

import (
	"context"
	"fmt"
	"net"

	"tei/internal/core"
	"tei/internal/permission"
)

// Dialer performs the connection after the check. A test can supply a fake one,
// so the check is exercised without opening a socket.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Network connects on behalf of one capability. The address is checked against
// the granted permissions and recorded before the connection is attempted, so a
// refused connection opens no socket.
type Network struct {
	Capability core.CanonicalID
	Guard      Guard
	Dialer     Dialer
}

// DialContext checks the address and then connects.
func (n Network) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	requirement := permission.Spec{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: address}
	decision, err := n.Guard.Check(n.Capability, []permission.Spec{requirement})
	if err != nil {
		return nil, err
	}
	if !decision.Allowed {
		return nil, fmt.Errorf("capability %s may not connect to %s: the granted permissions do not cover it", n.Capability, address)
	}
	if n.Dialer == nil {
		return nil, fmt.Errorf("no dialer is configured, so %s is not connected", address)
	}
	return n.Dialer.DialContext(ctx, network, address)
}

// DialerFunc adapts a function to Dialer.
type DialerFunc func(ctx context.Context, network, address string) (net.Conn, error)

// DialContext implements Dialer.
func (f DialerFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return f(ctx, network, address)
}
