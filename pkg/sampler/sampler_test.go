package sampler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkSample(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Howdy")
	}))
	defer ts.Close()

	s := New()

	for n := 0; n < b.N; n++ {
		s.Sample(ts.URL)
	}
}
