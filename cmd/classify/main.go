// Command classify prints the privacy badge for one or more URLs using the
// go.privatebychoice.com/pbc-classification library. It is a small, known-good
// reference for wiring the module into your own code.
//
// Usage:
//
//	classify https://youtube.com https://example.com
//	classify -json https://youtube.com
//	classify -first-party privatebychoice.com,theuntrackedlife.com https://privatebychoice.com
//	cat urls.txt | classify
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	classify "go.privatebychoice.com/pbc-classification"
)

func main() {
	dataFile := flag.String("data", "", "path to an additional dataset JSON file (overrides the built-in seed)")
	firstParty := flag.String("first-party", "", "comma-separated domains to treat as first-party (grade A)")
	asJSON := flag.Bool("json", false, "emit JSON (one object per line) instead of text")
	flag.Usage = usage
	flag.Parse()

	c, err := newClassifier(*dataFile, *firstParty)
	if err != nil {
		fmt.Fprintln(os.Stderr, "classify:", err)
		os.Exit(1)
	}

	urls := flag.Args()
	if len(urls) == 0 {
		urls = readStdin()
	}
	if len(urls) == 0 {
		usage()
		os.Exit(2)
	}

	enc := json.NewEncoder(os.Stdout)
	for _, u := range urls {
		r := c.Classify(u)
		if *asJSON {
			if err := enc.Encode(toResult(r)); err != nil {
				fmt.Fprintln(os.Stderr, "classify:", err)
				os.Exit(1)
			}
			continue
		}
		printText(r)
	}
}

func newClassifier(dataFile, firstParty string) (*classify.Classifier, error) {
	var opts []classify.Option
	if dataFile != "" {
		opts = append(opts, classify.WithDataFile(dataFile))
	}
	if fp := splitCSV(firstParty); len(fp) > 0 {
		opts = append(opts, classify.WithFirstParty(fp...))
	}
	return classify.New(opts...)
}

// result is the machine-readable shape — a preview of what an SSG per-page
// privacy manifest entry could look like.
type result struct {
	URL       string   `json:"url"`
	Domain    string   `json:"domain"`
	Matched   bool     `json:"matched"`
	Grade     string   `json:"grade"`     // letter: A–F or "?"
	GradeName string   `json:"gradeName"` // Clean/Considerate/Mixed/Tracking/Invasive/Unclassified
	Trust     string   `json:"trust"`
	Verified  string   `json:"verified,omitempty"`
	Stale     bool     `json:"stale"`
	Reasons   []string `json:"reasons"`
}

func toResult(r classify.Classification) result {
	return result{
		URL:       r.Input,
		Domain:    r.Domain,
		Matched:   r.Matched,
		Grade:     r.Grade.Letter(),
		GradeName: r.Grade.Name(),
		Trust:     r.Trust.String(),
		Verified:  r.Verified,
		Stale:     r.Stale,
		Reasons:   r.Reasons,
	}
}

func printText(r classify.Classification) {
	marker := r.Trust.Marker()
	if marker != "" {
		marker += " "
	}
	fmt.Printf("%s\n  %s%s %s\n", r.Input, marker, r.Grade.Icon(), r.Grade)
	if r.Domain != "" {
		fmt.Printf("  domain: %s\n", r.Domain)
	}
	if r.Trust != classify.TrustUnknown {
		line := "  source: " + r.Trust.String()
		if r.Verified != "" {
			line += " (verified " + r.Verified + ")"
		}
		if r.Stale {
			line += " [STALE]"
		}
		fmt.Println(line)
	}
	for _, reason := range r.Reasons {
		fmt.Printf("  - %s\n", reason)
	}
	if r.Note != "" {
		fmt.Printf("  note: %s\n", r.Note)
	}
	fmt.Println()
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// readStdin reads newline-delimited URLs from a pipe, if one is present.
func readStdin() []string {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice != 0 {
		return nil // interactive terminal, no pipe
	}
	var out []string
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			out = append(out, line)
		}
	}
	return out
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: classify [flags] <url> [url ...]")
	fmt.Fprintln(os.Stderr, "       cat urls.txt | classify [flags]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Flags:")
	flag.PrintDefaults()
}
