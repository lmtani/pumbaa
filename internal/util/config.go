package util

import (
	"fmt"
	"os"
	"text/template"
)

func createCromwellConfig(savePath string, config Config) error {
	fmt.Println("Creating cromwell config file...")
	configTemplate := getTemplate()

	// Parse the template
	tmpl, err := template.New("config").Parse(configTemplate)
	if err != nil {
		return err
	}

	// create a new file
	file, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Render the template with the configuration values
	config.Database.URL = fmt.Sprintf(
		"jdbc:mysql://%s:%d/cromwell?rewriteBatchedStatements=true",
		config.Database.Host, config.Database.Port)
	err = tmpl.Execute(file, config)
	if err != nil {
		return err
	}
	return nil
}

func getTemplate() string {

	template := `backend {
  default = {{ .Backend.Default }}

  providers {

    {{ .Backend.Provider }} {
      actor-factory = "{{ .Backend.ActorFactory }}"
      config {
        max-concurrent-workflows = {{ .Backend.MaxConcurrentWorkflows }}
        concurrent-job-limit = {{ .Backend.ConcurrentJobLimit }}

        filesystems {
          local {
            localization: [
              {{ range .Backend.FileSystems.Localization }}"{{ . }}", {{ end }}
            ]
          }
        }
      }
    }
  }
}

database {
  profile = "{{ .Database.Profile }}"
  db {
    driver = "{{ .Database.Driver }}"
    url = "{{ .Database.URL }}"
    user = "{{ .Database.User }}"
    password = "{{ .Database.Password }}"
    connectionTimeout = {{ .Database.ConnectionTimeout }}
  }
}

call-caching {
  enabled = {{ .CallCaching.Enabled }}
  invalidate-bad-cache-results = {{ .CallCaching.InvalidateBadCacheResults }}
}

docker {
    perform-registry-lookup-if-digest-is-provided = {{ .Docker.PerformRegistryLookupIfDigestIsProvided }}
}
`
	return template
}

type Backend struct {
	Default                string
	Provider               string
	ActorFactory           string
	MaxConcurrentWorkflows int
	ConcurrentJobLimit     int
	FileSystems            struct {
		Localization []string
	}
}

type Database struct {
	Profile           string
	Driver            string
	URL               string
	Host              string
	Port              int
	User              string
	Password          string
	ConnectionTimeout int
}

type CallCaching struct {
	Enabled                   bool
	InvalidateBadCacheResults bool
}

type Docker struct {
	PerformRegistryLookupIfDigestIsProvided bool
}

type Config struct {
	Backend
	Database
	CallCaching
	Docker
}
