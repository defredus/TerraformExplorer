package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/terraform-explorer/terraform-explorer/internal/app"
	"github.com/terraform-explorer/terraform-explorer/internal/discovery"
	"github.com/terraform-explorer/terraform-explorer/internal/httpapi"
	"github.com/terraform-explorer/terraform-explorer/internal/webui"
)

const usageText = `Terraform Explorer inspects Terraform configuration without executing it.

Usage:
  terraform-explorer scan [--pretty=true] <directory>
  terraform-explorer serve [--listen=127.0.0.1:8787]
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return 2
	}

	switch args[0] {
	case "scan":
		return runScan(args[1:], stdout, stderr)
	case "serve":
		return runServe(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usageText)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", args[0], usageText)
		return 2
	}
}

func runServe(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	listen := flags.String("listen", "127.0.0.1:8787", "HTTP listen address")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "serve does not accept positional arguments")
		return 2
	}

	handler := httpapi.New(app.NewStore(), webui.Assets())
	server := &http.Server{
		Addr:              *listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	fmt.Fprintf(stdout, "Terraform Explorer is running at http://%s\n", *listen)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(stderr, "serve: %v\n", err)
		return 1
	}
	return 0
}

func runScan(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("scan", flag.ContinueOnError)
	flags.SetOutput(stderr)
	pretty := flags.Bool("pretty", true, "indent JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprintln(stderr, "scan requires exactly one directory")
		return 2
	}

	result, err := discovery.ScanDir(flags.Arg(0))
	if err != nil {
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			fmt.Fprintf(stderr, "scan: %s: %v\n", pathErr.Path, pathErr.Err)
		} else {
			fmt.Fprintf(stderr, "scan: %v\n", err)
		}
		return 1
	}

	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if *pretty {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(stderr, "encode result: %v\n", err)
		return 1
	}

	return 0
}
