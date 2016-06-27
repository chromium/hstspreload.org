package origin

import (
	"net"
	"net/url"
)

// Origin holds the explicit semantics of an origin, based on the definition
// in https://tools.ietf.org/html/rfc6454
type Origin struct {
	// HostName is called HostName instead of Host to distinguish it from
	// url.URL.Host; this is similar to the a.href.host[name] quirk in Javascript.
	HostName string
	Scheme   string
	// A port is actually an integer, so we go against normal Go convention
	// and call it "PortString" to be painfully clear.
	// TODO: Should we allow this to be empty if the port is unspecified?
	PortString string
	// TODO: Add a port int?
}

// New constructs an Origin based on the given URL. Parts of the URL that are not
// relevant to the origin are ignored.
func New(u *url.URL) (Origin, error) {
	hostName, portString, err := net.SplitHostPort(canonicalAddr(u))
	if err != nil {
		return Origin{}, err
	}

	return Origin{
		HostName:   hostName,
		Scheme:     u.Scheme,
		PortString: portString,
	}, nil
}

// Parse is a convenience function that parses a URL string and constructs an
// Origin from it.
func Parse(urlString string) (Origin, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return Origin{}, err
	}

	return New(u)
}

// // NOTE: This currently always includes the port.
// func (o Origin) String() string {
// 	u := url.URL{
// 		Scheme: o.Scheme,
// 		Host:   net.JoinHostPort(o.HostName, o.PortString),
// 	}
// 	return u.String()
// }
