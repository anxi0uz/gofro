package gen

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

//go:embed tmpl/*
var tmplFS embed.FS

type Options struct {
	ProjectName string
	ModulePath  string
	Postgres    bool
	Redis       bool
	Grafana     bool
	Prometheus  bool
	Git         bool
}

func Generate(opts Options) error {
	if _, err := os.Stat(opts.ProjectName); err == nil {
		return fmt.Errorf("directory %q already exists", opts.ProjectName)
	}

	fmt.Printf("  creating %s/\n", opts.ProjectName)

	dirs := []string{
		"cmd",
		"configs",
		"internal/config",
		"internal/api",
		"internal/handler",
	}
	if opts.Postgres || opts.Redis {
		dirs = append(dirs, "internal/database")
	}
	if opts.Postgres {
		dirs = append(dirs, "pkg/storage", "migrations")
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(opts.ProjectName, dir), 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	type fileSpec struct {
		tmpl string
		dest string
		cond bool
	}

	files := []fileSpec{
		{"main_go.tmpl", "cmd/main.go", true},
		{"config_go.tmpl", "internal/config/config.go", true},
		{"config_toml.tmpl", "configs/config.toml", true},
		{"dockerfile.tmpl", "Dockerfile", true},
		{"compose_yml.tmpl", "docker-compose.yml", true},
		{"env.tmpl", ".env", true},
		{"gitignore.tmpl", ".gitignore", true},
		{"makefile.tmpl", "Makefile", true},
		{"handler_server_impl_go.tmpl", "internal/handler/server_impl.go", true},
		{"api_swagger_yaml.tmpl", "internal/api/api.swagger.yaml", true},
		{"api_gen_go.tmpl", "internal/api/gen.go", true},
		{"api_oapi_config_yaml.tmpl", "internal/api/oapi-codegen.yaml", true},
		{"postgres_go.tmpl", "internal/database/postgres.go", opts.Postgres},
		{"redis_go.tmpl", "internal/database/redis.go", opts.Redis},
		{"storage_go.tmpl", "pkg/storage/storage.go", opts.Postgres},
		{"prometheus_yml.tmpl", "configs/prometheus.yml", opts.Prometheus},
	}

	for _, f := range files {
		if !f.cond {
			continue
		}
		dest := filepath.Join(opts.ProjectName, f.dest)
		if err := generateFile(opts, f.tmpl, dest); err != nil {
			return fmt.Errorf("generate %s: %w", f.dest, err)
		}
		fmt.Printf("  wrote   %s\n", f.dest)
	}

	fmt.Println("\n  running go mod init...")
	if err := runInDir(opts.ProjectName, "go", "mod", "init", opts.ModulePath); err != nil {
		return fmt.Errorf("go mod init: %w", err)
	}

	fmt.Println("  running go mod tidy...")
	if err := runInDir(opts.ProjectName, "go", "mod", "tidy"); err != nil {
		fmt.Printf("  warning: go mod tidy failed: %v\n", err)
		fmt.Println("  run 'go mod tidy' manually after network is available")
	}

	if opts.Git {
		fmt.Println("  running git init...")
		if err := runInDir(opts.ProjectName, "git", "init"); err != nil {
			return fmt.Errorf("git init: %w", err)
		}
	}

	fmt.Printf("\ndone! next steps:\n")
	fmt.Printf("  cd %s\n", opts.ProjectName)
	if !opts.Git {
		fmt.Println("  git init && git add . && git commit -m 'initial'")
	}
	fmt.Println("  docker compose up -d")
	fmt.Println("  go run ./cmd/")

	return nil
}

func generateFile(opts Options, tmplName, destPath string) error {
	content, err := tmplFS.ReadFile("tmpl/" + tmplName)
	if err != nil {
		return err
	}

	t, err := template.New(tmplName).Parse(string(content))
	if err != nil {
		return err
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, opts)
}

func runInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
