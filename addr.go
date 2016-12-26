package netx

// NetAddr is a type satisifying the net.Addr interface.
type NetAddr struct {
	Net  string
	Addr string
}

// Netowrk returns a.Net
func (a *NetAddr) Network() string { return a.Net }

// String returns a.Addr
func (a *NetAddr) String() string { return a.Addr }
