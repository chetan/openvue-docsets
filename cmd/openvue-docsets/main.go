package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"git.bixby.io/chetan/openvue-docsets/internal/docset"
	"github.com/spf13/cobra"
)

var version = "dev"

func newCommand() *cobra.Command {
	root := &cobra.Command{Use: "openvue-docsets", Short: "Build offline OpenVue and OpenIcons Dash docsets", Version: version, Args: cobra.NoArgs, SilenceUsage: true, SilenceErrors: true}
	root.RunE = func(cmd *cobra.Command, _ []string) error { return cmd.Help() }
	root.SetVersionTemplate("{{.Version}}\n")
	root.AddCommand(&cobra.Command{Use: "version", Short: "Print the build version", Args: cobra.NoArgs, Run: func(cmd *cobra.Command, _ []string) { fmt.Fprintln(cmd.OutOrStdout(), version) }})
	var cfg docset.Config
	build := &cobra.Command{Use: "build", Short: "Build both docsets from local upstream checkouts", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return docset.Build(cmd.Context(), cfg, cmd.OutOrStdout())
	}}
	build.Flags().StringVar(&cfg.OpenVue, "openvue", ".cache/openvue", "OpenVue source checkout")
	build.Flags().StringVar(&cfg.OpenIcons, "icons", ".cache/openicons", "OpenIcons source checkout")
	build.Flags().StringVar(&cfg.Output, "output", "dist", "Output parent (existing docsets are not overwritten)")
	build.Flags().StringVar(&cfg.Examples, "examples", ".cache/examples", "Compiled offline Vue examples (empty for a reference-only build)")
	root.AddCommand(build)
	return root
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := newCommand().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
