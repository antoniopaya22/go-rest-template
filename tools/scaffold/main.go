// Command scaffold generates a new resource by cloning the tasks example.
//
//	go run ./tools/scaffold -name project
//
// It copies internal/tasks into a new feature package, adds the migrations,
// documents the endpoints in api/openapi.yaml and wires the handler into the
// application. The result compiles, passes its tests and exposes a working
// CRUD API; from there you rename fields and change the rules.
package main

import (
	"errors"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	sourceSingular = "task"
	sourcePlural   = "tasks"
)

func main() {
	name := flag.String("name", "", "singular name of the resource, one lowercase word (for example: project)")
	plural := flag.String("plural", "", "plural name, when adding an s is not right (for example: categories)")
	flag.Parse()

	if err := run(strings.ToLower(*name), strings.ToLower(*plural)); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

var word = regexp.MustCompile(`^[a-z]+$`)

func run(singular string, plural string) error {
	if plural == "" {
		plural = pluralize(singular)
	}
	switch {
	case !word.MatchString(singular) || !word.MatchString(plural):
		return errors.New("-name and -plural must be a single lowercase word, such as project or categories")
	case singular == plural:
		return errors.New("the singular and plural names must differ; set -plural")
	case singular == sourceSingular || plural == sourcePlural || plural == "users" || plural == "auth":
		return fmt.Errorf("%q is already taken", plural)
	}

	source := filepath.Join("internal", sourcePlural)
	target := filepath.Join("internal", plural)
	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("the %s example is required as the source: %w", source, err)
	}
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("%s already exists", target)
	}

	rename := renamer(singular, plural)
	var created []string

	// 1. The feature package.
	files, err := filepath.Glob(filepath.Join(source, "*.go"))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(target, 0o750); err != nil {
		return err
	}
	for _, file := range files {
		content, err := os.ReadFile(file) //nolint:gosec // path comes from a glob of the source tree.
		if err != nil {
			return err
		}

		formatted, err := format.Source([]byte(rename(removeOmitted(string(content)))))
		if err != nil {
			return fmt.Errorf("format %s: %w", file, err)
		}

		destination := filepath.Join(target, rename(filepath.Base(file)))
		if err := os.WriteFile(destination, formatted, 0o600); err != nil {
			return err
		}
		created = append(created, destination)
	}

	// 2. One migration per database driver.
	migrations, err := filepath.Glob(filepath.Join("migrations", "*", "*_"+sourcePlural+".sql"))
	if err != nil {
		return err
	}
	for _, file := range migrations {
		content, err := os.ReadFile(file) //nolint:gosec // path comes from a glob of the source tree.
		if err != nil {
			return err
		}

		directory := filepath.Dir(file)
		next, err := nextMigration(directory)
		if err != nil {
			return err
		}

		destination := filepath.Join(directory, fmt.Sprintf("%05d_%s.sql", next, plural))
		if err := os.WriteFile(destination, []byte(rename(string(content))), 0o600); err != nil {
			return err
		}
		created = append(created, destination)
	}

	// 3. The OpenAPI contract.
	spec := filepath.Join("api", "openapi.yaml")
	if err := edit(spec, func(content string) (string, error) {
		for block, anchor := range map[string]string{"tag": "tags", "paths": "paths", "response": "responses", "schemas": "schemas"} {
			body, err := between(content, "# scaffold:"+sourcePlural+":"+block+":begin", "# scaffold:"+sourcePlural+":"+block+":end")
			if err != nil {
				return "", err
			}
			content, err = insertBefore(content, "# scaffold:"+anchor, rename(body))
			if err != nil {
				return "", err
			}
		}
		return content, nil
	}); err != nil {
		return err
	}

	// 4. The composition root.
	module, err := modulePath()
	if err != nil {
		return err
	}
	if err := edit(filepath.Join("internal", "app", "app.go"), func(content string) (string, error) {
		content, err := insertBefore(content, "// scaffold:fields", fmt.Sprintf("\t%s *%s.Handler\n", plural, plural))
		if err != nil {
			return "", err
		}
		content, err = insertBefore(content, "// scaffold:wiring",
			fmt.Sprintf("\ta.%[1]s = %[1]s.NewHandler(%[1]s.NewService(%[1]s.NewRepository(a.db)))\n", plural))
		if err != nil {
			return "", err
		}

		// Added next to an existing import; gofmt sorts the block.
		anchor := fmt.Sprintf("\t%q\n", module+"/internal/auth")
		if !strings.Contains(content, anchor) {
			return "", errors.New("internal/app/app.go does not import the auth package")
		}
		content = strings.Replace(content, anchor, anchor+fmt.Sprintf("\t%q\n", module+"/internal/"+plural), 1)

		formatted, err := format.Source([]byte(content))
		return string(formatted), err
	}); err != nil {
		return err
	}
	if err := edit(filepath.Join("internal", "app", "routes.go"), func(content string) (string, error) {
		return insertBefore(content, "// scaffold:routes", fmt.Sprintf("\t\ta.%s.Routes(r, a.authn)\n", plural))
	}); err != nil {
		return err
	}

	fmt.Printf("Created the %s resource:\n", plural)
	for _, file := range created {
		fmt.Println("  " + file)
	}
	fmt.Printf(`
Updated api/openapi.yaml, internal/app/app.go and internal/app/routes.go.

Next:
  1. Edit the model in internal/%[1]s/%[2]s.go and the columns in the new migrations.
  2. Keep the %[3]s schemas in api/openapi.yaml in sync with the model.
  3. Run: make check
`, plural, singular, title(singular))
	return nil
}

// renamer returns a function that rewrites every form of the source name:
// tasks, Tasks, task and Task. It works in a single pass, so a new name that
// happens to contain "task" is not rewritten twice.
func renamer(singular string, plural string) func(string) string {
	replacements := map[string]string{
		sourcePlural:          plural,
		title(sourcePlural):   title(plural),
		sourceSingular:        singular,
		title(sourceSingular): title(singular),
	}
	pattern := regexp.MustCompile(title(sourcePlural) + "|" + sourcePlural + "|" + title(sourceSingular) + "|" + sourceSingular)

	return func(text string) string {
		return pattern.ReplaceAllStringFunc(text, func(match string) string { return replacements[match] })
	}
}

func title(s string) string { return strings.ToUpper(s[:1]) + s[1:] }

func pluralize(singular string) string {
	switch {
	case singular == "":
		return ""
	case strings.HasSuffix(singular, "y") && len(singular) > 1 && !strings.ContainsAny(singular[len(singular)-2:len(singular)-1], "aeiou"):
		return singular[:len(singular)-1] + "ies"
	case strings.HasSuffix(singular, "s"), strings.HasSuffix(singular, "x"), strings.HasSuffix(singular, "ch"), strings.HasSuffix(singular, "sh"):
		return singular + "es"
	default:
		return singular + "s"
	}
}

var omitted = regexp.MustCompile(`(?s)[ \t]*// scaffold:omit:begin\n.*?// scaffold:omit:end\n`)

// removeOmitted drops the parts of the example that only make sense there.
func removeOmitted(content string) string { return omitted.ReplaceAllString(content, "") }

func nextMigration(directory string) (int, error) {
	files, err := filepath.Glob(filepath.Join(directory, "*.sql"))
	if err != nil {
		return 0, err
	}
	sort.Strings(files)

	highest := 0
	for _, file := range files {
		var version int
		if _, err := fmt.Sscanf(filepath.Base(file), "%d_", &version); err == nil && version > highest {
			highest = version
		}
	}
	return highest + 1, nil
}

func edit(path string, change func(string) (string, error)) error {
	content, err := os.ReadFile(path) //nolint:gosec // fixed project paths.
	if err != nil {
		return err
	}

	updated, err := change(string(content))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return os.WriteFile(path, []byte(updated), 0o600)
}

// between returns the lines between two marker lines.
func between(content string, begin string, end string) (string, error) {
	start := strings.Index(content, begin)
	stop := strings.Index(content, end)
	if start < 0 || stop < start {
		return "", fmt.Errorf("markers %q and %q not found", begin, end)
	}

	start += strings.Index(content[start:], "\n") + 1
	stop = strings.LastIndex(content[:stop], "\n") + 1
	return content[start:stop], nil
}

// insertBefore inserts text before the line holding marker.
func insertBefore(content string, marker string, text string) (string, error) {
	index := strings.Index(content, marker)
	if index < 0 {
		return "", fmt.Errorf("marker %q not found", marker)
	}

	lineStart := strings.LastIndex(content[:index], "\n") + 1
	return content[:lineStart] + text + content[lineStart:], nil
}

func modulePath() (string, error) {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(content), "\n") {
		if module, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(module), nil
		}
	}
	return "", errors.New("go.mod has no module line")
}
