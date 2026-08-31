package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/KoukeNeko/taiga-cli/internal/config"
	"github.com/KoukeNeko/taiga-cli/internal/credential"
	"github.com/KoukeNeko/taiga-cli/internal/output"
	"github.com/KoukeNeko/taiga-cli/internal/taiga"
	"github.com/spf13/cobra"
)

type App struct {
	In          io.Reader
	Out         io.Writer
	Err         io.Writer
	HTTPClient  *http.Client
	Config      *config.Store
	GitLocal    *config.GitLocal
	Credentials credential.Store
	Getenv      func(string) string
	Cwd         string

	global globalOptions
}

type globalOptions struct {
	Profile string
	APIURL  string
	Project string
	JSON    bool
	Fields  []string
	NoInput bool
	NoColor bool
	Quiet   bool
	Verbose bool
}

type Settings struct {
	Profile      string `json:"profile"`
	APIURL       string `json:"api_url"`
	Project      string `json:"project,omitempty"`
	Token        string `json:"-"`
	RefreshToken string `json:"-"`
}

func New() (*App, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current directory: %w", err)
	}
	return &App{
		In:          os.Stdin,
		Out:         os.Stdout,
		Err:         os.Stderr,
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
		Config:      config.NewStore(path),
		GitLocal:    config.NewGitLocal(cwd),
		Credentials: credential.NewKeyringStore(),
		Getenv:      os.Getenv,
		Cwd:         cwd,
	}, nil
}

func (a *App) Execute(ctx context.Context, args []string) int {
	root := a.rootCommand()
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	if err == nil {
		return ExitSuccess
	}
	known, body := classifyError(err)
	renderer := a.renderer()
	_ = renderer.Failure(body)
	return known.ExitCode
}

func (a *App) renderer() output.Renderer {
	return output.Renderer{Out: a.Out, Err: a.Err, JSON: a.global.JSON, Fields: a.global.Fields, Quiet: a.global.Quiet}
}

func (a *App) rootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "taiga",
		Short:         "Manage Taiga projects from the command line",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if len(a.global.Fields) > 0 && !a.global.JSON {
				return usageError("--fields requires --json")
			}
			if a.Getenv("NO_COLOR") != "" {
				a.global.NoColor = true
			}
			return nil
		},
	}
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError(err.Error()) })
	root.SetOut(a.Out)
	root.SetErr(a.Err)
	flags := root.PersistentFlags()
	flags.StringVar(&a.global.Profile, "profile", "", "Taiga profile to use")
	flags.StringVar(&a.global.APIURL, "api-url", "", "complete Taiga API base URL")
	flags.StringVarP(&a.global.Project, "project", "p", "", "Taiga project slug")
	flags.BoolVar(&a.global.JSON, "json", false, "emit the versioned JSON contract")
	flags.StringSliceVar(&a.global.Fields, "fields", nil, "comma-separated JSON fields to include")
	flags.BoolVar(&a.global.NoInput, "no-input", false, "never prompt for input")
	flags.BoolVar(&a.global.NoColor, "no-color", false, "disable color output")
	flags.BoolVarP(&a.global.Quiet, "quiet", "q", false, "suppress non-essential human output")
	flags.BoolVarP(&a.global.Verbose, "verbose", "v", false, "print redacted HTTP diagnostics to stderr")
	_ = root.RegisterFlagCompletionFunc("profile", a.completeProfiles)
	_ = root.RegisterFlagCompletionFunc("project", a.completeProjects)
	root.AddCommand(
		a.versionCommand(),
		a.doctorCommand(),
		a.authCommand(),
		a.configCommand(),
		a.projectCommand(),
		a.issueCommand(),
		a.storyCommand(),
		a.taskCommand(),
		a.sprintCommand(),
		a.wikiCommand(),
		a.searchCommand(),
		a.attachmentCommand(),
		a.schemaCommand(),
		a.completionCommand(root),
	)
	return root
}

func (a *App) resolveSettings(ctx context.Context) (Settings, config.File, error) {
	cfg, err := a.Config.Load()
	if err != nil {
		return Settings{}, config.File{}, err
	}
	local := config.LocalValues{}
	if a.GitLocal != nil {
		values, localErr := a.GitLocal.Load(ctx)
		if localErr == nil {
			local = values
		} else if !errors.Is(localErr, config.ErrNotGitRepository) {
			return Settings{}, config.File{}, localErr
		}
	}
	profileName := firstNonEmpty(a.global.Profile, a.Getenv("TAIGA_PROFILE"), local.Profile, cfg.CurrentProfile, config.DefaultProfileName())
	profileName, err = config.NormalizeProfileName(profileName)
	if err != nil {
		return Settings{}, config.File{}, validationError("invalid_profile", err.Error())
	}
	profile := cfg.Profiles[profileName]
	apiURL := firstNonEmpty(a.global.APIURL, a.Getenv("TAIGA_API_URL"), profile.APIURL)
	if apiURL != "" {
		apiURL, err = taiga.NormalizeAPIURL(apiURL)
		if err != nil {
			return Settings{}, config.File{}, validationError("invalid_api_url", err.Error())
		}
	}
	project := firstNonEmpty(a.global.Project, a.Getenv("TAIGA_PROJECT"), local.Project, profile.Project)
	token := strings.TrimSpace(a.Getenv("TAIGA_TOKEN"))
	refreshToken := ""
	if token == "" && apiURL != "" && a.Credentials != nil {
		tokens, credentialErr := a.Credentials.Get(credential.Account(profileName, apiURL))
		if credentialErr == nil {
			token = tokens.AuthToken
			refreshToken = tokens.RefreshToken
		} else if !errors.Is(credentialErr, credential.ErrNotFound) {
			return Settings{}, config.File{}, credentialErr
		}
	}
	return Settings{Profile: profileName, APIURL: apiURL, Project: project, Token: token, RefreshToken: refreshToken}, cfg, nil
}

func (a *App) client(ctx context.Context, requireToken bool) (*taiga.Client, Settings, error) {
	settings, _, err := a.resolveSettings(ctx)
	if err != nil {
		return nil, Settings{}, err
	}
	if settings.APIURL == "" {
		return nil, Settings{}, validationError("missing_api_url", "no Taiga API URL configured; run `taiga auth login --host <url>` or pass --api-url")
	}
	if requireToken && settings.Token == "" {
		return nil, Settings{}, authRequired("no Taiga credential available; run `taiga auth login` or set TAIGA_TOKEN")
	}
	options := []taiga.ClientOption{taiga.WithHTTPClient(a.HTTPClient), taiga.WithToken(settings.Token)}
	if settings.RefreshToken != "" && a.Credentials != nil {
		account := credential.Account(settings.Profile, settings.APIURL)
		options = append(options, taiga.WithRefreshToken(settings.RefreshToken, func(authToken, refreshToken string) error {
			return a.Credentials.Set(account, credential.Tokens{AuthToken: authToken, RefreshToken: refreshToken})
		}))
	}
	if a.global.Verbose {
		options = append(options, taiga.WithVerbose(a.Err))
	}
	client, err := taiga.NewClient(settings.APIURL, options...)
	return client, settings, err
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
