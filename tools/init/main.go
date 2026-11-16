// Command init turns a copy of the template into your own project.
//
//	go run ./tools/init                                   # asks what it needs
//	go run ./tools/init -yes -module github.com/acme/billing -db postgres -example=false
//
// It renames the module and the application, removes the options you turn
// off together with their dependencies, deletes everything that belongs to
// the template itself (its website, changelog and this tool) and leaves a
// project that builds and passes its tests.
//
// The template marks optional code with comments:
//
//	x := 1 // init:feature            a single line
//	// init:feature:begin ... end      a block
//
// Removing a feature deletes the marked code; keeping it deletes only the
// markers. Nothing here is specific to Go, the same markers work in YAML,
// Makefiles and Markdown.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	templateModule = "github.com/antoniopaya22/go-rest-template"
	templateRepo   = "antoniopaya22/go-rest-template"
	templateName   = "go-rest-template"
	templateSnake  = "go_rest_template"
	templateTitle  = "Go REST Template"
)

// options is what the user chooses.
type options struct {
	dir    string
	module string
	name   string
	db     string

	example       bool
	metrics       bool
	tracing       bool
	rateLimit     bool
	observability bool
	helm          bool
	devcontainer  bool

	yes bool
}

func main() {
	var o options
	flag.StringVar(&o.dir, "dir", ".", "directory of the project to initialize")
	flag.StringVar(&o.module, "module", "", "Go module path of the new project, for example github.com/acme/billing")
	flag.StringVar(&o.name, "name", "", "application name (default: last element of the module path)")
	flag.StringVar(&o.db, "db", "both", "database drivers to keep: both, postgres or sqlite")
	flag.BoolVar(&o.example, "example", true, "keep the tasks example resource and the resource generator")
	flag.BoolVar(&o.metrics, "metrics", true, "keep Prometheus metrics")
	flag.BoolVar(&o.tracing, "tracing", true, "keep OpenTelemetry tracing")
	flag.BoolVar(&o.rateLimit, "ratelimit", true, "keep rate limiting")
	flag.BoolVar(&o.observability, "observability", true, "keep the Prometheus, Grafana and Tempo compose profile (needs postgres, metrics and tracing)")
	flag.BoolVar(&o.helm, "helm", true, "keep the Helm chart (needs postgres)")
	flag.BoolVar(&o.devcontainer, "devcontainer", true, "keep the dev container")
	flag.BoolVar(&o.yes, "yes", false, "do not ask questions; use the flags and their defaults")
	flag.Parse()

	if !o.yes {
		if err := ask(&o, os.Stdin, os.Stdout); err != nil {
			fail(err)
		}
	}

	if err := run(o, os.Stdout); err != nil {
		fail(err)
	}
}

// say prints to the terminal; a failed write there is not worth handling.
func say(out io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(out, format, args...)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

// ask fills the options interactively. Values given as flags are the defaults.
func ask(o *options, in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)

	text := func(question string, fallback string) (string, error) {
		if fallback != "" {
			say(out, "%s [%s]: ", question, fallback)
		} else {
			say(out, "%s: ", question)
		}
		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return "", fmt.Errorf("read answer: %w (use -yes to run without questions)", err)
		}
		if answer := strings.TrimSpace(line); answer != "" {
			return answer, nil
		}
		return fallback, nil
	}

	yesNo := func(question string, value *bool) error {
		fallback := "Y/n"
		if !*value {
			fallback = "y/N"
		}
		answer, err := text(question, fallback)
		if err != nil {
			return err
		}
		switch strings.ToLower(answer) {
		case "y", "yes":
			*value = true
		case "n", "no":
			*value = false
		}
		return nil
	}

	say(out, "This turns the template into your project. Press Enter to accept a default.\n\n")

	var err error
	if o.module, err = text("Go module path (for example github.com/acme/billing)", o.module); err != nil {
		return err
	}
	if o.name, err = text("Application name", defaultName(o.module, o.name)); err != nil {
		return err
	}
	if o.db, err = text("Database drivers: both, postgres or sqlite", o.db); err != nil {
		return err
	}

	questions := []struct {
		text  string
		value *bool
	}{
		{"Keep the tasks example resource and the resource generator?", &o.example},
		{"Keep Prometheus metrics?", &o.metrics},
		{"Keep OpenTelemetry tracing?", &o.tracing},
		{"Keep rate limiting?", &o.rateLimit},
		{"Keep the Prometheus, Grafana and Tempo compose profile?", &o.observability},
		{"Keep the Helm chart?", &o.helm},
		{"Keep the dev container?", &o.devcontainer},
	}
	for _, question := range questions {
		if err := yesNo(question.text, question.value); err != nil {
			return err
		}
	}

	say(out, "\n")
	return nil
}

func defaultName(module string, name string) string {
	if name != "" {
		return name
	}
	return module[strings.LastIndex(module, "/")+1:]
}

var (
	validModule = regexp.MustCompile(`^[A-Za-z0-9._~-]+(/[A-Za-z0-9._~-]+)+$`)
	validName   = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

func run(o options, out io.Writer) error {
	o.module = strings.TrimSpace(o.module)
	o.name = defaultName(o.module, strings.TrimSpace(o.name))
	o.db = strings.ToLower(o.db)

	switch {
	case !validModule.MatchString(o.module):
		return errors.New("-module must be a Go module path such as github.com/acme/billing")
	case o.module == templateModule:
		return errors.New("-module must differ from the module path of the template")
	case !validName.MatchString(o.name):
		return fmt.Errorf("the application name %q must be lowercase letters, digits and dashes; set -name", o.name)
	case o.db != "both" && o.db != "postgres" && o.db != "sqlite":
		return errors.New("-db must be both, postgres or sqlite")
	}

	current, err := os.ReadFile(filepath.Join(o.dir, "go.mod"))
	if err != nil {
		return fmt.Errorf("%s is not the root of the project: %w", o.dir, err)
	}
	if !bytes.Contains(current, []byte("module "+templateModule+"\n")) {
		return errors.New("this project has already been initialized")
	}

	// A feature is kept when its flag is on and everything it needs is kept.
	postgres := o.db != "sqlite"
	keep := map[string]bool{
		"sqlite":        o.db != "postgres",
		"postgres":      postgres,
		"example":       o.example,
		"metrics":       o.metrics,
		"otel":          o.tracing,
		"ratelimit":     o.rateLimit,
		"compose":       postgres,
		"observability": o.observability && postgres && o.metrics && o.tracing,
		"helm":          o.helm && postgres,
		"devcontainer":  o.devcontainer,
		// The template's own machinery never survives.
		"template": false,
		"site":     false,
	}

	removed, err := removePaths(o.dir, keep)
	if err != nil {
		return err
	}

	owner := ownerRepo(o.module, o.name)
	rename := strings.NewReplacer(
		templateModule, o.module,
		"ghcr.io/"+templateRepo, "ghcr.io/"+owner,
		templateRepo, owner,
		templateName, o.name,
		templateSnake, strings.ReplaceAll(o.name, "-", "_"),
		templateTitle, o.name,
	)

	if err := rewriteFiles(o.dir, keep, rename); err != nil {
		return err
	}

	chart := filepath.Join(o.dir, "deployments", "helm")
	if _, err := os.Stat(filepath.Join(chart, templateName)); err == nil && o.name != templateName {
		if err := os.Rename(filepath.Join(chart, templateName), filepath.Join(chart, o.name)); err != nil {
			return err
		}
	}

	if err := resetReleaseConfig(o.dir, keep["helm"]); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(o.dir, "README.md"), []byte(readme(o, keep)), 0o600); err != nil {
		return err
	}

	// Dropping a feature drops its dependencies, and the example environment
	// file lists exactly the variables that are left.
	if err := command(o.dir, nil, "go", "mod", "tidy"); err != nil {
		return err
	}
	example, err := os.Create(filepath.Join(o.dir, "configs", "env.example")) //nolint:gosec // fixed project path.
	if err != nil {
		return err
	}
	defer func() { _ = example.Close() }()
	if err := command(o.dir, example, "go", "run", "./cmd/api", "-print-config", "env"); err != nil {
		return err
	}

	say(out, "%s is ready.\n\n  module   %s\n  kept     %s\n  removed  %s\n\n", o.name, o.module, list(keep, true), list(keep, false))
	say(out, "Deleted %d files and directories (template machinery and the options turned off).\n", len(removed))
	say(out, "%s", `
Next:
  1. make check          build, lint and test what was generated
  2. make run            start the API on http://localhost:8080 (docs at /docs)
  3. Review LICENSE: it is the MIT license of the template. Add your own
     copyright line or replace it with the license of your project.
  4. Add CONTRIBUTING.md and SECURITY.md that describe your project.
  5. If this directory is a clone of the template, start a new history:
       rm -rf .git && git init
`)
	return nil
}

// ownerRepo derives "owner/repository" from a module path hosted on a forge,
// which is what container registries and repository links use.
func ownerRepo(module string, name string) string {
	parts := strings.Split(module, "/")
	if len(parts) >= 3 && strings.Contains(parts[0], ".") {
		return parts[1] + "/" + parts[2]
	}
	return name
}

func list(keep map[string]bool, kept bool) string {
	labels := map[string]string{
		"sqlite": "sqlite", "postgres": "postgres", "example": "example resource", "metrics": "metrics",
		"otel": "tracing", "ratelimit": "rate limiting", "observability": "observability stack",
		"helm": "helm chart", "devcontainer": "dev container",
	}

	var names []string
	for feature, label := range labels {
		if keep[feature] == kept {
			names = append(names, label)
		}
	}
	if len(names) == 0 {
		return "nothing"
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// paths lists what each feature owns outside of marked code.
var paths = map[string][]string{
	"template": {
		"tools/init", "scripts", "docs", "overrides", "mkdocs.yml", "requirements-docs.txt",
		"CHANGELOG.md", "CONTRIBUTING.md", "CODE_OF_CONDUCT.md", "SECURITY.md", "CITATION.cff",
		".github/ISSUE_TEMPLATE", ".github/CODEOWNERS", ".github/FUNDING.yml",
		".github/workflows/docs.yml", ".github/workflows/scorecard.yml",
		"deployments/README.md",
	},
	"example":       {"internal/tasks", "migrations/postgres/00002_tasks.sql", "migrations/sqlite/00002_tasks.sql", "tools/scaffold"},
	"otel":          {"internal/platform/tracing"},
	"metrics":       {"internal/platform/middleware/metrics.go"},
	"ratelimit":     {"internal/platform/middleware/ratelimit.go"},
	"sqlite":        {"internal/platform/database/sqlite.go", "migrations/sqlite"},
	"postgres":      {"internal/platform/database/postgres.go", "migrations/postgres"},
	"compose":       {"deployments/compose.yaml"},
	"observability": {"deployments/observability"},
	"helm":          {"deployments/helm"},
	"devcontainer":  {".devcontainer"},
}

func removePaths(dir string, keep map[string]bool) ([]string, error) {
	var removed []string
	for feature, owned := range paths {
		if keep[feature] {
			continue
		}
		for _, path := range owned {
			full := filepath.Join(dir, filepath.FromSlash(path))
			if _, err := os.Lstat(full); err != nil {
				continue
			}
			if err := os.RemoveAll(full); err != nil {
				return nil, err
			}
			removed = append(removed, path)
		}
	}

	// Directories left empty by the removals.
	for _, path := range []string{"tools", "deployments"} {
		full := filepath.Join(dir, path)
		if entries, err := os.ReadDir(full); err == nil && len(entries) == 0 {
			if err := os.Remove(full); err != nil {
				return nil, err
			}
		}
	}

	sort.Strings(removed)
	return removed, nil
}

var skippedDirs = map[string]bool{".git": true, "bin": true, "data": true, "dist": true, "site": true, "node_modules": true}

// rewriteFiles applies the markers and the renames to every text file.
func rewriteFiles(dir string, keep map[string]bool, rename *strings.Replacer) error {
	return filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skippedDirs[entry.Name()] && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}

		original, err := os.ReadFile(path) //nolint:gosec // walking the project tree.
		if err != nil {
			return err
		}
		if bytes.IndexByte(original, 0) >= 0 {
			return nil // binary file
		}

		updated, err := applyMarkers(string(original), keep)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		updated = rename.Replace(updated)

		if strings.HasSuffix(path, ".go") && updated != string(original) {
			formatted, err := format.Source([]byte(updated))
			if err != nil {
				return fmt.Errorf("format %s: %w", path, err)
			}
			updated = string(formatted)
		}

		if updated == string(original) {
			return nil
		}
		return os.WriteFile(path, []byte(updated), 0o600) //nolint:gosec // walking the project tree.
	})
}

var (
	// A line that is only a block marker, in any comment syntax.
	blockMarker = regexp.MustCompile(`^\s*(?://|#|<!--)\s*(init|scaffold):([a-z]+)(?::[a-z]+)?:(begin|end)\s*(?:-->)?\s*$`)
	// A marker at the end of a line of code.
	lineMarker = regexp.MustCompile(`\s*(?://|#)\s*init:([a-z]+)\s*$`)
	// An anchor used by the resource generator, alone on its line.
	scaffoldAnchor = regexp.MustCompile(`^\s*(?://|#)\s*scaffold:[a-z]+\s*$`)
)

// applyMarkers removes the code of dropped features and the markers of kept ones.
//
// The scaffold markers belong to the example feature: they delimit the parts
// of the tasks example in the OpenAPI document and mark where the generator
// inserts code, so they stay only while the example does.
func applyMarkers(content string, keep map[string]bool) (string, error) {
	lines := strings.SplitAfter(content, "\n")
	result := make([]string, 0, len(lines))

	// Stack of open blocks; dropping counts the ones being removed.
	var open []bool
	dropping := 0

	for _, line := range lines {
		text := strings.TrimRight(line, "\r\n")

		if match := blockMarker.FindStringSubmatch(text); match != nil {
			kind, feature, edge := match[1], match[2], match[3]

			kept := keep[feature]
			if kind == "scaffold" {
				kept = keep["example"]
			} else if _, known := keep[feature]; !known {
				return "", fmt.Errorf("unknown feature %q in marker %q", feature, strings.TrimSpace(text))
			}

			if edge == "begin" {
				open = append(open, kept)
				if !kept {
					dropping++
				}
			} else {
				if len(open) == 0 {
					return "", fmt.Errorf("marker %q closes a block that was never opened", strings.TrimSpace(text))
				}
				if !open[len(open)-1] {
					dropping--
				}
				open = open[:len(open)-1]
			}

			// Scaffold block markers stay with the example so the generator
			// can still find its source; every other marker goes.
			if kind == "scaffold" && kept && dropping == 0 {
				result = append(result, line)
			}
			continue
		}

		if dropping > 0 {
			continue
		}

		if scaffoldAnchor.MatchString(text) && !keep["example"] {
			continue
		}

		if match := lineMarker.FindStringSubmatch(text); match != nil {
			kept, known := keep[match[1]]
			if !known {
				return "", fmt.Errorf("unknown feature %q in marker %q", match[1], strings.TrimSpace(text))
			}
			if !kept {
				continue
			}
			line = lineMarker.ReplaceAllString(text, "") + line[len(text):]
		}

		result = append(result, line)
	}

	if len(open) > 0 {
		return "", errors.New("a marker block is never closed")
	}
	return collapseBlankLines(strings.Join(result, "")), nil
}

var blankRun = regexp.MustCompile(`\n{3,}`)

// collapseBlankLines removes the double blank lines that deleting a block leaves behind.
func collapseBlankLines(content string) string {
	return blankRun.ReplaceAllString(content, "\n\n")
}

// resetReleaseConfig points release automation at a project with no releases yet.
func resetReleaseConfig(dir string, helm bool) error {
	manifest := filepath.Join(dir, ".release-please-manifest.json")
	if _, err := os.Stat(manifest); err == nil {
		if err := os.WriteFile(manifest, []byte("{\n  \".\": \"0.0.0\"\n}\n"), 0o600); err != nil {
			return err
		}
	}

	path := filepath.Join(dir, "release-please-config.json")
	raw, err := os.ReadFile(path) //nolint:gosec // fixed project path.
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	// The bootstrap commit belongs to the history of the template.
	delete(config, "bootstrap-sha")
	if !helm {
		if packages, ok := config["packages"].(map[string]any); ok {
			if root, ok := packages["."].(map[string]any); ok {
				delete(root, "extra-files")
			}
		}
	}

	encoded, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o600)
}

func command(dir string, stdout io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), name, args...) //nolint:gosec // fixed commands.
	cmd.Dir = dir
	cmd.Stdout = stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, stderr.String())
	}
	return nil
}

func readme(o options, keep map[string]bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", o.name)
	b.WriteString("REST API built from [go-rest-template](https://github.com/" + templateRepo + ").\n\n")

	b.WriteString("## Run it\n\n```sh\nmake run\n```\n\n")
	b.WriteString("The API listens on <http://localhost:8080>. The interactive reference is at\n")
	b.WriteString("<http://localhost:8080/docs> and the OpenAPI document at `/openapi.yaml`.\n")
	if keep["sqlite"] {
		b.WriteString("In development it uses a SQLite file under `data/`, so nothing else is needed.\n")
	} else {
		b.WriteString("It needs PostgreSQL: `make compose-up` starts the API together with a database.\n")
	}

	b.WriteString("\nTo sign in, create the first administrator at startup:\n\n```sh\n")
	b.WriteString("BOOTSTRAP_ADMIN_ENABLED=true BOOTSTRAP_ADMIN_USERNAME=admin \\\n")
	b.WriteString("BOOTSTRAP_ADMIN_FIRST_NAME=Ada BOOTSTRAP_ADMIN_LAST_NAME=Lovelace \\\n")
	b.WriteString("BOOTSTRAP_ADMIN_PASSWORD=change-me-now-123 make run\n```\n\n")

	b.WriteString("## Configure it\n\n")
	b.WriteString("Everything is configured with environment variables. [configs/env.example](configs/env.example)\n")
	b.WriteString("lists each one with its default; copy it to `.env` to override values locally.\n\n")

	b.WriteString("## Work on it\n\n```sh\nmake help     # every available target\nmake check    # tidy, vet, lint and test\n")
	if keep["example"] {
		b.WriteString("make resource name=project   # generate a new resource from the tasks example\n")
	}
	b.WriteString("```\n\n")

	b.WriteString("## Layout\n\n```text\n")
	b.WriteString("api/            OpenAPI contract, embedded in the binary and enforced at runtime\n")
	b.WriteString("cmd/            entry points: api (the server) and migrate\n")
	b.WriteString("internal/app    composition root: wiring and routes\n")
	b.WriteString("internal/auth   tokens, passwords and the authentication middleware\n")
	b.WriteString("internal/users  the users feature\n")
	if keep["example"] {
		b.WriteString("internal/tasks  an example feature to copy from\n")
	}
	b.WriteString("internal/platform  shared infrastructure: database, HTTP conventions, middleware\n")
	b.WriteString("migrations/     SQL migrations, embedded in the binary\n")
	b.WriteString("```\n\n")
	b.WriteString("The guides of the template apply to this project:\n<https://antoniopaya22.github.io/go-rest-template/>\n")
	return b.String()
}
