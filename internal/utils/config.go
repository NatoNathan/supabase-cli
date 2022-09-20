package utils

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/spf13/afero"
)

var (
	DbImage     string
	NetId       string
	DbId        string
	KongId      string
	GotrueId    string
	InbucketId  string
	RealtimeId  string
	RestId      string
	StorageId   string
	DifferId    string
	PgmetaId    string
	StudioId    string
	DenoRelayId string

	InitialSchemaSql string
	//go:embed templates/initial_schemas/13.sql
	initialSchemaPg13Sql string
	//go:embed templates/initial_schemas/14.sql
	initialSchemaPg14Sql string

	authExternalProviders = []string{
		"apple",
		"azure",
		"bitbucket",
		"discord",
		"facebook",
		"github",
		"gitlab",
		"google",
		"twitch",
		"twitter",
		"slack",
		"spotify",
	}

	//go:embed templates/init_config.toml
	initConfigEmbed       string
	initConfigTemplate, _ = template.New("initConfig").Parse(initConfigEmbed)

	//go:embed templates/init_config.test.toml
	testInitConfigEmbed       string
	testInitConfigTemplate, _ = template.New("initConfig.test").Parse(testInitConfigEmbed)
)

var Config config

type (
	config struct {
		ProjectId string `toml:"project_id"`
		Api       api
		Db        db
		Studio    studio
		Inbucket  inbucket
		Auth      auth
		// TODO
		// Scripts   scripts
	}

	api struct {
		Port            uint
		Schemas         []string
		ExtraSearchPath []string `toml:"extra_search_path"`
		MaxRows         uint     `toml:"max_rows"`
		ApiURL          string   `toml:"api_url_env_var"`
	}

	db struct {
		Port         uint
		MajorVersion uint   `toml:"major_version"`
		DbURL        string `toml:"db_url_env_var"`
	}

	studio struct {
		Port      uint
		StudioURL string `toml:"studio_url_env_var"`
	}

	inbucket struct {
		Port        uint
		SmtpPort    uint   `toml:"smtp_port"`
		Pop3Port    uint   `toml:"pop3_port"`
		InbucketURL string `toml:"inbucket_url_env_var"`
	}

	auth struct {
		SiteUrl                string   `toml:"site_url"`
		AdditionalRedirectUrls []string `toml:"additional_redirect_urls"`
		JwtExpiry              uint     `toml:"jwt_expiry"`
		EnableSignup           *bool    `toml:"enable_signup"`
		Email                  email
		External               map[string]provider
		AnonKey                string `toml:"anon_key_env_var"`
		ServiceRoleKey         string `toml:"service_role_key_env_var"`
	}

	email struct {
		EnableSignup         *bool `toml:"enable_signup"`
		DoubleConfirmChanges *bool `toml:"double_confirm_changes"`
		EnableConfirmations  *bool `toml:"enable_confirmations"`
	}

	provider struct {
		Enabled  bool
		ClientId string `toml:"client_id"`
		Secret   string
	}

	// TODO
	// scripts struct {
	// 	BeforeMigrations string `toml:"before_migrations"`
	// 	AfterMigrations  string `toml:"after_migrations"`
	// }
)

func LoadConfig() error {
	return LoadConfigFS(afero.NewOsFs())
}

func LoadConfigFS(fsys afero.Fs) error {
	// TODO: provide a config interface for all sub commands to use fsys
	if _, err := toml.DecodeFS(afero.NewIOFS(fsys), ConfigPath, &Config); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("Missing config: %w", err)
	} else if err != nil {
		return fmt.Errorf("Failed to read config: %w", err)
	}

	// Process decoded TOML.
	{
		if Config.ProjectId == "" {
			return errors.New("Missing required field in config: project_id")
		} else {
			NetId = "supabase_network_" + Config.ProjectId
			DbId = "supabase_db_" + Config.ProjectId
			KongId = "supabase_kong_" + Config.ProjectId
			GotrueId = "supabase_auth_" + Config.ProjectId
			InbucketId = "supabase_inbucket_" + Config.ProjectId
			RealtimeId = "supabase_realtime_" + Config.ProjectId
			RestId = "supabase_rest_" + Config.ProjectId
			StorageId = "supabase_storage_" + Config.ProjectId
			DifferId = "supabase_differ_" + Config.ProjectId
			PgmetaId = "supabase_pg_meta_" + Config.ProjectId
			StudioId = "supabase_studio_" + Config.ProjectId
			DenoRelayId = "supabase_deno_relay_" + Config.ProjectId
		}
		if Config.Api.Port == 0 {
			return errors.New("Missing required field in config: api.port")
		}
		if Config.Api.MaxRows == 0 {
			Config.Api.MaxRows = 1000
		}
		if Config.Api.ApiURL == "" {
			Config.Api.ApiURL = "API_URL"
		}
		if Config.Db.Port == 0 {
			return errors.New("Missing required field in config: db.port")
		}
		switch Config.Db.MajorVersion {
		case 0:
			return errors.New("Missing required field in config: db.major_version")
		case 12:
			return errors.New("Postgres version 12.x is unsupported. To use the CLI, either start a new project or follow project migration steps here: https://supabase.com/docs/guides/database#migrating-between-projects.")
		case 13:
			DbImage = Pg13Image
			InitialSchemaSql = initialSchemaPg13Sql
		case 14:
			DbImage = Pg14Image
			InitialSchemaSql = initialSchemaPg14Sql
		default:
			return fmt.Errorf("Failed reading config: Invalid %s: %v.", Aqua("db.major_version"), Config.Db.MajorVersion)
		}
		if Config.Db.DbURL == "" {
			Config.Db.DbURL = "DB_URL"
		}
		if Config.Studio.Port == 0 {
			return errors.New("Missing required field in config: studio.port")
		}
		if Config.Studio.StudioURL == "" {
			Config.Studio.StudioURL = "STUDIO_URL"
		}
		if Config.Inbucket.Port == 0 {
			return errors.New("Missing required field in config: inbucket.port")
		}
		if Config.Inbucket.InbucketURL == "" {
			Config.Inbucket.InbucketURL = "INBUCKET_URL"
		}
		if Config.Auth.SiteUrl == "" {
			return errors.New("Missing required field in config: auth.site_url")
		}
		if Config.Auth.AnonKey == "" {
			Config.Auth.AnonKey = "ANON_KEY"
		}
		if Config.Auth.ServiceRoleKey == "" {
			Config.Auth.ServiceRoleKey = "SERVICE_ROLE_KEY"
		}
		if Config.Auth.JwtExpiry == 0 {
			Config.Auth.JwtExpiry = 3600
		}
		if Config.Auth.EnableSignup == nil {
			x := true
			Config.Auth.EnableSignup = &x
		}
		if Config.Auth.Email.EnableSignup == nil {
			x := true
			Config.Auth.Email.EnableSignup = &x
		}
		if Config.Auth.Email.DoubleConfirmChanges == nil {
			x := true
			Config.Auth.Email.DoubleConfirmChanges = &x
		}
		if Config.Auth.Email.EnableConfirmations == nil {
			x := true
			Config.Auth.Email.EnableConfirmations = &x
		}
		if Config.Auth.External == nil {
			Config.Auth.External = map[string]provider{}
		}

		for _, ext := range authExternalProviders {
			if _, ok := Config.Auth.External[ext]; !ok {
				Config.Auth.External[ext] = provider{
					Enabled:  false,
					ClientId: "",
					Secret:   "",
				}
			} else if Config.Auth.External[ext].Enabled {
				maybeLoadEnv := func(s string) (string, error) {
					matches := regexp.MustCompile(`^env\((.*)\)$`).FindStringSubmatch(s)
					if len(matches) == 0 {
						return s, nil
					}

					envName := matches[1]
					value := os.Getenv(envName)
					if value == "" {
						return "", errors.New(`Error evaluating "env(` + envName + `)": environment variable ` + envName + " is unset.")
					}

					return value, nil
				}

				var clientId, secret string

				if Config.Auth.External[ext].ClientId == "" {
					return fmt.Errorf("Missing required field in config: auth.external.%s.client_id", ext)
				} else {
					v, err := maybeLoadEnv(Config.Auth.External[ext].ClientId)
					if err != nil {
						return err
					}
					clientId = v
				}
				if Config.Auth.External[ext].Secret == "" {
					return fmt.Errorf("Missing required field in config: auth.external.%s.secret", ext)
				} else {
					v, err := maybeLoadEnv(Config.Auth.External[ext].Secret)
					if err != nil {
						return err
					}
					secret = v
				}

				Config.Auth.External[ext] = provider{
					Enabled:  true,
					ClientId: clientId,
					Secret:   secret,
				}
			}
		}
	}

	return nil
}

func WriteConfig(fsys afero.Fs, test bool) error {
	// Using current directory name as project id
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir := filepath.Base(cwd)

	var initConfigBuf bytes.Buffer
	var tmpl *template.Template
	if test {
		tmpl = testInitConfigTemplate
	} else {
		tmpl = initConfigTemplate
	}

	if err := tmpl.Execute(
		&initConfigBuf,
		struct{ ProjectId string }{ProjectId: dir},
	); err != nil {
		return err
	}

	if err := MkdirIfNotExistFS(fsys, filepath.Dir(ConfigPath)); err != nil {
		return err
	}

	if err := afero.WriteFile(fsys, ConfigPath, initConfigBuf.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}
