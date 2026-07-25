package auth_test

import (
	"net/http"
	"net/url"
	"testing"

	"gitlab.com/iglou.eu/goulc/hided"
	"gitlab.com/iglou.eu/goulc/http/client/auth"
)

func BenchmarkBasic_Header(b *testing.B) {
	basic, err := auth.NewBasic("Mufasa", hided.NewString("Circle of Life"))
	if err != nil {
		b.Fatalf("NewBasic() error = %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = basic.Header(http.MethodGet, nil, nil)
	}
}

func BenchmarkDigest_Header(b *testing.B) {
	digest := &auth.Digest{
		Username: "Mufasa",
		Password: hided.NewString("Circle of Life"),
		Parameters: auth.DigestParameters{
			Algorithm: auth.DigestSHA256,
			Realm:     "http-auth@example.org",
			URI:       "/dir/index.html",
			QOP:       "auth",
			Nonce:     "7ypf/xlj9XXwfDPEoM4URrv/xwf94BcCAzFZH4GiTo0v",
			CNonce:    "f2/wE4q74E6zIJEtWaHKaf5wv/H5QzzpXusqGemxURZJ",
			NC:        "00000001",
			Opaque:    "FQhe/qaU925kfnzjCev0ciny7QMkPqMAFRtzCUYo5tdS",
		},
	}
	reqURL := &url.URL{Path: "/dir/index.html"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = digest.Header(http.MethodGet, reqURL, nil)
	}
}
