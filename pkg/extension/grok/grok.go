package grok

import (
	"fmt"
	"strings"

	"github.com/vjeantet/grok"
)

// Grok ...
type Grok struct {
	*grok.Grok
}

// NewGrok ...
func NewGrok(grokMatch, grokPattern, grokPatternFrom, grokPatternSplitter string, grokNamedCapture bool) (*Grok, error) {
	if grokMatch == "" {
		return &Grok{}, nil
	}

	groker, _ := grok.NewWithConfig(&grok.Config{NamedCapturesOnly: grokNamedCapture})
	g := &Grok{groker}

	if grokPattern != "" {
		var patternNames []string
		grokPatterns := strings.Split(grokPattern, grokPatternSplitter)
		for _, v := range grokPatterns {
			patternNames = strings.Split(v, "=")
			if len(patternNames) != 2 {
				return g, fmt.Errorf("error: parsing grok-pattern, missing '=' separator")
			}
			err := g.AddPattern(patternNames[0], patternNames[1])
			if err != nil {
				return g, fmt.Errorf("error: adding grok pattern: %v", err)
			}
		}
	}

	if grokPatternFrom != "" {
		err := g.AddPatternsFromPath(grokPatternFrom)
		if err != nil {
			return g, fmt.Errorf("error: adding grok pattern from %s: %v", grokPatternFrom, err)
		}
	}

	return g, nil
}

// ParseLine ...
func (g *Grok) ParseLine(pattern, logMessage string, line []byte) (map[string]string, []byte, error) {

	if g.Grok == nil {
		return nil, line, nil
	}

	// TODO: create a PR to grok upstream for returning a regexp
	// doing so we avoid to compile the regexp twice
	grokMatch, err := g.Match(pattern, logMessage)
	if err != nil {
		return map[string]string{"line": logMessage, "err": err.Error()}, nil, err
	}
	if !grokMatch {
		// do not try parse this line, because it will return an empty map
		return map[string]string{"line": logMessage, "err": "grok pattern does not match log line"},
			nil,
			fmt.Errorf("error: grok pattern does not match line: %s", logMessage)
	}

	grokLine, err := g.Parse(pattern, logMessage)
	if err != nil {
		return map[string]string{"line": logMessage, "err": err.Error()}, nil, err
	}

	return grokLine, nil, nil

}
