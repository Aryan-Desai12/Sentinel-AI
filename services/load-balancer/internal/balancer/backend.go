package balancer

import "net/url"

type Backend struct {
	URL *url.URL
}
