package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type BenchmarkResult struct {
	Name       string
	Framework  string
	Category   string
	Scenario   string
	Iterations int64
	NsPerOp    float64
	BytesPerOp int64
	AllocsOp   int64
}

type CategoryResults struct {
	Category string
	Results  []BenchmarkResult
}

var frameworkColors = map[string]string{
	"Needle":         "\033[32m",
	"NeedleParallel": "\033[36m",
	"Do":             "\033[33m",
	"Dig":            "\033[35m",
	"Fx":             "\033[34m",
}

const reset = "\033[0m"
const bold = "\033[1m"
const dim = "\033[2m"

func main() {
	fmt.Println()
	fmt.Printf("%s%s╔══════════════════════════════════════════════════════════════════╗%s\n", bold, "\033[36m", reset)
	fmt.Printf("%s%s║         🪡  Needle DI Framework Benchmark Suite                  ║%s\n", bold, "\033[36m", reset)
	fmt.Printf("%s%s╚══════════════════════════════════════════════════════════════════╝%s\n", bold, "\033[36m", reset)
	fmt.Println()

	fmt.Printf("%sRunning benchmarks...%s\n\n", dim, reset)

	benchDir := ".."
	if len(os.Args) > 1 && os.Args[1] != "--json" {
		benchDir = os.Args[1]
	}

	cmd := exec.Command("go", "test", "-bench=.", "-benchmem", "-count=3", "-benchtime=100ms")
	cmd.Dir = benchDir
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			fmt.Fprintf(os.Stderr, "Benchmark failed: %s\n", string(exitErr.Stderr))
		}
		os.Exit(1)
	}

	results := parseResults(output)
	grouped := groupByCategory(results)

	for _, cat := range grouped {
		printCategory(cat)
	}

	printSummary(grouped)

	if len(os.Args) > 1 && os.Args[1] == "--json" {
		exportJSON(results)
	}
}

func parseResults(output []byte) []BenchmarkResult {
	var results []BenchmarkResult
	benchPattern := regexp.MustCompile(`^Benchmark(\w+)-\d+\s+(\d+)\s+([\d.]+) ns/op\s+(\d+) B/op\s+(\d+) allocs/op`)
	namePattern := regexp.MustCompile(`^([^_]+)_([^_]+)_(\w+)$`)

	seen := make(map[string][]BenchmarkResult)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		matches := benchPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		name := matches[1]
		iterations, _ := strconv.ParseInt(matches[2], 10, 64)
		nsPerOp, _ := strconv.ParseFloat(matches[3], 64)
		bytesPerOp, _ := strconv.ParseInt(matches[4], 10, 64)
		allocsOp, _ := strconv.ParseInt(matches[5], 10, 64)

		nameParts := namePattern.FindStringSubmatch(name)
		var category, scenario, framework string
		if nameParts != nil {
			category = nameParts[1]
			scenario = nameParts[2]
			framework = nameParts[3]
		} else {
			parts := strings.Split(name, "_")
			if len(parts) >= 2 {
				framework = parts[len(parts)-1]
				category = parts[0]
				scenario = strings.Join(parts[1:len(parts)-1], "_")
			}
		}

		key := name
		seen[key] = append(
			seen[key], BenchmarkResult{
				Name:       name,
				Framework:  framework,
				Category:   category,
				Scenario:   scenario,
				Iterations: iterations,
				NsPerOp:    nsPerOp,
				BytesPerOp: bytesPerOp,
				AllocsOp:   allocsOp,
			},
		)
	}

	for _, runs := range seen {
		if len(runs) == 0 {
			continue
		}

		var totalNs float64
		var totalBytes, totalAllocs int64
		for _, r := range runs {
			totalNs += r.NsPerOp
			totalBytes += r.BytesPerOp
			totalAllocs += r.AllocsOp
		}
		count := float64(len(runs))

		avg := runs[0]
		avg.NsPerOp = totalNs / count
		avg.BytesPerOp = int64(float64(totalBytes) / count)
		avg.AllocsOp = int64(float64(totalAllocs) / count)
		results = append(results, avg)
	}

	return results
}

func groupByCategory(results []BenchmarkResult) []CategoryResults {
	groups := make(map[string][]BenchmarkResult)
	for _, r := range results {
		key := r.Category + "_" + r.Scenario
		groups[key] = append(groups[key], r)
	}

	var ordered []CategoryResults
	categoryOrder := []string{
		"Provide_Simple", "Provide_Chain",
		"Invoke_Singleton", "Invoke_Chain",
		"Named_10",
		"Lifecycle_10", "Lifecycle_50",
		"LifecycleWithWork_10", "LifecycleWithWork_50",
	}

	for _, catKey := range categoryOrder {
		if results, ok := groups[catKey]; ok {
			sort.Slice(
				results, func(i, j int) bool {
					return results[i].NsPerOp < results[j].NsPerOp
				},
			)
			ordered = append(
				ordered, CategoryResults{
					Category: catKey,
					Results:  results,
				},
			)
		}
	}

	for key, results := range groups {
		found := false
		for _, o := range categoryOrder {
			if o == key {
				found = true
				break
			}
		}
		if !found {
			sort.Slice(
				results, func(i, j int) bool {
					return results[i].NsPerOp < results[j].NsPerOp
				},
			)
			ordered = append(
				ordered, CategoryResults{
					Category: key,
					Results:  results,
				},
			)
		}
	}

	return ordered
}

func printCategory(cat CategoryResults) {
	title := formatCategoryTitle(cat.Category)
	fmt.Printf("%s┌──────────────────────────────────────────────────────────────────┐%s\n", dim, reset)
	fmt.Printf("%s│ %s%-64s%s │\n", dim, bold, title, reset)
	fmt.Printf("%s├──────────────────────────────────────────────────────────────────┤%s\n", dim, reset)

	if len(cat.Results) == 0 {
		fmt.Printf("%s│ No results                                                       │%s\n", dim, reset)
		fmt.Printf("%s└──────────────────────────────────────────────────────────────────┘%s\n", dim, reset)
		fmt.Println()
		return
	}

	fastest := cat.Results[0].NsPerOp

	for i, r := range cat.Results {
		color := frameworkColors[r.Framework]
		if color == "" {
			color = reset
		}

		speedup := ""
		if i > 0 && fastest > 0 {
			ratio := r.NsPerOp / fastest
			speedup = fmt.Sprintf("(%.1fx slower)", ratio)
		} else if i == 0 {
			speedup = "(fastest)"
		}

		bar := makeBar(r.NsPerOp, fastest, 20)

		fmt.Printf(
			"%s│%s %s%-16s%s %s %s%10s %s%10d B %s%6d allocs%s │\n",
			dim, reset,
			color, r.Framework, reset,
			bar,
			dim, formatNs(r.NsPerOp), reset,
			r.BytesPerOp,
			dim, r.AllocsOp, reset,
		)

		if speedup != "" {
			fmt.Printf(
				"%s│                  %s%-40s%s              │%s\n",
				dim, dim, speedup, reset, reset,
			)
		}
	}

	fmt.Printf("%s└──────────────────────────────────────────────────────────────────┘%s\n", dim, reset)
	fmt.Println()
}

func formatCategoryTitle(cat string) string {
	parts := strings.Split(cat, "_")
	titles := map[string]string{
		"Provide_Simple":       "📦 Provider Registration (Simple)",
		"Provide_Chain":        "📦 Provider Registration (Dependency Chain)",
		"Invoke_Singleton":     "🔍 Service Resolution (Singleton)",
		"Invoke_Chain":         "🔍 Service Resolution (Dependency Chain)",
		"Named_10":             "🏷️  Named Services (10 services)",
		"Lifecycle_10":         "🔄 Lifecycle Start/Stop (10 services)",
		"Lifecycle_50":         "🔄 Lifecycle Start/Stop (50 services)",
		"LifecycleWithWork_10": "⏱️  Lifecycle with Work (10 services, 1ms each)",
		"LifecycleWithWork_50": "⏱️  Lifecycle with Work (50 services, 1ms each)",
	}

	if title, ok := titles[cat]; ok {
		return title
	}

	for i, p := range parts {
		parts[i] = strings.Title(strings.ToLower(p))
	}
	return strings.Join(parts, " ")
}

func makeBar(value, fastest float64, width int) string {
	if fastest == 0 {
		return strings.Repeat("█", width)
	}

	ratio := value / fastest
	if ratio > 10 {
		ratio = 10
	}

	filled := int(float64(width) / ratio)
	if filled < 1 {
		filled = 1
	}
	if filled > width {
		filled = width
	}

	return fmt.Sprintf(
		"\033[32m%s\033[31m%s\033[0m",
		strings.Repeat("█", filled),
		strings.Repeat("░", width-filled),
	)
}

func formatNs(ns float64) string {
	if ns >= 1_000_000 {
		return fmt.Sprintf("%.2f ms", ns/1_000_000)
	}
	if ns >= 1_000 {
		return fmt.Sprintf("%.2f µs", ns/1_000)
	}
	return fmt.Sprintf("%.0f ns", ns)
}

func printSummary(groups []CategoryResults) {
	fmt.Printf("%s%s╔══════════════════════════════════════════════════════════════════╗%s\n", bold, "\033[36m", reset)
	fmt.Printf("%s%s║                         📊 Summary                               ║%s\n", bold, "\033[36m", reset)
	fmt.Printf("%s%s╚══════════════════════════════════════════════════════════════════╝%s\n", bold, "\033[36m", reset)
	fmt.Println()

	wins := make(map[string]int)
	for _, cat := range groups {
		if len(cat.Results) > 0 {
			wins[cat.Results[0].Framework]++
		}
	}

	type frameworkWins struct {
		name string
		wins int
	}

	var sorted []frameworkWins
	for name, count := range wins {
		sorted = append(sorted, frameworkWins{name, count})
	}
	sort.Slice(
		sorted, func(i, j int) bool {
			return sorted[i].wins > sorted[j].wins
		},
	)

	total := len(groups)
	for i, fw := range sorted {
		medal := ""
		switch i {
		case 0:
			medal = "🥇"
		case 1:
			medal = "🥈"
		case 2:
			medal = "🥉"
		}

		color := frameworkColors[fw.name]
		if color == "" {
			color = reset
		}

		bar := strings.Repeat("█", fw.wins*3)
		fmt.Printf(
			"  %s %s%-16s%s %s%s%s  %d/%d benchmarks\n",
			medal, color, fw.name, reset, "\033[32m", bar, reset, fw.wins, total,
		)
	}

	fmt.Println()
	fmt.Printf("%s%sFrameworks compared:%s\n", dim, bold, reset)
	fmt.Printf(
		"  %s• Needle%s       - This library (github.com/danpasecinic/needle)\n", frameworkColors["Needle"], reset,
	)
	fmt.Printf("  %s• samber/do%s    - Generics-based DI (github.com/samber/do)\n", frameworkColors["Do"], reset)
	fmt.Printf("  %s• uber/dig%s     - Reflection-based DI (go.uber.org/dig)\n", frameworkColors["Dig"], reset)
	fmt.Printf("  %s• uber/fx%s      - Full application framework (go.uber.org/fx)\n", frameworkColors["Fx"], reset)
	fmt.Println()
}

func exportJSON(results []BenchmarkResult) {
	output := struct {
		Benchmarks []BenchmarkResult `json:"benchmarks"`
	}{
		Benchmarks: results,
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	_ = os.WriteFile("benchmark_results.json", data, 0644)
	fmt.Printf("%sResults exported to benchmark_results.json%s\n", dim, reset)
}
