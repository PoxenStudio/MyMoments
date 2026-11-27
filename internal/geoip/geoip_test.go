package geoip

import "testing"

func TestParseIPIP(t *testing.T) {
	cases := []struct {
		name string
		body string
		want Result
		ok   bool
	}{
		{
			name: "省市运营商",
			body: "当前 IP：223.104.3.18  来自于：中国 广东 深圳  电信\n",
			want: Result{IP: "223.104.3.18", Country: "中国", Province: "广东", City: "深圳", Source: sourceIPIP},
			ok:   true,
		},
		{
			name: "直辖市带市后缀",
			body: "当前 IP：1.2.3.4 来自于：中国 北京 北京市  联通\n",
			want: Result{IP: "1.2.3.4", Country: "中国", Province: "北京", City: "北京", Source: sourceIPIP},
			ok:   true,
		},
		{
			name: "仅国家与城市",
			body: "当前 IP：8.8.8.8  来自于：美国 洛杉矶\n",
			want: Result{IP: "8.8.8.8", Country: "美国", City: "洛杉矶", Source: sourceIPIP},
			ok:   true,
		},
		{
			name: "内网地址",
			body: "当前 IP：192.168.1.10  来自于：局域网\n",
			ok:   false,
		},
		{
			name: "无法识别",
			body: "unknown",
			ok:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseIPIP(tc.body)
			if !tc.ok {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseIPWho(t *testing.T) {
	body := []byte(`{"success":true,"ip":"1.2.3.4","country":"China","region":"Guangdong","city":"Shenzhen","latitude":22.54,"longitude":114.06}`)
	got, err := parseIPWho(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.City != "Shenzhen" || got.Lat == nil || *got.Lat != 22.54 || got.Lon == nil || *got.Lon != 114.06 {
		t.Fatalf("unexpected result: %+v", got)
	}

	if _, err := parseIPWho([]byte(`{"success":false,"message":"reserved range"}`)); err == nil {
		t.Fatal("expected error for failed lookup")
	}
}
