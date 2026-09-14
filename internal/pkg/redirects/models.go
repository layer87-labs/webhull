package redirects

// Rule is a validated, normalized redirect ready for route registration.
type Rule struct {
	// From is the absolute request path without a trailing slash,
	// e.g. "/souveraenitaet". The trailing-slash spelling is served too.
	From string

	// To is the target: an internal path (fragment allowed) or an absolute
	// http(s) URL.
	To string

	// Status is the HTTP status code sent with the Location header.
	Status int
}
