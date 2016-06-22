package origin

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {

	cases := []struct {
		url    string
		wanted Origin
	}{
		{"http://localhost:8080", Origin{HostName: "localhost", Scheme: "http", PortString: "8080"}},
		{"http://example.com", Origin{HostName: "example.com", Scheme: "http", PortString: "80"}},
		{"http://example.com/", Origin{HostName: "example.com", Scheme: "http", PortString: "80"}},
		{"http://example.com:80", Origin{HostName: "example.com", Scheme: "http", PortString: "80"}},
		{"http://example.com/path", Origin{HostName: "example.com", Scheme: "http", PortString: "80"}},
		{"https://example.com/#yolo", Origin{HostName: "example.com", Scheme: "https", PortString: "443"}},
		{"https://example.com:443", Origin{HostName: "example.com", Scheme: "https", PortString: "443"}},
		{"https://a.b.example.com", Origin{HostName: "a.b.example.com", Scheme: "https", PortString: "443"}},
		{"https://alice@a.b.example.com:9001/path", Origin{HostName: "a.b.example.com", Scheme: "https", PortString: "9001"}},
	}

	for _, tt := range cases {
		o, err := Parse(tt.url)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(o, tt.wanted) {
			t.Errorf("Unexpected: %#v", o)
		}
	}

}
