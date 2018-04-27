package regex

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var percent = regexp.MustCompile("%.")

// IsValid check if regex contains a percent sign
func IsValid(str string) bool {
	if strings.Contains(str, "%") {
		return true
	}
	return false
}

// ParseDate matches each strftime format sequence and ReplaceAllStringFunc
// looks up each format sequence in the conversion table regexToStrftime
// to replace with a defined time format
func ParseDate(now time.Time, regex string) string {

	// %b     locale's abbreviated month name (Jan)
	// %B     locale's full month name (January)
	// %d     day of month (01)
	// %F     full date; same as %Y.%m.%d
	// %j     day of year (001..366)
	// %m     month (01..12)
	// %y     last two digits of year (00..99)
	// %Y     year (2018)
	var regexToStrftime = map[string]string{
		/*dayZeroPadded         */ `%d`: now.Format("02"),
		/*monthShort            */ `%b`: now.Format("Jan"),
		/*monthFull             */ `%B`: now.Format("January"),
		/*monthFull             */ `%F`: now.Format("2006.01.02"),
		/*monthZeroPadded       */ `%m`: now.Format("01"),
		/*yearCentury           */ `%Y`: now.Format("2006"),
		/*yearZeroPadded        */ `%y`: now.Format("06"),
		/*dayOfYearZeroPadded   */ `%j`: fmt.Sprintf("%d", now.YearDay()),
	}

	return percent.ReplaceAllStringFunc(regex, func(s string) string {
		return regexToStrftime[s]
	})

}
