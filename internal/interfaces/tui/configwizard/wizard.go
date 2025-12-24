package configwizard

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// ConfigWizard runs an interactive configuration wizard using huh forms.
func ConfigWizard() error {
	cfg, _ := config.LoadFileConfig()

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(common.PrimaryColor).
		MarginBottom(1)

	fmt.Println(titleStyle.Render("🐗 Pumbaa Configuration Wizard"))
	fmt.Println()

	// Step 1: Choose LLM Provider
	var provider string
	if cfg.LLMProvider != "" {
		provider = cfg.LLMProvider
	}

	providerForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose your LLM Provider").
				Description("Select the AI backend you want to use").
				Options(
					huh.NewOption("Ollama (Local, Free)", "ollama"),
					huh.NewOption("Gemini API (Google AI Studio)", "gemini"),
					huh.NewOption("Vertex AI (Google Cloud)", "vertex"),
				).
				Value(&provider),
		),
	).WithTheme(huh.ThemeDracula())

	if err := providerForm.Run(); err != nil {
		return err
	}

	cfg.LLMProvider = provider

	// Step 2: Provider-specific configuration
	switch provider {
	case "ollama":
		if err := configureOllama(cfg); err != nil {
			return err
		}
	case "gemini":
		if err := configureGemini(cfg); err != nil {
			return err
		}
	case "vertex":
		if err := configureVertex(cfg); err != nil {
			return err
		}
	}

	// Step 3: Cromwell configuration
	if err := configureCromwell(cfg); err != nil {
		return err
	}

	// Step 4: Optional WDL directory
	if err := configureWDL(cfg); err != nil {
		return err
	}

	// Save configuration
	if err := config.SaveFileConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Success message
	successStyle := lipgloss.NewStyle().
		Foreground(common.StatusSucceeded).
		Bold(true)

	fmt.Println()
	fmt.Println(successStyle.Render("✓ Configuration saved successfully!"))
	fmt.Printf("  File: %s\n", config.DefaultConfigPath())
	fmt.Println()
	fmt.Println("You can now use: pumbaa chat")

	return nil
}

func configureOllama(cfg *config.FileConfig) error {
	host := cfg.OllamaHost
	if host == "" {
		host = "http://localhost:11434"
	}
	model := cfg.OllamaModel
	if model == "" {
		model = "llama3.2:3b"
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Ollama Host").
				Description("URL of your Ollama server").
				Value(&host),
			huh.NewInput().
				Title("Ollama Model").
				Description("Model name (e.g., llama3.2:3b, gpt-oss:20b)").
				Value(&model),
		),
	).WithTheme(huh.ThemeDracula())

	if err := form.Run(); err != nil {
		return err
	}

	cfg.OllamaHost = host
	cfg.OllamaModel = model
	return nil
}

func configureGemini(cfg *config.FileConfig) error {
	apiKey := cfg.GeminiAPIKey
	model := cfg.GeminiModel
	if model == "" {
		model = "gemini-2.5-flash"
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Gemini API Key").
				Description("Get your key at: https://aistudio.google.com/apikey").
				Value(&apiKey).
				EchoMode(huh.EchoModePassword),
			huh.NewSelect[string]().
				Title("Gemini Model").
				Options(
					huh.NewOption("gemini-2.5-flash (Recommended)", "gemini-2.5-flash"),
					huh.NewOption("gemini-2.0-flash", "gemini-2.0-flash"),
					huh.NewOption("gemini-1.5-pro", "gemini-1.5-pro"),
					huh.NewOption("gemini-1.5-flash", "gemini-1.5-flash"),
				).
				Value(&model),
		),
	).WithTheme(huh.ThemeDracula())

	if err := form.Run(); err != nil {
		return err
	}

	cfg.GeminiAPIKey = apiKey
	cfg.GeminiModel = model
	return nil
}

func configureVertex(cfg *config.FileConfig) error {
	project := cfg.VertexProject
	location := cfg.VertexLocation
	if location == "" {
		location = "us-central1"
	}
	model := cfg.VertexModel
	if model == "" {
		model = "gemini-2.5-flash"
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("GCP Project ID").
				Description("Your Google Cloud project ID").
				Value(&project),
			huh.NewInput().
				Title("Vertex AI Location").
				Description("Region (e.g., us-central1, europe-west1)").
				Value(&location),
			huh.NewSelect[string]().
				Title("Vertex AI Model").
				Options(
					huh.NewOption("gemini-2.5-flash (Recommended)", "gemini-2.5-flash"),
					huh.NewOption("gemini-2.0-flash", "gemini-2.0-flash"),
					huh.NewOption("gemini-1.5-pro", "gemini-1.5-pro"),
					huh.NewOption("gemini-1.5-flash", "gemini-1.5-flash"),
				).
				Value(&model),
		),
	).WithTheme(huh.ThemeDracula())

	if err := form.Run(); err != nil {
		return err
	}

	cfg.VertexProject = project
	cfg.VertexLocation = location
	cfg.VertexModel = model
	return nil
}

func configureCromwell(cfg *config.FileConfig) error {
	host := cfg.CromwellHost
	if host == "" {
		host = "http://localhost:8000"
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Cromwell Server URL").
				Description("URL of your Cromwell workflow engine").
				Value(&host),
		),
	).WithTheme(huh.ThemeDracula())

	if err := form.Run(); err != nil {
		return err
	}

	cfg.CromwellHost = host
	return nil
}

func configureWDL(cfg *config.FileConfig) error {
	var configure bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Configure WDL Directory?").
				Description("Optional: Add a directory with WDL files for chat context").
				Value(&configure),
		),
	).WithTheme(huh.ThemeDracula())

	if err := confirmForm.Run(); err != nil {
		return err
	}

	if configure {
		// Use directory picker
		startPath := cfg.WDLDirectory
		selectedPath, err := RunDirectoryPicker(startPath)
		if err != nil {
			return err
		}
		if selectedPath != "" {
			cfg.WDLDirectory = selectedPath
		}
	}

	return nil
}
