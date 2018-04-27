package regex

import (
	"fmt"
	"testing"
	"time"
)

func Test_IsValid(t *testing.T) {
	tests := []struct {
		name string
		args string
		want bool
	}{
		{name: "index regex", args: "docker-F%", want: true},
		{name: "index string", args: "docker", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValid(tt.args); got != tt.want {
				t.Errorf("indexFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ParseDate(t *testing.T) {
	type args struct {
		now     time.Time
		pattern string
	}

	today := time.Now()
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "fullDate", args: args{now: time.Now(), pattern: "docker-%F"}, want: fmt.Sprintf("docker-%s", today.Format("2006.01.02"))},
		{name: "fullDateCustom", args: args{now: time.Now(), pattern: "docker.%Y-%m-%d"}, want: fmt.Sprintf("docker.%s", today.Format("2006-01-02"))},
		{name: "monthShort", args: args{now: time.Now(), pattern: "docker-%b"}, want: fmt.Sprintf("docker-%s", today.Format("Jan"))},
		{name: "monthFull", args: args{now: time.Now(), pattern: "docker-%B"}, want: fmt.Sprintf("docker-%s", today.Format("January"))},
		{name: "yearZeroPadded", args: args{now: time.Now(), pattern: "docker-%y"}, want: fmt.Sprintf("docker-%s", today.Format("06"))},
		{name: "yearZeroPadded", args: args{now: time.Now(), pattern: "docker-%Y"}, want: fmt.Sprintf("docker-%s", today.Format("2006"))},
		{name: "yearZeroPadded", args: args{now: time.Now(), pattern: "docker-%m"}, want: fmt.Sprintf("docker-%s", today.Format("01"))},
		{name: "yearZeroPadded", args: args{now: time.Now(), pattern: "docker-%d"}, want: fmt.Sprintf("docker-%s", today.Format("02"))},
		{name: "dayOfYearZeroPadded", args: args{now: time.Now(), pattern: "docker-%j"}, want: fmt.Sprintf("docker-%d", today.YearDay())},
		{name: "zeroRegex", args: args{now: time.Now(), pattern: "docker"}, want: "docker"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseDate(tt.args.now, tt.args.pattern); got != tt.want {
				t.Errorf("regex() = %v, want %v", got, tt.want)
			}
		})
	}
}
