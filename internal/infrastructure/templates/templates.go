// Package templates provides embedded HTML templates for report generation.
package templates

import (
	"bytes"
	"embed"
	"encoding/base64"
	"html/template"
)

//go:embed report.html favicon.png
var templateFS embed.FS

// ReportData contains the data to be injected into the HTML template.
type ReportData struct {
	DataJSON      template.JS // Raw task data as JSON
	WorkflowsJSON template.JS // List of workflow IDs
	FaviconBase64 string      // Favicon as base64 encoded PNG
}

// RenderReport generates the HTML report from the template and data.
func RenderReport(data ReportData) (string, error) {
	// Load favicon as base64
	faviconBytes, err := templateFS.ReadFile("favicon.png")
	if err == nil {
		data.FaviconBase64 = base64.StdEncoding.EncodeToString(faviconBytes)
	}

	tmpl, err := template.ParseFS(templateFS, "report.html")
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
