package spec

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

type Acceptance struct {
	Reference   string
	Ambiguities int
}

type Verification struct {
	Verdict string
	OK      []string
	Partial int
	Missing int
}

var (
	referenceRe = regexp.MustCompile(`(?i)^(?:reference|référence)\s*:\s*([0-9a-f]{7,40})\b`)
	ambigHeadRe = regexp.MustCompile(`(?i)^(#{1,6})\s+.*ambigu`)
	anyHeadRe   = regexp.MustCompile(`^(#{1,6})\s`)
	listItemRe  = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+(.*\S)\s*$`)
	checkedRe   = regexp.MustCompile(`^\[[xX]\]`)
	resolvedRe  = regexp.MustCompile(`(?i)~~|r[eé]solu|resolved`)
	nothingRe   = regexp.MustCompile(`(?i)^(none|n/a|aucune?|-)\.?$`)
	verdictRe   = regexp.MustCompile(`(?i)verdict[^|]*?\b(PASS|FAIL)\b`)
)

func lines(r io.Reader) []string {
	var out []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		out = append(out, sc.Text())
	}
	return out
}

func ParseAcceptance(r io.Reader) Acceptance {
	var a Acceptance
	level := 0
	for _, line := range lines(r) {
		if a.Reference == "" {
			clean := strings.TrimLeft(strings.ReplaceAll(strings.ReplaceAll(line, "*", ""), "_", ""), " \t>-")
			if m := referenceRe.FindStringSubmatch(clean); m != nil {
				a.Reference = m[1]
				continue
			}
		}
		if h := anyHeadRe.FindStringSubmatch(line); h != nil {
			if m := ambigHeadRe.FindStringSubmatch(line); m != nil {
				level = len(m[1])
			} else if level > 0 && len(h[1]) <= level {
				level = 0
			}
			continue
		}
		if level == 0 {
			continue
		}
		m := listItemRe.FindStringSubmatch(line)
		if m == nil || checkedRe.MatchString(m[1]) || resolvedRe.MatchString(m[1]) || nothingRe.MatchString(m[1]) {
			continue
		}
		a.Ambiguities++
	}
	return a
}

func ParseVerification(r io.Reader) Verification {
	var v Verification
	for _, line := range lines(r) {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			if m := verdictRe.FindStringSubmatch(line); m != nil && v.Verdict == "" {
				v.Verdict = strings.ToUpper(m[1])
			}
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		for _, c := range cells[1:] {
			switch strings.ToUpper(strings.Trim(strings.TrimSpace(c), "*`")) {
			case "OK":
				v.OK = append(v.OK, strings.TrimSpace(cells[0]))
			case "PARTIAL":
				v.Partial++
			case "MISSING":
				v.Missing++
			default:
				continue
			}
			break
		}
	}
	return v
}
