package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/pozeydon-code/generator-microservices-go/internal/application"
	"github.com/pozeydon-code/generator-microservices-go/internal/tui"
)

const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

var runTUIProgram = tui.Run

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return ExitUsage
	}

	switch args[0] {
	case "generate":
		return runGenerate(args[1:], stdout, stderr)
	case "tui":
		return runTUI(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		printUsage(stdout)
		return ExitOK
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return ExitUsage
	}
}

func runTUI(args []string, stdout, stderr io.Writer) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			printUsage(stdout)
			return ExitOK
		}
	}
	flags := flag.NewFlagSet("tui", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "", "Path to the microgen JSON configuration file")
	outputDir := flags.String("output", "", "Directory where generated files will be planned")
	force := flags.Bool("force", false, "Plan replacement of a verified microgen-owned generated directory")
	newConfig := flags.Bool("new", false, "Create a starter config at --config before opening the TUI")
	flags.Usage = func() { printUsage(flags.Output()) }
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return ExitOK
		}
		return ExitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(flags.Args(), " "))
		return ExitUsage
	}
	if strings.TrimSpace(*configPath) == "" {
		fmt.Fprintln(stderr, "missing required --config path")
		return ExitUsage
	}
	if strings.TrimSpace(*outputDir) == "" {
		fmt.Fprintln(stderr, "missing required --output directory")
		return ExitUsage
	}

	service, err := application.DefaultService()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return ExitError
	}
	request := application.GenerateRequest{ConfigPath: *configPath, OutputDir: *outputDir, Force: *force}
	if *newConfig {
		if _, err := service.CreateStarterConfig(*configPath); err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return ExitError
		}
		request.ConfigBootstrapped = true
	}
	plan, err := service.PlanGeneration(request)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return ExitError
	}
	if err := runTUIProgram(plan, request, service.PlanGeneration, service.Generate, service.UpdateSolutionSettings, service.UpdateServiceSettings, service.UpdateEntitySettings, service.TargetFrameworkSuggestions()); err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return ExitError
	}
	return ExitOK
}

func runGenerate(args []string, stdout, stderr io.Writer) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			printUsage(stdout)
			return ExitOK
		}
	}
	flags := flag.NewFlagSet("generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "", "Path to the microgen JSON configuration file")
	outputDir := flags.String("output", "", "Directory where generated files will be written")
	force := flags.Bool("force", false, "Replace a verified microgen-owned generated directory")
	flags.Usage = func() { printUsage(flags.Output()) }
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return ExitOK
		}
		return ExitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(flags.Args(), " "))
		return ExitUsage
	}
	if strings.TrimSpace(*configPath) == "" {
		fmt.Fprintln(stderr, "missing required --config path")
		return ExitUsage
	}
	if strings.TrimSpace(*outputDir) == "" {
		fmt.Fprintln(stderr, "missing required --output directory")
		return ExitUsage
	}

	service, err := application.DefaultService()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return ExitError
	}
	result, err := service.Generate(application.GenerateRequest{ConfigPath: *configPath, OutputDir: *outputDir, Force: *force})
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return ExitError
	}
	if result.Warning != "" {
		fmt.Fprintf(stderr, "Warning: %s\n", result.Warning)
	}

	fmt.Fprintf(stdout, "Generated %d files in %s\n", result.Plan.FileCount, result.OutputDir)
	return ExitOK
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "Usage: microgen generate --config <path> --output <dir> [--force]")
	fmt.Fprintln(writer, "       microgen tui --config <path> --output <dir> [--force] [--new]")
	fmt.Fprintln(writer, "  --new creates a starter config at --config and refuses to overwrite an existing file.")
	fmt.Fprintln(writer, "  --force replaces only a verified microgen-owned generated directory.")
}
