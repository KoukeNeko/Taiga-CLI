package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/KoukeNeko/aihki/internal/config"
	"github.com/KoukeNeko/aihki/internal/credential"
	"github.com/KoukeNeko/aihki/internal/taiga"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func (a *App) authCommand() *cobra.Command {
	command := &cobra.Command{Use: "auth", Short: "Authenticate with Taiga"}
	command.AddCommand(a.authLoginCommand(), a.authLogoutCommand(), a.authStatusCommand())
	return command
}

func (a *App) authLoginCommand() *cobra.Command {
	var host, username string
	var withToken bool
	command := &cobra.Command{
		Use:   "login",
		Short: "Authenticate and save a Taiga credential",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if host != "" && a.global.APIURL != "" {
				return usageError("--host and --api-url are mutually exclusive")
			}
			settings, cfg, err := a.resolveSettings(ctx)
			if err != nil {
				return err
			}
			apiURL := a.global.APIURL
			if host != "" {
				front, err := taiga.DiscoverAPI(ctx, a.HTTPClient, host)
				if err != nil {
					return err
				}
				apiURL = front.API
			}
			if apiURL == "" {
				apiURL = settings.APIURL
			}
			if apiURL == "" {
				return validationError("missing_api_url", "provide --host or --api-url for the first login")
			}
			apiURL, err = taiga.NormalizeAPIURL(apiURL)
			if err != nil {
				return validationError("invalid_api_url", err.Error())
			}
			clientOptions := []taiga.ClientOption{taiga.WithHTTPClient(a.HTTPClient)}
			if a.global.Verbose {
				clientOptions = append(clientOptions, taiga.WithVerbose(a.Err))
			}
			client, err := taiga.NewClient(apiURL, clientOptions...)
			if err != nil {
				return err
			}
			var tokens credential.Tokens
			var user taiga.User
			if withToken {
				data, err := io.ReadAll(io.LimitReader(a.In, 64<<10))
				if err != nil {
					return fmt.Errorf("read token from stdin: %w", err)
				}
				tokens.AuthToken = strings.TrimSpace(string(data))
				if tokens.AuthToken == "" {
					return validationError("empty_token", "--with-token requires a token on stdin")
				}
				client.SetToken(tokens.AuthToken)
				user, err = client.Me(ctx)
				if err != nil {
					return err
				}
			} else {
				if a.global.NoInput || !a.stdinTTY() {
					return validationError("input_required", "interactive login requires a TTY; use --with-token or AIHKI_TOKEN for automation")
				}
				if username == "" {
					username, err = a.readLine("Username: ")
					if err != nil {
						return err
					}
				}
				password, err := a.readPassword("Password: ")
				if err != nil {
					return err
				}
				response, err := client.Login(ctx, username, password)
				if err != nil {
					return err
				}
				tokens = credential.Tokens{AuthToken: response.AuthToken, RefreshToken: response.RefreshToken}
				user = taiga.User{ID: response.ID, Username: response.Username, FullName: response.FullName}
			}
			updateProfile(&cfg, settings.Profile, func(profile *config.Profile) { profile.APIURL = apiURL })
			cfg.CurrentProfile = settings.Profile
			if err := a.Config.Save(cfg); err != nil {
				return err
			}
			if err := a.Credentials.Set(credential.Account(settings.Profile, apiURL), tokens); err != nil {
				return err
			}
			result := map[string]any{"profile": settings.Profile, "api_url": apiURL, "user": user}
			if a.global.JSON {
				return a.renderer().Data(result)
			}
			if !a.global.Quiet {
				_, _ = fmt.Fprintf(a.Out, "Logged in to %s as %s (profile %s)\n", apiURL, user.Username, settings.Profile)
			}
			return nil
		},
	}
	command.Flags().StringVar(&host, "host", "", "address of the Taiga web app or its API")
	command.Flags().StringVarP(&username, "username", "u", "", "Taiga username or email")
	command.Flags().BoolVar(&withToken, "with-token", false, "read a bearer token from stdin")
	return command
}

func (a *App) authLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the saved credential for the current profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			settings, _, err := a.resolveSettings(cmd.Context())
			if err != nil {
				return err
			}
			if settings.APIURL == "" {
				return validationError("missing_api_url", "current profile has no API URL")
			}
			if err := a.Credentials.Delete(credential.Account(settings.Profile, settings.APIURL)); err != nil {
				return err
			}
			result := map[string]any{"profile": settings.Profile, "logged_out": true}
			if a.global.JSON {
				return a.renderer().Data(result)
			}
			if !a.global.Quiet {
				_, _ = fmt.Fprintf(a.Out, "Logged out profile %s\n", settings.Profile)
			}
			return nil
		},
	}
}

func (a *App) authStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, settings, err := a.client(cmd.Context(), true)
			if err != nil {
				return err
			}
			user, err := client.Me(cmd.Context())
			if err != nil {
				return err
			}
			result := map[string]any{"profile": settings.Profile, "api_url": settings.APIURL, "project": settings.Project, "user": user, "authenticated": true}
			if a.global.JSON {
				return a.renderer().Data(result)
			}
			_, _ = fmt.Fprintf(a.Out, "Authenticated to %s as %s (profile %s)\n", settings.APIURL, user.Username, settings.Profile)
			return nil
		},
	}
}

func (a *App) stdinTTY() bool {
	file, ok := a.In.(*os.File)
	// x/term takes an int, and a descriptor is small and non-negative.
	return ok && term.IsTerminal(int(file.Fd())) // #nosec G115
}

func (a *App) readLine(prompt string) (string, error) {
	_, _ = fmt.Fprint(a.Err, prompt)
	line, err := bufio.NewReader(a.In).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read input: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", validationError("empty_input", "input cannot be empty")
	}
	return line, nil
}

func (a *App) readPassword(prompt string) (string, error) {
	file, ok := a.In.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) { // #nosec G115 -- see stdinTTY
		return "", validationError("input_required", "password input requires a TTY")
	}
	_, _ = fmt.Fprint(a.Err, prompt)
	data, err := term.ReadPassword(int(file.Fd())) // #nosec G115 -- see stdinTTY
	_, _ = fmt.Fprintln(a.Err)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	password := strings.TrimSpace(string(data))
	if password == "" {
		return "", validationError("empty_password", "password cannot be empty")
	}
	return password, nil
}

// ensureProfile creates the named profile when it is absent so that selecting a
// profile records it even before any of its settings are known.
func ensureProfile(cfg *config.File, name string) {
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]config.Profile{}
	}
	if _, ok := cfg.Profiles[name]; !ok {
		cfg.Profiles[name] = config.Profile{}
	}
}

// updateProfile applies change to the named profile and stores the result,
// creating the profile when it is absent.
func updateProfile(cfg *config.File, name string, change func(*config.Profile)) {
	ensureProfile(cfg, name)
	profile := cfg.Profiles[name]
	change(&profile)
	cfg.Profiles[name] = profile
}
