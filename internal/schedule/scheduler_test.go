package schedule

import (
	"fmt"
	"strings"
	"testing"
)

// --- ValidIntervals ---

func TestValidIntervals(t *testing.T) {
	intervals := ValidIntervals()

	// Must contain the four supported intervals.
	expected := map[string]bool{
		"daily":     false,
		"weekly":    false,
		"every-12h": false,
		"every-6h":  false,
	}
	for _, iv := range intervals {
		if _, ok := expected[iv]; ok {
			expected[iv] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("ValidIntervals() missing %q", name)
		}
	}
}

func TestValidIntervals_NoDuplicates(t *testing.T) {
	intervals := ValidIntervals()
	seen := make(map[string]bool)
	for _, iv := range intervals {
		if seen[iv] {
			t.Errorf("ValidIntervals() contains duplicate %q", iv)
		}
		seen[iv] = true
	}
}

// --- IsValidInterval ---

func TestIsValidInterval_Valid(t *testing.T) {
	for _, iv := range []string{"daily", "weekly", "every-12h", "every-6h"} {
		if !IsValidInterval(iv) {
			t.Errorf("IsValidInterval(%q) = false, want true", iv)
		}
	}
}

func TestIsValidInterval_Invalid(t *testing.T) {
	invalid := []string{"", "hourly", "monthly", "daily-t", "Every-12h", "DAILY", "12h"}
	for _, iv := range invalid {
		if IsValidInterval(iv) {
			t.Errorf("IsValidInterval(%q) = true, want false", iv)
		}
	}
}

// --- ScheduleEntry ---

func TestScheduleEntry_Fields(t *testing.T) {
	entry := ScheduleEntry{
		Profile:  "work",
		Interval: "daily",
		Raw:      "0 2 * * * bak backup --profile work # bak-cli",
	}

	if entry.Profile != "work" {
		t.Errorf("Profile = %q, want 'work'", entry.Profile)
	}
	if entry.Interval != "daily" {
		t.Errorf("Interval = %q, want 'daily'", entry.Interval)
	}
	if entry.Raw == "" {
		t.Error("Raw should not be empty")
	}
}

// --- Scheduler interface check ---

func TestSchedulerInterface(t *testing.T) {
	// NewScheduler returns the platform-appropriate implementation.
	s := NewScheduler()
	if s == nil {
		t.Fatal("NewScheduler() returned nil")
	}
	// Verify the returned value satisfies the Scheduler interface.
	_, ok := interface{}(s).(Scheduler)
	if !ok {
		t.Fatalf("NewScheduler() returned %T which does not implement Scheduler", s)
	}
}

// --- Cron line format (cross-platform helpers) ---

func TestFormatCronLine(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		interval string
		wantCmd  string
	}{
		{name: "daily", profile: "work", interval: "daily", wantCmd: "0 2 * * *"},
		{name: "weekly", profile: "home", interval: "weekly", wantCmd: "0 3 * * 0"},
		{name: "every-12h", profile: "dev", interval: "every-12h", wantCmd: "0 */12 * * *"},
		{name: "every-6h", profile: "test", interval: "every-6h", wantCmd: "0 */6 * * *"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := formatCronLine(tt.profile, tt.interval)
			if len(line) == 0 {
				t.Fatal("formatCronLine returned empty string")
			}
			if len(line) < len(tt.wantCmd) || line[:len(tt.wantCmd)] != tt.wantCmd {
				t.Errorf("cron line = %q, want prefix %q", line, tt.wantCmd)
			}
			if !containsTag(line) {
				t.Errorf("cron line missing bak-cli tag: %q", line)
			}
		})
	}
}

func TestFormatCronLine_CommandContent(t *testing.T) {
	line := formatCronLine("work", "daily")
	wantCmd := "bak backup --profile work && bak push --profile work"
	if !containsSubstring(line, wantCmd) {
		t.Errorf("cron line = %q, want to contain %q", line, wantCmd)
	}
}

// --- Cron line parsing (cross-platform helpers) ---

func TestParseCronLine_Valid(t *testing.T) {
	line := "0 2 * * * bak backup --profile work && bak push --profile work # bak-cli:work"
	entry, ok := parseCronLine(line)
	if !ok {
		t.Fatal("parseCronLine returned false for valid tagged line")
	}
	if entry.Profile != "work" {
		t.Errorf("Profile = %q, want 'work'", entry.Profile)
	}
	if entry.Interval != "daily" {
		t.Errorf("Interval = %q, want 'daily'", entry.Interval)
	}
	if entry.Raw != line {
		t.Errorf("Raw = %q, want original line", entry.Raw)
	}
}

func TestParseCronLine_NotBakCli(t *testing.T) {
	lines := []string{
		"0 2 * * * /usr/bin/backup.sh",
		"# just a comment",
		"",
		"* * * * * some-other-job # other-tag",
	}
	for _, line := range lines {
		_, ok := parseCronLine(line)
		if ok {
			t.Errorf("parseCronLine(%q) = true, want false for non-bak-cli line", line)
		}
	}
}

func TestParseCronLine_Malformed(t *testing.T) {
	line := "0 2 * * * # bak-cli:work"
	_, ok := parseCronLine(line)
	if ok {
		t.Errorf("parseCronLine(%q) = true, want false for malformed line", line)
	}
}

func TestIntervalFromCron(t *testing.T) {
	tests := []struct {
		cronPrefix string
		want       string
	}{
		{"0 2 * * *", "daily"},
		{"0 3 * * 0", "weekly"},
		{"0 */12 * * *", "every-12h"},
		{"0 */6 * * *", "every-6h"},
		{"30 4 * * *", ""},
	}

	for _, tt := range tests {
		t.Run(tt.cronPrefix, func(t *testing.T) {
			got := intervalFromCron(tt.cronPrefix)
			if got != tt.want {
				t.Errorf("intervalFromCron(%q) = %q, want %q", tt.cronPrefix, got, tt.want)
			}
		})
	}
}

// --- Helpers ---

func containsTag(line string) bool {
	for _, tag := range []string{"# bak-cli:", "# bak-cli"} {
		if len(line) >= len(tag) {
			for i := 0; i <= len(line)-len(tag); i++ {
				if line[i:i+len(tag)] == tag {
					return true
				}
			}
		}
	}
	return false
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// --- mockScheduler for testing Scheduler logic cross-platform ---

// mockCronScheduler implements Scheduler using an in-memory string for testing.
type mockCronScheduler struct {
	content *string
}

func (m *mockCronScheduler) Create(profile string, interval string) error {
	current := *m.content
	newLine := formatCronLine(profile, interval)

	var lines []string
	if current != "" {
		for _, line := range strings.Split(strings.TrimSpace(current), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.Contains(line, "# bak-cli:"+profile) || strings.Contains(line, "# bak-cli "+profile) {
				continue
			}
			lines = append(lines, line)
		}
	}
	lines = append(lines, newLine)
	*m.content = strings.Join(lines, "\n") + "\n"
	return nil
}

func (m *mockCronScheduler) Remove(profile string) error {
	current := *m.content
	var lines []string
	found := false
	for _, line := range strings.Split(strings.TrimSpace(current), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "# bak-cli:"+profile) || strings.Contains(line, "# bak-cli "+profile) {
			found = true
			continue
		}
		lines = append(lines, line)
	}
	if !found {
		return fmt.Errorf("no schedule found for profile %q", profile)
	}
	*m.content = strings.Join(lines, "\n") + "\n"
	return nil
}

func (m *mockCronScheduler) List() ([]ScheduleEntry, error) {
	current := *m.content
	var entries []ScheduleEntry
	for _, line := range strings.Split(current, "\n") {
		line = strings.TrimSpace(line)
		if entry, ok := parseCronLine(line); ok {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func TestMockScheduler_CreateListRemove(t *testing.T) {
	var content string
	s := &mockCronScheduler{content: &content}

	// Create.
	if err := s.Create("work", "daily"); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := s.Create("home", "weekly"); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// List.
	entries, err := s.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("List() = %d entries, want 2", len(entries))
	}

	// Remove.
	if err := s.Remove("work"); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}

	entries, err = s.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("List() after Remove = %d entries, want 1", len(entries))
	}
	if entries[0].Profile != "home" {
		t.Errorf("remaining Profile = %q, want 'home'", entries[0].Profile)
	}
}

func TestMockScheduler_DuplicateCreate(t *testing.T) {
	var content string
	s := &mockCronScheduler{content: &content}

	s.Create("work", "daily")
	s.Create("work", "daily")

	entries, _ := s.List()
	if len(entries) != 1 {
		t.Fatalf("List() = %d entries, want 1 (duplicate should be replaced)", len(entries))
	}
}

// --- Schtasks argument building ---

func TestBuildSchtasksArgs_Create(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		interval string
		wantArgs []string // substrings that must appear in the args
	}{
		{
			name:     "daily",
			profile:  "work",
			interval: "daily",
			wantArgs: []string{"/create", "/tn", "bak-cli-work", "/sc", "daily", "/st", "02:00",
				"bak backup --profile work && bak push --profile work"},
		},
		{
			name:     "weekly",
			profile:  "home",
			interval: "weekly",
			wantArgs: []string{"/create", "/tn", "bak-cli-home", "/sc", "weekly", "/st", "03:00"},
		},
		{
			name:     "every-12h",
			profile:  "dev",
			interval: "every-12h",
			wantArgs: []string{"/create", "/tn", "bak-cli-dev", "/sc", "hourly", "/mo", "12"},
		},
		{
			name:     "every-6h",
			profile:  "test",
			interval: "every-6h",
			wantArgs: []string{"/create", "/tn", "bak-cli-test", "/sc", "hourly", "/mo", "6"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildSchtasksCreateArgs(tt.profile, tt.interval)
			argStr := strings.Join(args, " ")

			for _, want := range tt.wantArgs {
				if !strings.Contains(argStr, want) {
					t.Errorf("args missing %q in: %s", want, argStr)
				}
			}
		})
	}
}

func TestBuildSchtasksArgs_Create_Command(t *testing.T) {
	args := buildSchtasksCreateArgs("work", "daily")
	argStr := strings.Join(args, " ")
	wantCmd := "bak backup --profile work && bak push --profile work"
	if !strings.Contains(argStr, wantCmd) {
		t.Errorf("args = %s, want to contain %q", argStr, wantCmd)
	}
}

func TestBuildSchtasksArgs_Remove(t *testing.T) {
	args := buildSchtasksDeleteArgs("work")
	if len(args) == 0 {
		t.Fatal("buildSchtasksDeleteArgs returned empty")
	}
	argStr := strings.Join(args, " ")
	if !strings.Contains(argStr, "/delete") {
		t.Error("missing /delete flag")
	}
	if !strings.Contains(argStr, "bak-cli-work") {
		t.Error("missing task name bak-cli-work")
	}
}

func TestBuildSchtasksArgs_Query(t *testing.T) {
	args := buildSchtasksQueryArgs()
	if len(args) == 0 {
		t.Fatal("buildSchtasksQueryArgs returned empty")
	}
	argStr := strings.Join(args, " ")
	if !strings.Contains(argStr, "/query") {
		t.Error("missing /query flag")
	}
	if !strings.Contains(argStr, "/fo") {
		t.Error("missing /fo flag (format)")
	}
}

// --- Schtasks interval mapping ---

func TestIntervalToSchtasks(t *testing.T) {
	tests := []struct {
		interval string
		wantSC   string
		wantMO   string
		wantST   string
	}{
		{"daily", "daily", "", "02:00"},
		{"weekly", "weekly", "", "03:00"},
		{"every-12h", "hourly", "12", "00:00"},
		{"every-6h", "hourly", "6", "00:00"},
	}

	for _, tt := range tests {
		t.Run(tt.interval, func(t *testing.T) {
			sc, mo, st := intervalToSchtasksParams(tt.interval)
			if sc != tt.wantSC {
				t.Errorf("sc = %q, want %q", sc, tt.wantSC)
			}
			if mo != tt.wantMO {
				t.Errorf("mo = %q, want %q", mo, tt.wantMO)
			}
			if st != tt.wantST {
				t.Errorf("st = %q, want %q", st, tt.wantST)
			}
		})
	}
}

func TestIntervalToSchtasks_DefaultFallback(t *testing.T) {
	sc, mo, st := intervalToSchtasksParams("unknown")
	if sc != "daily" {
		t.Errorf("sc = %q, want 'daily' (default)", sc)
	}
	if mo != "" {
		t.Errorf("mo = %q, want ''", mo)
	}
	if st != "02:00" {
		t.Errorf("st = %q, want '02:00'", st)
	}
}

// --- Schtasks CSV parsing ---

func TestParseSchtasksCSV_ValidEntries(t *testing.T) {
	output := `"bak-cli-work","2026-06-06 02:00:00","Ready"
"bak-cli-home","2026-06-06 03:00:00","Ready"`
	entries := parseSchtasksCSV(output)
	if len(entries) != 2 {
		t.Fatalf("parseSchtasksCSV returned %d entries, want 2", len(entries))
	}
	if entries[0].Profile != "work" {
		t.Errorf("entry[0].Profile = %q, want 'work'", entries[0].Profile)
	}
	if entries[1].Profile != "home" {
		t.Errorf("entry[1].Profile = %q, want 'home'", entries[1].Profile)
	}
}

func TestParseSchtasksCSV_EmptyOutput(t *testing.T) {
	entries := parseSchtasksCSV("")
	if len(entries) != 0 {
		t.Errorf("parseSchtasksCSV(empty) = %d entries, want 0", len(entries))
	}
}

func TestParseSchtasksCSV_NonBakCLI(t *testing.T) {
	output := `"SomeTask","2026-06-06 02:00:00","Ready"
"AnotherTask","2026-06-06 04:00:00","Ready"`
	entries := parseSchtasksCSV(output)
	if len(entries) != 0 {
		t.Errorf("parseSchtasksCSV on non-bak-cli CSV = %d entries, want 0", len(entries))
	}
}

func TestParseSchtasksCSV_MixedEntries(t *testing.T) {
	output := `"SomeTask","2026-06-06 02:00:00","Ready"
"bak-cli-dev","2026-06-06 08:00:00","Ready"
"AnotherTask","2026-06-06 04:00:00","Ready"`
	entries := parseSchtasksCSV(output)
	if len(entries) != 1 {
		t.Fatalf("parseSchtasksCSV with mixed = %d entries, want 1", len(entries))
	}
	if entries[0].Profile != "dev" {
		t.Errorf("Profile = %q, want 'dev'", entries[0].Profile)
	}
}

func TestParseSchtasksCSV_SingleColumn(t *testing.T) {
	// Malformed CSV with only task name, no extra columns.
	output := `"bak-cli-test"`
	entries := parseSchtasksCSV(output)
	if len(entries) != 1 {
		t.Fatalf("parseSchtasksCSV with single column = %d entries, want 1", len(entries))
	}
	if entries[0].Profile != "test" {
		t.Errorf("Profile = %q, want 'test'", entries[0].Profile)
	}
}

func TestParseSchtasksCSV_WhitespaceTrim(t *testing.T) {
	output := "\n\"bak-cli-ws\"\n\n"
	entries := parseSchtasksCSV(output)
	if len(entries) != 1 {
		t.Fatalf("parseSchtasksCSV with whitespace = %d entries, want 1", len(entries))
	}
	if entries[0].Profile != "ws" {
		t.Errorf("Profile = %q, want 'ws'", entries[0].Profile)
	}
}

// --- formatCronLine default fallback ---

func TestFormatCronLine_DefaultFallback(t *testing.T) {
	line := formatCronLine("test", "unknown-interval")
	if len(line) == 0 {
		t.Fatal("formatCronLine should not return empty")
	}
	if line[:9] != "0 2 * * *" {
		t.Errorf("line prefix = %q, want '0 2 * * *' (default daily)", line[:9])
	}
	if !strings.Contains(line, "# bak-cli:test") {
		t.Errorf("missing bak-cli:test tag in: %s", line)
	}
}

// --- parseCronLine additional edge cases ---

func TestParseCronLine_TagWithoutColon(t *testing.T) {
	// Bare "# bak-cli" tag (no colon or profile) is parsed but profile is empty.
	line := "0 2 * * * /bin/backup.sh # bak-cli"
	entry, ok := parseCronLine(line)
	if !ok {
		t.Fatal("parseCronLine returned false for valid tagged line")
	}
	if entry.Profile != "" {
		t.Errorf("Profile = %q, want '' (no profile after bare # bak-cli)", entry.Profile)
	}
}

func TestParseCronLine_WhitespaceOnly(t *testing.T) {
	_, ok := parseCronLine("   \t  ")
	if ok {
		t.Error("parseCronLine on whitespace = true, want false")
	}
}

func TestParseCronLine_CommentedBakCLI(t *testing.T) {
	line := "# 0 2 * * * bak backup --profile work # bak-cli:work"
	_, ok := parseCronLine(line)
	if ok {
		t.Error("parseCronLine on commented line = true, want false")
	}
}

func TestParseCronLine_FewerThan7Fields(t *testing.T) {
	// Needs at least 7 fields. With 6 fields (5 cron + 1 command that is '#'),
	// the 6th field starts with '#' → rejected.
	line := "0 2 * * * # bak-cli:work"
	_, ok := parseCronLine(line)
	if ok {
		t.Error("parseCronLine with 6th field = '#' = true, want false")
	}
}

func TestParseCronLine_Exactly6FieldsNoTag(t *testing.T) {
	line := "0 2 * * * cmd"
	_, ok := parseCronLine(line)
	if ok {
		t.Error("parseCronLine on non-tagged cron line = true, want false")
	}
}
