package httpapi

import (
	"testing"

	"mymoments/internal/config"
)

func TestOriginAllowed(t *testing.T) {
	cases := []struct {
		name   string
		dev    bool
		origin string
		host   string
		want   bool
	}{
		{name: "同源", origin: "https://mm.example.com", host: "mm.example.com", want: true},
		{name: "跨域拒绝", origin: "https://evil.example.com", host: "mm.example.com", want: false},
		{name: "开发模式本机跨端口", dev: true, origin: "http://localhost:5173", host: "127.0.0.1:8080", want: true},
		{name: "生产模式本机跨端口拒绝", origin: "http://localhost:5173", host: "127.0.0.1:8080", want: false},
		{name: "开发模式外部来源仍拒绝", dev: true, origin: "https://evil.example.com", host: "127.0.0.1:8080", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{cfg: config.Config{Dev: tc.dev}}
			if got := s.originAllowed(tc.origin, tc.host); got != tc.want {
				t.Fatalf("originAllowed(%q, %q) = %v, want %v", tc.origin, tc.host, got, tc.want)
			}
		})
	}
}
