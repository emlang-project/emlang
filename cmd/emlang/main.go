package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/emlang-project/emlang/internal/ast"
	"github.com/emlang-project/emlang/internal/buildinfo"
	"github.com/emlang-project/emlang/internal/config"
	"github.com/emlang-project/emlang/internal/diagram"
	"github.com/emlang-project/emlang/internal/formatter"
	"github.com/emlang-project/emlang/internal/linter"
	"github.com/emlang-project/emlang/internal/parser"
	"github.com/emlang-project/emlang/internal/serve"
	"github.com/spf13/pflag"
)

const specVersion = "1.0.0"

func main() {
	args, configPath := extractConfigFlag(os.Args[1:])

	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]

	// Commands that don't need config
	switch command {
	case "init":
		cmdInit()
		return
	case "version":
		fmt.Printf("emlang version %s (spec %s)\n", buildinfo.Version, specVersion)
		return
	case "help", "-h", "--help":
		printUsage()
		return
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "parse":
		cmdParse(args[1:])
	case "lint":
		cmdLint(args[1:], cfg)
	case "fmt":
		cmdFmt(args[1:], cfg)
	case "diagram":
		cmdDiagram(args[1:], cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func extractConfigFlag(args []string) (remaining []string, configPath string) {
	for i := 0; i < len(args); i++ {
		if (args[i] == "-c" || args[i] == "--config") && i+1 < len(args) {
			configPath = args[i+1]
			i++
		} else {
			remaining = append(remaining, args[i])
		}
	}
	return
}

func printUsage() {
	fmt.Println("emlang - The Emlang toolchain (https://emlang-project.github.io/)")
	fmt.Println()
	fmt.Println("Usage: emlang [-c <config>] <command> [arguments]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -c, --config <file>  Path to config file (default: .emlang.yaml, or EMLANG_CONFIG env)")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  parse <file>         Parse a YAML source file and show structure (use - for stdin)")
	fmt.Println("  lint <file>...       Lint YAML source files for issues (use - for stdin)")
	fmt.Println("  fmt <file>...        Format YAML source files (use - for stdin, -w for in-place,")
	fmt.Println("                       --check to list unformatted files)")
	fmt.Println("                       --keys short|long: override key style")
	fmt.Println("  diagram <file>       Generate an HTML diagram (use - for stdin, -o file for output)")
	fmt.Println("                       --serve [--address 127.0.0.1] [--port 8274]: live-reload server")
	fmt.Println("  init                 Create a .emlang.yaml config file with defaults")
	fmt.Println("  version              Print version information")
	fmt.Println("  help                 Show this help message")
}

const defaultConfig = `# emlang configuration file
# Documentation: https://emlang-project.github.io/

lint:
  # ignore:
  #   - command-without-event
  #   - orphan-exception
  #   - slice-missing-event
  #   - trigger-without-command
  #   - unknown-test-element
  #   - view-without-event

fmt:
  # keys: short

diagram:
  # serve:
  #   address: 127.0.0.1
  #   port: 8274

  # css:
  #   --diagram-bg-color: "#ffffff"
  #   --diagram-text-color: "#212529"
  #   --diagram-border-color: "#ced4da"
  #
  #   --trigger-bg-color: "#e9ecef"
  #   --command-bg-color: "#a5d8ff"
  #   --event-bg-color: "#ffd8a8"
  #   --exception-bg-color: "#ffc9c9"
  #   --view-bg-color: "#b2f2bb"
  #   --item-border-radius: 0.5em
  #   --item-width: auto
  #   --item-min-width: 8em
  #   --item-max-width: none
  #
  #   --diagram-font-family: system-ui
  #   --props-font-family: monospace
  #
  #   --slicename-font-size: 2em
  #   --slicename-font-weight: normal
  #   --swimlane-font-size: 1.5em
  #   --swimlane-font-weight: normal
  #   --testname-font-size: 1em
  #   --testname-font-weight: bold
  #   --label-font-size: 0.75em
  #   --label-font-weight: normal
  #   --props-font-size: 0.75em
  #   --props-font-weight: normal
`

func cmdInit() {
	const path = ".emlang.yaml"
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(os.Stderr, "Error: %s already exists\n", path)
		os.Exit(1)
	}
	if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("Created %s\n", path)
}

func parseFile(arg string) (*ast.Document, string) {
	doc, name, err := readDocument(arg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return doc, name
}

// readDocument reads and parses arg, a file path or "-" for stdin. It
// returns the name to show in messages along with the document.
func readDocument(arg string) (*ast.Document, string, error) {
	var input io.Reader
	var name string

	if arg == "-" {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, "<stdin>", fmt.Errorf("Error reading input: %v", err)
		}
		input = bytes.NewReader(content)
		name = "<stdin>"
	} else {
		f, err := os.Open(arg)
		if err != nil {
			return nil, arg, fmt.Errorf("Error reading input: %v", err)
		}
		defer f.Close()
		input = f
		name = arg
	}

	doc, err := parser.Parse(input)
	if err != nil {
		return nil, name, fmt.Errorf("Parse error in %s: %v", name, err)
	}

	return doc, name, nil
}

func cmdParse(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: emlang parse <file>")
		os.Exit(1)
	}

	doc, name := parseFile(args[0])

	fmt.Printf("Parsed %s successfully\n", name)
	fmt.Println("----------------------------------------")
	printDocument(doc)
}

func printDocument(doc *ast.Document) {
	fmt.Printf("Document with %d slice(s)\n", len(doc.Slices))

	for _, sd := range doc.SubDocs {
		for _, name := range sd.SliceOrder {
			fmt.Println()
			printSlice(name, sd.Slices[name])
		}
	}
}

func printSlice(name string, slice *ast.Slice) {
	displayName := name
	if displayName == "" {
		displayName = "(anonymous)"
	}
	fmt.Printf("Slice: %s\n", displayName)
	fmt.Printf("  %d element(s)\n", len(slice.Elements))
	for _, elem := range slice.Elements {
		printElement("    ", elem)
	}

	if len(slice.Tests) > 0 {
		fmt.Printf("  %d attached test(s)\n", len(slice.Tests))
		for testName, test := range slice.Tests {
			printTest("  "+testName, test)
		}
	}
}

func printTest(name string, test *ast.Test) {
	fmt.Printf("Test: %s\n", name)

	if len(test.Given) > 0 {
		fmt.Printf("  Given: %d element(s)\n", len(test.Given))
		for _, elem := range test.Given {
			printElement("    ", elem)
		}
	}

	if len(test.When) > 0 {
		fmt.Printf("  When: %d element(s)\n", len(test.When))
		for _, elem := range test.When {
			printElement("    ", elem)
		}
	}

	if len(test.Then) > 0 {
		fmt.Printf("  Then: %d element(s)\n", len(test.Then))
		for _, elem := range test.Then {
			printElement("    ", elem)
		}
	}
}

func printElement(indent string, elem *ast.Element) {
	swimlane := ""
	if elem.Swimlane != "" {
		swimlane = elem.Swimlane + "/"
	}

	fmt.Printf("%s%s: %s%s\n", indent, elem.Type, swimlane, elem.Name)

	if len(elem.Props) > 0 {
		fmt.Printf("%s  props:\n", indent)
		for _, p := range elem.Props {
			fmt.Printf("%s    %s: %v\n", indent, p.Key, p.Value)
		}
	}
}

// cmdFmt formats one file to stdout, or several files in place (-w) or as
// a check (--check). With several files, it keeps going after a file that
// cannot be read, parsed or written, so that one run reports every file.
func cmdFmt(args []string, cfg *config.Config) {
	flags := pflag.NewFlagSet("fmt", pflag.ExitOnError)
	writeFlag := flags.BoolP("write", "w", false, "write result to source files instead of stdout")
	checkFlag := flags.Bool("check", false, "list files that are not formatted, without changing them")
	keysFlag := flags.String("keys", "", "key style: short or long")
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: emlang fmt [-w | --check] [--keys short|long] <file>...")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	if flags.NArg() < 1 {
		flags.Usage()
		os.Exit(1)
	}
	files := flags.Args()

	if *writeFlag && *checkFlag {
		fmt.Fprintln(os.Stderr, "Error: -w and --check cannot be used together")
		os.Exit(1)
	}
	if !*writeFlag && !*checkFlag && len(files) > 1 {
		fmt.Fprintln(os.Stderr, "Error: several files need -w or --check")
		os.Exit(1)
	}
	for _, file := range files {
		if *writeFlag && file == "-" {
			fmt.Fprintln(os.Stderr, "Error: -w cannot be used with stdin")
			os.Exit(1)
		}
		if file == "-" && len(files) > 1 {
			fmt.Fprintln(os.Stderr, "Error: - (stdin) cannot be used with other files")
			os.Exit(1)
		}
	}

	// Priority: flag > config > formatter default
	keyStyle := cfg.Fmt.Keys
	if flags.Changed("keys") {
		keyStyle = *keysFlag
	}
	opts := formatter.Options{KeyStyle: keyStyle}

	failed := false
	for _, file := range files {
		doc, name, err := readDocument(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed = true
			continue
		}

		out, err := formatter.Format(doc, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Format error in %s: %v\n", name, err)
			failed = true
			continue
		}

		switch {
		case *checkFlag:
			if !bytes.Equal(out, doc.RawSource) {
				fmt.Println(name)
				failed = true
			}
		case *writeFlag:
			// A file already formatted is not rewritten, so its
			// modification time stays as it was.
			if bytes.Equal(out, doc.RawSource) {
				continue
			}
			if err := os.WriteFile(file, out, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", file, err)
				failed = true
			}
		default:
			os.Stdout.Write(out)
		}
	}

	if failed {
		os.Exit(1)
	}
}

func cmdDiagram(args []string, cfg *config.Config) {
	flags := pflag.NewFlagSet("diagram", pflag.ExitOnError)
	outputFile := flags.StringP("output", "o", "", "output file")
	serveFlag := flags.Bool("serve", false, "start a live-reload HTTP server")
	portFlag := flags.Int("port", 0, "port for the live-reload server")
	addressFlag := flags.String("address", "", "listen address for the live-reload server")
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: emlang diagram [-o output.html] [--serve [--address 127.0.0.1] [--port 8274]] <file>")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	if flags.NArg() < 1 {
		flags.Usage()
		os.Exit(1)
	}

	if *serveFlag && *outputFile != "" {
		fmt.Fprintln(os.Stderr, "Error: --serve and -o are mutually exclusive")
		os.Exit(1)
	}

	inputArg := flags.Arg(0)

	warnRenamedCSSVariables(cfg.Diagram.CSS)

	if *serveFlag {
		if inputArg == "-" {
			fmt.Fprintln(os.Stderr, "Error: --serve cannot be used with stdin")
			os.Exit(1)
		}

		// Priority: flag > config > default
		addr := "127.0.0.1"
		if cfg.Diagram.Serve.Address != "" {
			addr = cfg.Diagram.Serve.Address
		}
		if flags.Changed("address") {
			addr = *addressFlag
		}

		port := 8274
		if cfg.Diagram.Serve.Port != 0 {
			port = cfg.Diagram.Serve.Port
		}
		if flags.Changed("port") {
			port = *portFlag
		}

		if err := serve.Start(inputArg, addr, port, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	doc, _ := parseFile(inputArg)

	gen := diagram.New()
	gen.CSSOverrides = cfg.Diagram.CSS
	html, err := gen.Generate(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Diagram generation error: %v\n", err)
		os.Exit(1)
	}

	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, html, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	} else {
		os.Stdout.Write(html)
	}
}

// warnRenamedCSSVariables prints a warning for each deprecated CSS variable
// name in the config, with the name to use instead.
func warnRenamedCSSVariables(css map[string]string) {
	names := make([]string, 0, len(css))
	for name := range css {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if current, ok := diagram.RenamedCSSVariable(name); ok {
			fmt.Fprintf(os.Stderr, "Warning: diagram.css: %s is deprecated, use %s\n", name, current)
		}
	}
}

// cmdLint lints each file on its own and keeps going after a file that
// cannot be read or parsed, so that one run reports every file.
func cmdLint(args []string, cfg *config.Config) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: emlang lint <file>...")
		os.Exit(1)
	}

	stdinCount := 0
	for _, arg := range args {
		if arg == "-" {
			stdinCount++
		}
	}
	if stdinCount > 1 {
		fmt.Fprintln(os.Stderr, "Error: - (stdin) can be given only once")
		os.Exit(1)
	}

	totalErrors, totalWarnings := 0, 0
	for _, arg := range args {
		errors, warnings := lintFile(arg, cfg)
		totalErrors += errors
		totalWarnings += warnings
	}

	if len(args) > 1 {
		fmt.Printf("Total: %d file(s), %d error(s), %d warning(s)\n", len(args), totalErrors, totalWarnings)
	}

	if totalErrors > 0 {
		os.Exit(1)
	}
}

// lintFile lints one file and prints its report. A file that cannot be
// read or parsed counts as one error.
func lintFile(arg string, cfg *config.Config) (errorCount, warningCount int) {
	doc, name, err := readDocument(arg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1, 0
	}

	lint := linter.New()
	for _, rule := range cfg.Lint.Ignore {
		lint.IgnoreRules[rule] = true
	}
	issues := lint.Lint(doc)

	if len(issues) == 0 {
		fmt.Printf("%s: OK (no issues found)\n", name)
		return 0, 0
	}

	for _, issue := range issues {
		if issue.Severity == linter.SeverityError {
			errorCount++
		} else {
			warningCount++
		}
	}

	fmt.Printf("%s: %d issue(s) found\n", name, len(issues))
	fmt.Println("----------------------------------------")

	for _, issue := range issues {
		severity := "warning"
		if issue.Severity == linter.SeverityError {
			severity = "error"
		}
		fmt.Printf("%s:%d:%d: %s: %s [%s]\n",
			name, issue.Line, issue.Column, severity, issue.Message, issue.Rule)
	}

	fmt.Println("----------------------------------------")
	fmt.Printf("Summary: %d error(s), %d warning(s)\n", errorCount, warningCount)

	return errorCount, warningCount
}
