package docker

import "testing"

func Test_parseAddress(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "URI", args: args{address: "http://127.0.0.1:9200"}, wantErr: false},
		{name: "missing protocol", args: args{address: "127.0.0.1:9200"}, wantErr: true},
		{name: "missing port", args: args{address: "http://127.0.0.1"}, wantErr: true},
		{name: "only ip address", args: args{address: "http://127.0.0.1"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := parseAddress(tt.args.address); (err != nil) != tt.wantErr {
				t.Errorf("parseAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
