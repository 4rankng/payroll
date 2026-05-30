package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

type TestStatus string

const (
	StatusPass TestStatus = "PASS"
	StatusFail TestStatus = "FAIL"
	StatusSkip TestStatus = "SKIP"
)

type TestResult struct {
	Name     string
	Flow     string
	Status   TestStatus
	Duration time.Duration
	Error    string
}

type Reporter struct {
	results []TestResult
	start   time.Time
	passed  int
	failed  int
	skipped int
	w       io.Writer
}

func NewReporter() *Reporter {
	return &Reporter{start: time.Now(), w: os.Stdout}
}

// SetOutput sets the writer for all reporter output.
func (r *Reporter) SetOutput(w io.Writer) {
	r.w = w
}

func (r *Reporter) printf(format string, args ...any) {
	_, _ = fmt.Fprintf(r.w, format, args...)
}

func (r *Reporter) println(args ...any) {
	_, _ = fmt.Fprintln(r.w, args...)
}

func (r *Reporter) Record(result TestResult) {
	r.results = append(r.results, result)
	switch result.Status {
	case StatusPass:
		r.passed++
	case StatusFail:
		r.failed++
	case StatusSkip:
		r.skipped++
	}
}

func (r *Reporter) PrintResult(result TestResult) {
	statusStr := ""
	switch result.Status {
	case StatusPass:
		statusStr = fmt.Sprintf("%s  PASS  %s", colorGreen, colorReset)
	case StatusFail:
		statusStr = fmt.Sprintf("%s  FAIL  %s", colorRed, colorReset)
	case StatusSkip:
		statusStr = fmt.Sprintf("%s  SKIP  %s", colorYellow, colorReset)
	}

	duration := fmt.Sprintf("%s(%.1fs)%s", colorDim, result.Duration.Seconds(), colorReset)
	r.printf("  %s %s %s\n", statusStr, result.Name, duration)

	if result.Status == StatusFail && result.Error != "" {
		indented := indentLines(result.Error, "         ")
		r.printf("%s%s%s\n", colorRed, indented, colorReset)
	}
}

func (r *Reporter) PrintHeader(baseURL string) {
	r.println()
	r.printf("%s%s═══════════════════════════════════════════════════════════%s\n", colorBold, colorCyan, colorReset)
	r.printf("%s  Payroll Backend - Integration Tests%s\n", colorBold, colorReset)
	r.printf("%s%s═══════════════════════════════════════════════════════════%s\n", colorBold, colorCyan, colorReset)
	r.printf("  Base URL: %s\n", baseURL)
	r.printf("  Started:  %s\n", time.Now().Format("2006-01-02 15:04:05"))
	r.println()
}

func (r *Reporter) PrintSection(name string) {
	r.println()
	r.printf("%s%s── %s ──%s\n", colorBold, colorCyan, name, colorReset)
	r.println()
}

func (r *Reporter) PrintDiscovery(data *TestData) {
	r.printf("  Discovered: %d projects, %d employees, %d banks\n",
		len(data.Projects), len(data.Employees), len(data.Banks))
	if data.WeeklyProject != nil {
		r.printf("  Weekly project: %s (ID %d, %d weekly employees)\n",
			data.WeeklyProject.Name, data.WeeklyProject.ID, data.WeeklyProject.WeeklySalaryEmployeeCount)
	}
	if data.MonthlyProject != nil {
		r.printf("  Monthly project: %s (ID %d, %d monthly employees)\n",
			data.MonthlyProject.Name, data.MonthlyProject.ID, data.MonthlyProject.MonthlySalaryEmployeeCount)
	}
	if data.EmployeeForAdvance != nil {
		r.printf("  Advance payment employee: %s (ID %d, username: %s)\n",
			data.EmployeeForAdvance.Fullname, data.EmployeeForAdvance.ID, ptrStr(data.EmployeeForAdvance.Username))
	}
	if len(data.Partners) > 0 {
		names := make([]string, len(data.Partners))
		for i, p := range data.Partners {
			names[i] = p.Username
		}
		r.printf("  Partner users: %s\n", strings.Join(names, ", "))
	}
	r.println()
}

func (r *Reporter) PrintSummary() {
	total := r.passed + r.failed + r.skipped
	elapsed := time.Since(r.start)

	r.println()
	r.printf("%s%s═══════════════════════════════════════════════════════════%s\n", colorBold, colorCyan, colorReset)
	r.printf("%s  Test Results%s\n", colorBold, colorReset)
	r.printf("%s%s═══════════════════════════════════════════════════════════%s\n", colorBold, colorCyan, colorReset)

	if len(r.results) == 0 {
		r.println("  No tests were executed.")
		return
	}

	// Table header
	padName := 50
	r.printf("  ┌──────────┬─%s─┬──────────┐\n", strings.Repeat("─", padName))
	r.printf("  │ %-8s │ %-*s │ %-8s │\n", "Status", padName, "Test Name", "Time")
	r.printf("  ├──────────┼─%s─┼──────────┤\n", strings.Repeat("─", padName))

	// Table rows grouped by flow
	currentFlow := ""
	for _, res := range r.results {
		if res.Flow != currentFlow {
			if currentFlow != "" {
				r.printf("  ├──────────┼─%s─┼──────────┤\n", strings.Repeat("─", padName))
			}
			currentFlow = res.Flow
			r.printf("  │ %s%-8s%s │ %-*s │ %-8s │\n",
				colorBold, currentFlow, colorReset,
				padName, "", "")
		}

		statusDisplay := ""
		switch res.Status {
		case StatusPass:
			statusDisplay = fmt.Sprintf("%s  PASS  %s", colorGreen, colorReset)
		case StatusFail:
			statusDisplay = fmt.Sprintf("%s  FAIL  %s", colorRed, colorReset)
		case StatusSkip:
			statusDisplay = fmt.Sprintf("%s  SKIP  %s", colorYellow, colorReset)
		}

		name := truncate(res.Name, padName)
		r.printf("  │ %s │ %-*s │ %s%5.1fs%s  │\n",
			statusDisplay, padName, name,
			colorDim, res.Duration.Seconds(), colorReset)

		if res.Status == StatusFail && res.Error != "" {
			wrapped := wrapText(res.Error, padName-2)
			for _, line := range wrapped {
				r.printf("  │ %-8s   %-*s   %-8s  │\n", "", padName, "  "+line, "")
			}
		}
	}

	r.printf("  └──────────┴─%s─┴──────────┘\n", strings.Repeat("─", padName))

	r.println()
	r.printf("  Duration: %s\n", elapsed.Round(time.Second))
	r.printf("  Total: %d | %sPassed: %d%s | %sFailed: %d%s | %sSkipped: %d%s\n",
		total,
		colorGreen, r.passed, colorReset,
		colorRed, r.failed, colorReset,
		colorYellow, r.skipped, colorReset)
	r.println()
}

func (r *Reporter) ExitCode() int {
	if r.failed > 0 {
		return 1
	}
	return 0
}

// RunTest executes a test function, records and prints the result.
func (r *Reporter) RunTest(flow, name string, fn func() error) {
	start := time.Now()
	var err error
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				err = fmt.Errorf("panic: %v", rec)
			}
		}()
		err = fn()
	}()

	result := TestResult{
		Name:     name,
		Flow:     flow,
		Duration: time.Since(start),
	}

	if err != nil {
		result.Status = StatusFail
		result.Error = err.Error()
	} else {
		result.Status = StatusPass
	}

	r.Record(result)
	r.PrintResult(result)
}

func (r *Reporter) Skip(flow, name, reason string) {
	result := TestResult{
		Name:   name,
		Flow:   flow,
		Status: StatusSkip,
		Error:  reason,
	}
	r.Record(result)
	r.PrintResult(result)
}

func indentLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func ptrStr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func wrapText(s string, width int) []string {
	if width <= 0 {
		width = 40
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	current := ""
	for _, word := range words {
		if len(current)+len(word)+1 > width {
			if current != "" {
				lines = append(lines, current)
			}
			if len(word) > width {
				for len(word) > width {
					lines = append(lines, word[:width])
					word = word[width:]
				}
				current = word
			} else {
				current = word
			}
		} else {
			if current == "" {
				current = word
			} else {
				current += " " + word
			}
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
