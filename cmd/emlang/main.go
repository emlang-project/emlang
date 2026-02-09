package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/emlang-project/emlang/internal/ast"
	"github.com/emlang-project/emlang/internal/config"
	"github.com/emlang-project/emlang/internal/diagram"
	"github.com/emlang-project/emlang/internal/formatter"
	"github.com/emlang-project/emlang/internal/linter"
	"github.com/emlang-project/emlang/internal/parser"
	"github.com/emlang-project/emlang/internal/serve"
	"github.com/spf13/pflag"
)

const version = "1.0.0"
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
		fmt.Printf("emlang version %s (spec %s)\n", version, specVersion)
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
	case "repl":
		cmdRepl(args[1:], cfg)
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
	fmt.Println("  lint <file>          Lint a YAML source file for issues (use - for stdin)")
	fmt.Println("  fmt <file>           Format a YAML source file (use - for stdin, -w for in-place)")
	fmt.Println("                       --keys short|long: override key style")
	fmt.Println("  repl [file]          Start an interactive REPL with live diagram preview")
	fmt.Println("                       --address, --port: server options")
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

fmt:
  # keys: long

repl:
  # address: 127.0.0.1
  # port: 8275

diagram:
  # serve:
  #   address: 127.0.0.1
  #   port: 8274

  # css:
  #   --text-color: "#212529"
  #   --border-color: "#ced4da"
  #
  #   --trigger-color: "#e9ecef"
  #   --command-color: "#a5d8ff"
  #   --event-color: "#ffd8a8"
  #   --exception-color: "#ffc9c9"
  #   --view-color: "#b2f2bb"
  #   --item-border-radius: 0.5em
  #
  #   --font-family-normal: system-ui
  #   --font-family-props: monospace
  #
  #   --font-size-slicename: 2em
  #   --font-weight-slicename: normal
  #   --font-size-swimlane: 1.5em
  #   --font-weight-swimlane: normal
  #   --font-size-testname: 1em
  #   --font-weight-testname: bold
  #   --font-size-label: 0.75em
  #   --font-weight-label: normal
  #   --font-size-props: 0.75em
  #   --font-weight-props: normal
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
	var input io.Reader
	var name string

	if arg == "-" {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}
		input = bytes.NewReader(content)
		name = "<stdin>"
	} else {
		f, err := os.Open(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		input = f
		name = arg
	}

	doc, err := parser.Parse(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error in %s: %v\n", name, err)
		os.Exit(1)
	}

	return doc, name
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

func cmdFmt(args []string, cfg *config.Config) {
	flags := pflag.NewFlagSet("fmt", pflag.ExitOnError)
	writeFlag := flags.BoolP("write", "w", false, "write result to source file instead of stdout")
	keysFlag := flags.String("keys", "", "key style: short or long")
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: emlang fmt [-w] [--keys short|long] <file>")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	if flags.NArg() < 1 {
		flags.Usage()
		os.Exit(1)
	}

	inputArg := flags.Arg(0)

	if *writeFlag && inputArg == "-" {
		fmt.Fprintln(os.Stderr, "Error: -w cannot be used with stdin")
		os.Exit(1)
	}

	doc, _ := parseFile(inputArg)

	// Priority: flag > config > default
	keyStyle := "long"
	if cfg.Fmt.Keys != "" {
		keyStyle = cfg.Fmt.Keys
	}
	if flags.Changed("keys") {
		keyStyle = *keysFlag
	}

	out := formatter.Format(doc, formatter.Options{KeyStyle: keyStyle})

	if *writeFlag {
		if err := os.WriteFile(inputArg, out, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", inputArg, err)
			os.Exit(1)
		}
	} else {
		os.Stdout.Write(out)
	}
}

func cmdRepl(args []string, cfg *config.Config) {
	flags := pflag.NewFlagSet("repl", pflag.ExitOnError)
	portFlag := flags.Int("port", 0, "port for the REPL server")
	addressFlag := flags.String("address", "", "listen address for the REPL server")
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: emlang repl [--address 127.0.0.1] [--port 8275] [file]")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	var filePath string
	if flags.NArg() > 0 {
		filePath = flags.Arg(0)
	}

	// Priority: flag > config > default
	addr := "127.0.0.1"
	if cfg.Repl.Address != "" {
		addr = cfg.Repl.Address
	}
	if flags.Changed("address") {
		addr = *addressFlag
	}

	port := 8275
	if cfg.Repl.Port != 0 {
		port = cfg.Repl.Port
	}
	if flags.Changed("port") {
		port = *portFlag
	}

	if err := serve.StartRepl(filePath, addr, port, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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

func cmdLint(args []string, cfg *config.Config) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: emlang lint <file>")
		os.Exit(1)
	}

	doc, name := parseFile(args[0])

	lint := linter.New()
	for _, rule := range cfg.Lint.Ignore {
		lint.IgnoreRules[rule] = true
	}
	issues := lint.Lint(doc)

	if len(issues) == 0 {
		fmt.Printf("%s: OK (no issues found)\n", name)
		return
	}

	errorCount := 0
	warningCount := 0
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

	if errorCount > 0 {
		os.Exit(1)
	}
}
