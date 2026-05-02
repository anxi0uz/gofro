package cmd

import (
	"fmt"
	"os"

	"github.com/anxi0uz/gofro/internal/gen"
	"github.com/spf13/cobra"
)

var (
	flagPostgres   bool
	flagRedis      bool
	flagGrafana    bool
	flagPrometheus bool
	flagGit        bool
	flagGithub     string
	flagModule     string
)

var newCmd = &cobra.Command{
	Use:   "new <project-name>",
	Short: "Create a new Go project from template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]

		if flagGrafana && !flagPrometheus {
			fmt.Println("note: --grafana requires --prometheus, enabling automatically")
			flagPrometheus = true
		}

		modulePath := projectName
		if flagModule != "" {
			modulePath = flagModule
		} else if flagGithub != "" {
			modulePath = fmt.Sprintf("github.com/%s/%s", flagGithub, projectName)
		}

		opts := gen.Options{
			ProjectName: projectName,
			ModulePath:  modulePath,
			Postgres:    flagPostgres,
			Redis:       flagRedis,
			Grafana:     flagGrafana,
			Prometheus:  flagPrometheus,
			Git:         flagGit,
		}

		if err := gen.Generate(opts); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	newCmd.Flags().BoolVar(&flagPostgres, "postgres", false, "add PostgreSQL to compose and project")
	newCmd.Flags().BoolVar(&flagRedis, "redis", false, "add Redis to compose and project")
	newCmd.Flags().BoolVar(&flagGrafana, "grafana", false, "add Grafana to compose (implies --prometheus)")
	newCmd.Flags().BoolVar(&flagPrometheus, "prometheus", false, "add Prometheus to compose and config")
	newCmd.Flags().BoolVar(&flagGit, "git", false, "run git init in the generated project")
	newCmd.Flags().StringVar(&flagGithub, "github", "", "GitHub username — sets module path to github.com/<user>/<project>")
	newCmd.Flags().StringVar(&flagModule, "module", "", "full module path for go mod init (overrides --github)")
}
