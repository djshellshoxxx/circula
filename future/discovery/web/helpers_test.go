package web_test

import (
	"net/http/httptest"

	d "github.com/djshellshoxxx/circula/future/discovery"
)

type recorderT = httptest.ResponseRecorder

func dInput(cat string, tags ...string) d.DiscoveryInput {
	in := d.DiscoveryInput{Tags: tags}
	if cat != "" {
		in.CategoryID, in.Type = cat, d.TypeGoods
	}
	return in
}
