package docker

import (
	"fmt"
	"regexp"
	"time"

	"github.com/Sirupsen/logrus"
)

// %. matches each strftime format sequence and ReplaceAllStringFunc
// looks up each format sequence in the conversion table strftimeToRegex
// to replace with a defined regular expression
var percent = regexp.MustCompile("%.")

func indexRegex(now time.Time, indexRegex string) string {

	// %b     locale's abbreviated month name (Jan)
	// %B     locale's full month name (January)
	// %d     day of month (01)
	// %F     full date; same as %Y.%m.%d
	// %j     day of year (001..366)
	// %m     month (01..12)
	// %y     last two digits of year (00..99)
	// %Y     year (2018)
	var strftimeToRegex = map[string]string{
		/*dayZeroPadded         */ `%d`: fmt.Sprintf("%02d", now.Day()),
		/*monthShort            */ `%b`: now.Month().String()[:3],
		/*monthFull             */ `%B`: now.Month().String(),
		/*monthFull             */ `%F`: fmt.Sprintf("%d.%02d.%02d", now.Year(), int(now.Month()), now.Day()),
		/*monthZeroPadded       */ `%m`: fmt.Sprintf("%02d", int(now.Month())),
		/*yearCentury           */ `%Y`: fmt.Sprintf("%d", now.Year()),
		/*yearZeroPadded        */ `%y`: fmt.Sprintf("%d", now.Year())[2:],
		/*dayOfYearZeroPadded   */ `%j`: fmt.Sprintf("%d", now.YearDay()),
		/*testSecond            */ `%z`: fmt.Sprintf("%02d", now.Second()),
	}

	var indexName = percent.ReplaceAllStringFunc(indexRegex, func(s string) string {
		return strftimeToRegex[s]
	})
	logrus.WithField("indexname", indexName).Debug("index name generated from regex")

	return indexName

}
