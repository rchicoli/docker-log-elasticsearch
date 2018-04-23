package docker

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func Test_indexFlag(t *testing.T) {
	tests := []struct {
		name string
		args string
		want bool
	}{
		// TODO: Add test cases.
		{name: "index regex", args: "docker-F%", want: true},
		{name: "index string", args: "docker", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indexFlag(tt.args); got != tt.want {
				t.Errorf("indexFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_indexRegex(t *testing.T) {
	type args struct {
		now        time.Time
		indexRegex string
	}

	today := time.Now()
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "fullDate", args: args{now: time.Now(), indexRegex: "docker-%F"}, want: fmt.Sprintf("docker-%s", today.Format("2006.01.02"))},
		{name: "fullDateCustom", args: args{now: time.Now(), indexRegex: "docker.%Y-%m-%d"}, want: fmt.Sprintf("docker.%s", today.Format("2006-01-02"))},
		{name: "monthShort", args: args{now: time.Now(), indexRegex: "docker-%b"}, want: fmt.Sprintf("docker-%s", strings.ToLower(today.Format("Jan")))},
		{name: "monthFull", args: args{now: time.Now(), indexRegex: "docker-%B"}, want: fmt.Sprintf("docker-%s", strings.ToLower(today.Format("January")))},
		{name: "yearZeroPadded", args: args{now: time.Now(), indexRegex: "docker-%y"}, want: fmt.Sprintf("docker-%s", today.Format("06"))},
		{name: "yearZeroPadded", args: args{now: time.Now(), indexRegex: "docker-%Y"}, want: fmt.Sprintf("docker-%s", today.Format("2006"))},
		{name: "yearZeroPadded", args: args{now: time.Now(), indexRegex: "docker-%m"}, want: fmt.Sprintf("docker-%s", today.Format("01"))},
		{name: "yearZeroPadded", args: args{now: time.Now(), indexRegex: "docker-%d"}, want: fmt.Sprintf("docker-%s", today.Format("02"))},
		{name: "dayOfYearZeroPadded", args: args{now: time.Now(), indexRegex: "docker-%j"}, want: fmt.Sprintf("docker-%d", today.YearDay())},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indexRegex(tt.args.now, tt.args.indexRegex); got != tt.want {
				t.Errorf("indexRegex() = %v, want %v", got, tt.want)
			}
		})
	}
}
