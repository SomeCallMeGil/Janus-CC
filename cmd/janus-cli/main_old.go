//go:build ignore

// Janus CLI - Command-line interface for Janus
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var (
	apiURL     string
	outputJSON bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "janus-cli",
		Short: "Janus Security Testing Platform CLI",
		Long:  "Command-line interface for the Janus security testing platform",
	}

	rootCmd.PersistentFlags().StringVar(&apiURL, "api", "http://localhost:8080", "API server URL")
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	// Scenario commands
	scenarioCmd := &cobra.Command{
		Use:   "scenario",
		Short: "Manage scenarios",
	}

	scenarioCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all scenarios",
		RunE:  listScenarios,
	})

	scenarioCmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create a new scenario",
		RunE:  createScenario,
	})

	scenarioCmd.AddCommand(&cobra.Command{
		Use:   "get [id]",
		Short: "Get scenario details",
		Args:  cobra.ExactArgs(1),
		RunE:  getScenario,
	})

	scenarioCmd.AddCommand(&cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a scenario",
		Args:  cobra.ExactArgs(1),
		RunE:  deleteScenario,
	})

	scenarioCmd.AddCommand(&cobra.Command{
		Use:   "stats [id]",
		Short: "Get scenario statistics",
		Args:  cobra.ExactArgs(1),
		RunE:  getScenarioStats,
	})

	// Generate command
	generateCmd := &cobra.Command{
		Use:   "generate [scenario-id]",
		Short: "Generate files for a scenario",
		Args:  cobra.ExactArgs(1),
		RunE:  generateFiles,
	}

	// Encrypt command
	encryptCmd := &cobra.Command{
		Use:   "encrypt [scenario-id]",
		Short: "Encrypt files in a scenario",
		Args:  cobra.ExactArgs(1),
		RunE:  encryptFiles,
	}
	encryptCmd.Flags().Float64P("percentage", "p", 25.0, "Percentage of files to encrypt")
	encryptCmd.Flags().StringP("password", "w", "", "Encryption password")
	encryptCmd.Flags().StringP("mode", "m", "partial", "Encryption mode (full/partial)")

	// Job commands
	jobCmd := &cobra.Command{
		Use:   "job",
		Short: "Manage encryption jobs",
	}

	jobCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all jobs",
		RunE:  listJobs,
	})

	jobCmd.AddCommand(&cobra.Command{
		Use:   "get [id]",
		Short: "Get job details",
		Args:  cobra.ExactArgs(1),
		RunE:  getJob,
	})

	// Export command
	exportCmd := &cobra.Command{
		Use:   "export [scenario-id]",
		Short: "Export file manifest to CSV",
		Args:  cobra.ExactArgs(1),
		RunE:  exportManifest,
	}
	exportCmd.Flags().StringP("output", "o", "manifest.csv", "Output file path")

	// Health check
	healthCmd := &cobra.Command{
		Use:   "health",
		Short: "Check server health",
		RunE:  checkHealth,
	}

	// Add all commands
	rootCmd.AddCommand(scenarioCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(encryptCmd)
	rootCmd.AddCommand(jobCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(healthCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// API client helpers
func apiGet(path string) ([]byte, error) {
	resp, err := http.Get(apiURL + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	return body, nil
}

func apiPost(path string, data interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(apiURL+path, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	return body, nil
}

func apiDelete(path string) error {
	req, err := http.NewRequest("DELETE", apiURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(body))
	}

	return nil
}

// Command implementations
func checkHealth(cmd *cobra.Command, args []string) error {
	body, err := apiGet("/api/v1/health")
	if err != nil {
		return err
	}

	if outputJSON {
		fmt.Println(string(body))
	} else {
		fmt.Println("✓ Server is healthy")
	}

	return nil
}

func listScenarios(cmd *cobra.Command, args []string) error {
	body, err := apiGet("/api/v1/scenarios")
	if err != nil {
		return err
	}

	if outputJSON {
		fmt.Println(string(body))
		return nil
	}

	var result struct {
		Scenarios []struct {
			ID          string    `json:"id"`
			Name        string    `json:"name"`
			Status      string    `json:"status"`
			CreatedAt   time.Time `json:"created_at"`
		} `json:"scenarios"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tCREATED")
	for _, s := range result.Scenarios {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			s.ID[:8], s.Name, s.Status, s.CreatedAt.Format("2006-01-02"))
	}
	w.Flush()

	return nil
}

func createScenario(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	template, _ := cmd.Flags().GetString("template")

	if name == "" {
		return fmt.Errorf("name is required (--name)")
	}
	if template == "" {
		return fmt.Errorf("template is required (--template)")
	}

	data := map[string]interface{}{
		"name":     name,
		"template": template,
	}

	body, err := apiPost("/api/v1/scenarios", data)
	if err != nil {
		return err
	}

	if outputJSON {
		fmt.Println(string(body))
	} else {
		var result struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		json.Unmarshal(body, &result)
		fmt.Printf("✓ Created scenario: %s (ID: %s)\n", result.Name, result.ID)
	}

	return nil
}

func getScenario(cmd *cobra.Command, args []string) error {
	id := args[0]
	body, err := apiGet("/api/v1/scenarios/" + id)
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	return nil
}

func deleteScenario(cmd *cobra.Command, args []string) error {
	id := args[0]
	
	if err := apiDelete("/api/v1/scenarios/" + id); err != nil {
		return err
	}

	fmt.Printf("✓ Deleted scenario: %s\n", id)
	return nil
}

func getScenarioStats(cmd *cobra.Command, args []string) error {
	id := args[0]
	body, err := apiGet("/api/v1/scenarios/" + id + "/stats")
	if err != nil {
		return err
	}

	if outputJSON {
		fmt.Println(string(body))
		return nil
	}

	var stats struct {
		TotalFiles       int     `json:"total_files"`
		EncryptedFiles   int     `json:"encrypted_files"`
		PendingFiles     int     `json:"pending_files"`
		EncryptedPercent float64 `json:"encrypted_percent"`
		TotalSize        int64   `json:"total_size"`
	}

	if err := json.Unmarshal(body, &stats); err != nil {
		return err
	}

	fmt.Printf("Total Files:     %d\n", stats.TotalFiles)
	fmt.Printf("Encrypted:       %d (%.1f%%)\n", stats.EncryptedFiles, stats.EncryptedPercent)
	fmt.Printf("Pending:         %d\n", stats.PendingFiles)
	fmt.Printf("Total Size:      %d MB\n", stats.TotalSize/1024/1024)

	return nil
}

func generateFiles(cmd *cobra.Command, args []string) error {
	id := args[0]
	
	_, err := apiPost("/api/v1/scenarios/"+id+"/generate", nil)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Generation started for scenario: %s\n", id)
	fmt.Println("  (This runs in the background. Check server logs for progress)")

	return nil
}

func encryptFiles(cmd *cobra.Command, args []string) error {
	id := args[0]
	percentage, _ := cmd.Flags().GetFloat64("percentage")
	password, _ := cmd.Flags().GetString("password")
	mode, _ := cmd.Flags().GetString("mode")

	if password == "" {
		return fmt.Errorf("password is required (--password)")
	}

	data := map[string]interface{}{
		"percentage": percentage,
		"password":   password,
		"mode":       mode,
	}

	_, err := apiPost("/api/v1/scenarios/"+id+"/encrypt", data)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Encryption started for scenario: %s\n", id)
	fmt.Printf("  Percentage: %.1f%%\n", percentage)
	fmt.Printf("  Mode: %s\n", mode)
	fmt.Println("  (This runs in the background. Check server logs for progress)")

	return nil
}

func listJobs(cmd *cobra.Command, args []string) error {
	body, err := apiGet("/api/v1/jobs")
	if err != nil {
		return err
	}

	if outputJSON {
		fmt.Println(string(body))
		return nil
	}

	var result struct {
		Jobs []struct {
			ID               int64     `json:"id"`
			ScenarioID       string    `json:"scenario_id"`
			Status           string    `json:"status"`
			TargetPercentage float64   `json:"target_percentage"`
			ScheduledAt      time.Time `json:"scheduled_at"`
		} `json:"jobs"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSCENARIO\tSTATUS\tTARGET%\tSCHEDULED")
	for _, j := range result.Jobs {
		fmt.Fprintf(w, "%d\t%s\t%s\t%.1f%%\t%s\n",
			j.ID, j.ScenarioID[:8], j.Status, j.TargetPercentage,
			j.ScheduledAt.Format("2006-01-02 15:04"))
	}
	w.Flush()

	return nil
}

func getJob(cmd *cobra.Command, args []string) error {
	id := args[0]
	body, err := apiGet("/api/v1/jobs/" + id)
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	return nil
}

func exportManifest(cmd *cobra.Command, args []string) error {
	id := args[0]
	output, _ := cmd.Flags().GetString("output")

	resp, err := http.Get(apiURL + "/api/v1/scenarios/" + id + "/export")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(body))
	}

	outFile, err := os.Create(output)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Exported manifest to: %s\n", output)
	return nil
}

func init() {
	// Add flags to create scenario command
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new scenario",
		RunE:  createScenario,
	}
	createCmd.Flags().StringP("name", "n", "", "Scenario name (required)")
	createCmd.Flags().StringP("template", "t", "", "Template name (required)")
	createCmd.MarkFlagRequired("name")
	createCmd.MarkFlagRequired("template")
}
