package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

type BenchmarkResult struct {
	Name       string  `json:"name"`
	Framework  string  `json:"framework"`
	Category   string  `json:"category"`
	Scenario   string  `json:"scenario"`
	Iterations int64   `json:"iterations"`
	NsPerOp    float64 `json:"ns_per_op"`
	BytesPerOp int64   `json:"bytes_per_op"`
	AllocsOp   int64   `json:"allocs_per_op"`
}

type CategoryResults struct {
	Category string
	Results  []BenchmarkResult
}

var frameworkColors = map[string]text.Color{
	"Needle":         text.FgGreen,
	"NeedleParallel": text.FgCyan,
	"Do":             text.FgYellow,
	"Dig":            text.FgMagenta,
	"Fx":             text.FgBlue,
}

func main() {
	markdown := flag.Bool("md", false, "Output in markdown format")
	jsonOut := flag.Bool("json", false, "Export results to JSON file")
	flag.Parse()

	fmt.Println()
	printHeader(*markdown)

	benchDir := "."
	args := flag.Args()
	if len(args) > 0 {
		benchDir = args[0]
	}

	if !*markdown {
		fmt.Printf("\033[2mRunning benchmarks...\033[0m\n\n")
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
		printCategory(cat, *markdown)
	}

	printSummary(grouped, *markdown)

	if *jsonOut {
		exportJSON(results)
	}
}

func printHeader(markdown bool) {
	if markdown {
		fmt.Println("# Needle DI Framework Benchmark Results")
		fmt.Println()
		return
	}

	fmt.Println("\033[1m\033[36m╔══════════════════════════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[1m\033[36m║         🪡  Needle DI Framework Benchmark Suite                  ║\033[0m")
	fmt.Println("\033[1m\033[36m╚══════════════════════════════════════════════════════════════════╝\033[0m")
	fmt.Println()
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

func printCategory(cat CategoryResults, markdown bool) {
	if len(cat.Results) == 0 {
		return
	}

	title := formatCategoryTitle(cat.Category)
	fastest := cat.Results[0].NsPerOp

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	if markdown {
		t.AppendHeader(table.Row{"Framework", "Time", "Memory", "Allocs", "Comparison"})
	} else {
		t.SetTitle(title)
		t.AppendHeader(table.Row{"Framework", "Time", "Memory", "Allocs", "vs Fastest"})
	}

	for i, r := range cat.Results {
		comparison := ""
		if i == 0 {
			comparison = "fastest"
		} else if fastest > 0 {
			ratio := r.NsPerOp / fastest
			comparison = fmt.Sprintf("%.1fx slower", ratio)
		}

		row := table.Row{
			r.Framework,
			formatNs(r.NsPerOp),
			fmt.Sprintf("%d B", r.BytesPerOp),
			fmt.Sprintf("%d", r.AllocsOp),
			comparison,
		}
		t.AppendRow(row)
	}

	if markdown {
		fmt.Printf("### %s\n\n", title)
		fmt.Println(t.RenderMarkdown())
		fmt.Println()
	} else {
		t.SetStyle(table.StyleRounded)
		t.Style().Title.Align = text.AlignCenter
		t.Style().Options.SeparateRows = false

		for i, r := range cat.Results {
			if color, ok := frameworkColors[r.Framework]; ok {
				t.SetColumnConfigs(
					[]table.ColumnConfig{
						{Number: 1, Colors: text.Colors{color}},
					},
				)
				_ = i
			}
		}

		t.Render()
		fmt.Println()
	}
}

func formatCategoryTitle(cat string) string {
	titles := map[string]string{
		"Provide_Simple":       "Provider Registration (Simple)",
		"Provide_Chain":        "Provider Registration (Dependency Chain)",
		"Invoke_Singleton":     "Service Resolution (Singleton)",
		"Invoke_Chain":         "Service Resolution (Dependency Chain)",
		"Named_10":             "Named Services (10 services)",
		"Lifecycle_10":         "Lifecycle Start/Stop (10 services)",
		"Lifecycle_50":         "Lifecycle Start/Stop (50 services)",
		"LifecycleWithWork_10": "Lifecycle with Work (10 services, 1ms each)",
		"LifecycleWithWork_50": "Lifecycle with Work (50 services, 1ms each)",
	}

	if title, ok := titles[cat]; ok {
		return title
	}

	parts := strings.Split(cat, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, " ")
}

func formatNs(ns float64) string {
	if ns >= 1_000_000 {
		return fmt.Sprintf("%.2f ms", ns/1_000_000)
	}
	if ns >= 1_000 {
		return fmt.Sprintf("%.2f us", ns/1_000)
	}
	return fmt.Sprintf("%.0f ns", ns)
}

func printSummary(groups []CategoryResults, markdown bool) {
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

	if markdown {
		fmt.Println("## Summary")
		fmt.Println()
		fmt.Println("| Rank | Framework | Wins |")
		fmt.Println("|------|-----------|------|")
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
			fmt.Printf("| %s | %s | %d/%d |\n", medal, fw.name, fw.wins, total)
		}
		fmt.Println()
		fmt.Println("**Frameworks compared:**")
		fmt.Println("- **Needle** - This library (github.com/danpasecinic/needle)")
		fmt.Println("- **samber/do** - Generics-based DI (github.com/samber/do)")
		fmt.Println("- **uber/dig** - Reflection-based DI (go.uber.org/dig)")
		fmt.Println("- **uber/fx** - Full application framework (go.uber.org/fx)")
		fmt.Println()
	} else {
		fmt.Println("\033[1m\033[36m╔══════════════════════════════════════════════════════════════════╗\033[0m")
		fmt.Println("\033[1m\033[36m║                           Summary                                ║\033[0m")
		fmt.Println("\033[1m\033[36m╚══════════════════════════════════════════════════════════════════╝\033[0m")
		fmt.Println()

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Rank", "Framework", "Wins"})

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
			t.AppendRow(table.Row{medal, fw.name, fmt.Sprintf("%d/%d", fw.wins, total)})
		}

		t.SetStyle(table.StyleRounded)
		t.Render()
		fmt.Println()

		fmt.Println("\033[2m\033[1mFrameworks compared:\033[0m")
		fmt.Println("  \033[32m• Needle\033[0m       - This library (github.com/danpasecinic/needle)")
		fmt.Println("  \033[33m• samber/do\033[0m    - Generics-based DI (github.com/samber/do)")
		fmt.Println("  \033[35m• uber/dig\033[0m     - Reflection-based DI (go.uber.org/dig)")
		fmt.Println("  \033[34m• uber/fx\033[0m      - Full application framework (go.uber.org/fx)")
		fmt.Println()
	}
}

func exportJSON(results []BenchmarkResult) {
	output := struct {
		Benchmarks []BenchmarkResult `json:"benchmarks"`
	}{
		Benchmarks: results,
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	_ = os.WriteFile("benchmark_results.json", data, 0644)
	fmt.Println("\033[2mResults exported to benchmark_results.json\033[0m")
}
