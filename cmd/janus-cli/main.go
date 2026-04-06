// Janus CLI - Command-line interface for Janus
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"

	// ===== ADDED: Enhanced generation imports =====
	"janus/internal/core/generator/enhanced"
	"janus/internal/core/generator/models"
	"janus/internal/core/generator/validator"
	// ==============================================
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

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new scenario",
		RunE:  createScenario,
	}
	createCmd.Flags().StringP("name", "n", "", "Scenario name (required)")
	createCmd.Flags().StringP("template", "t", "", "Template name (required)")
	createCmd.MarkFlagRequired("name")
	createCmd.MarkFlagRequired("template")
	scenarioCmd.AddCommand(createCmd)

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

	// Generate command (old)
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

	// ===== ADDED: Enhanced generate command =====
	generateQuickCmd := &cobra.Command{
		Use:   "quick",
		Short: "Quick generation with flexible options",
		Long:  "Generate files with enhanced features: size/count modes, PII/filler distribution, disk space validation",
		RunE:  generateQuick,
	}

	generateQuickCmd.Flags().String("name", "CLI Generated", "Scenario name")
	generateQuickCmd.Flags().String("preset", "", "Use a prebuilt scenario (see: janus-cli gen presets)")
	generateQuickCmd.Flags().String("total-size", "", "Total size (e.g., 5GB, 100MB) - uses size mode")
	generateQuickCmd.Flags().Int("file-count", 0, "Number of files - uses count mode")
	generateQuickCmd.Flags().String("file-size-min", "1KB", "Minimum file size")
	generateQuickCmd.Flags().String("file-size-max", "10MB", "Maximum file size")
	generateQuickCmd.Flags().Float64("pii-percent", 10, "PII percentage (0-100)")
	generateQuickCmd.Flags().String("pii-type", "standard", "PII type: standard, healthcare, or financial")
	generateQuickCmd.Flags().Float64("filler-percent", 90, "Filler percentage (0-100, must total 100 with pii-percent)")
	generateQuickCmd.Flags().String("output", "./payloads/quick", "Output directory")
	generateQuickCmd.Flags().Int("seed", 0, "Random seed for reproducible generation (0=random)")
	generateQuickCmd.Flags().Bool("watch", false, "Stream progress via WebSocket until complete")

	presetsCmd := &cobra.Command{
		Use:   "presets",
		Short: "List available prebuilt scenarios",
		RunE:  listPresets,
	}

	genCmd := &cobra.Command{
		Use:   "gen",
		Short: "Enhanced file generation",
	}
	genCmd.AddCommand(generateQuickCmd)
	genCmd.AddCommand(presetsCmd)

	// Profile commands
	profileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage generation profiles",
	}

	profileCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new profile",
		RunE:  createProfile,
	}
	profileCreateCmd.Flags().String("name", "", "Profile name (required)")
	profileCreateCmd.Flags().String("description", "", "Profile description")
	profileCreateCmd.Flags().String("total-size", "", "Total size (e.g. 5GB) — size mode")
	profileCreateCmd.Flags().Int("file-count", 0, "Number of files — count mode")
	profileCreateCmd.Flags().String("file-size-min", "1KB", "Minimum file size")
	profileCreateCmd.Flags().String("file-size-max", "10MB", "Maximum file size")
	profileCreateCmd.Flags().Float64("pii-percent", 10, "PII percentage (0-100)")
	profileCreateCmd.Flags().String("pii-type", "standard", "PII type: standard, healthcare, financial")
	profileCreateCmd.Flags().Float64("filler-percent", 90, "Filler percentage")
	profileCreateCmd.Flags().String("output", "./payloads", "Output directory")
	profileCreateCmd.Flags().Int("seed", 0, "Seed for reproducible generation (0=random)")
	profileCreateCmd.MarkFlagRequired("name")

	profileListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		RunE:  listProfiles,
	}

	profileShowCmd := &cobra.Command{
		Use:   "show [id]",
		Short: "Show full profile details",
		Args:  cobra.ExactArgs(1),
		RunE:  showProfile,
	}

	profileUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update a profile",
		Args:  cobra.ExactArgs(1),
		RunE:  updateProfile,
	}
	profileUpdateCmd.Flags().String("name", "", "New profile name")
	profileUpdateCmd.Flags().String("description", "", "New description")
	profileUpdateCmd.Flags().String("total-size", "", "Total size (e.g. 5GB)")
	profileUpdateCmd.Flags().Int("file-count", 0, "Number of files")
	profileUpdateCmd.Flags().String("file-size-min", "", "Minimum file size")
	profileUpdateCmd.Flags().String("file-size-max", "", "Maximum file size")
	profileUpdateCmd.Flags().Float64("pii-percent", 0, "PII percentage (0-100)")
	profileUpdateCmd.Flags().Float64("filler-percent", 0, "Filler percentage (0-100)")
	profileUpdateCmd.Flags().String("pii-type", "", "PII type: standard, healthcare, financial")
	profileUpdateCmd.Flags().String("output", "", "Output directory")

	profileDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE:  deleteProfile,
	}

	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileUpdateCmd)
	profileCmd.AddCommand(profileDeleteCmd)

	// gen profile <id> [--watch]
	genProfileCmd := &cobra.Command{
		Use:   "profile [id]",
		Short: "Generate using a saved profile",
		Args:  cobra.ExactArgs(1),
		RunE:  generateFromProfile,
	}
	genProfileCmd.Flags().String("output", "", "Override output directory")
	genProfileCmd.Flags().Bool("watch", false, "Stream progress via WebSocket until complete")
	genCmd.AddCommand(genProfileCmd)

	// Add all commands
	rootCmd.AddCommand(scenarioCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(encryptCmd)
	rootCmd.AddCommand(jobCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(genCmd)
	rootCmd.AddCommand(profileCmd)

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

func apiPut(path string, data interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPut, apiURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
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

// ===== ADDED: Enhanced generation handler =====

func listPresets(cmd *cobra.Command, args []string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PRESET\tMODE\tSIZE/COUNT\tPII%\tTYPE")
	for _, name := range enhanced.ListPrebuiltScenarios() {
		p, _ := enhanced.GetPrebuiltScenario(name)
		mode := "count"
		constraint := fmt.Sprintf("%d files", p.FileCount)
		if p.TotalSize != "" {
			mode = "size"
			constraint = p.TotalSize
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%.0f%%\t%s\n",
			name, mode, constraint, p.PIIPercent, p.PIIType)
	}
	w.Flush()
	return nil
}

func generateQuick(cmd *cobra.Command, args []string) error {
	preset, _ := cmd.Flags().GetString("preset")

	var opts enhanced.QuickGenerateOptions

	if preset != "" {
		p, ok := enhanced.GetPrebuiltScenario(preset)
		if !ok {
			return fmt.Errorf("unknown preset %q — run 'janus-cli gen presets' to list available", preset)
		}
		opts = p
		// Allow flag overrides on top of preset
		if cmd.Flags().Changed("name") {
			opts.Name, _ = cmd.Flags().GetString("name")
		}
		if cmd.Flags().Changed("output") {
			opts.OutputPath, _ = cmd.Flags().GetString("output")
		}
		if cmd.Flags().Changed("seed") {
			seed, _ := cmd.Flags().GetInt("seed")
			opts.Seed = int64(seed)
		}
	} else {
		name, _ := cmd.Flags().GetString("name")
		totalSize, _ := cmd.Flags().GetString("total-size")
		fileCount, _ := cmd.Flags().GetInt("file-count")
		fileSizeMin, _ := cmd.Flags().GetString("file-size-min")
		fileSizeMax, _ := cmd.Flags().GetString("file-size-max")
		piiPercent, _ := cmd.Flags().GetFloat64("pii-percent")
		piiType, _ := cmd.Flags().GetString("pii-type")
		fillerPercent, _ := cmd.Flags().GetFloat64("filler-percent")
		output, _ := cmd.Flags().GetString("output")
		seed, _ := cmd.Flags().GetInt("seed")

		opts = enhanced.QuickGenerateOptions{
			Name:           name,
			OutputPath:     output,
			TotalSize:      totalSize,
			FileCount:      fileCount,
			FileSizeMin:    fileSizeMin,
			FileSizeMax:    fileSizeMax,
			PIIPercent:     piiPercent,
			PIIType:        piiType,
			FillerPercent:  fillerPercent,
			Formats:        []string{"csv", "json", "txt"},
			DirectoryDepth: 3,
			Seed:           int64(seed),
		}
	}

	// Validate before sending
	enhancedOpts, err := opts.ToEnhancedOptions()
	if err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	v := validator.New()

	fmt.Println("Validating options...")
	validation := v.ValidateAll(enhancedOpts)
	if validation.HasErrors() {
		return fmt.Errorf("validation failed:\n%s", validation.ErrorMessages())
	}

	diskValidation, diskInfo := v.ValidateDiskSpace(enhancedOpts)
	if diskValidation.HasErrors() {
		return fmt.Errorf("disk space validation failed:\n%s", diskValidation.ErrorMessages())
	}

	if len(validation.Warnings) > 0 || len(diskValidation.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		if len(validation.Warnings) > 0 {
			fmt.Println(validation.WarningMessages())
		}
		if len(diskValidation.Warnings) > 0 {
			fmt.Println(diskValidation.WarningMessages())
		}
		fmt.Print("\nProceed anyway? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Show plan
	fmt.Println("\nGeneration Plan:")
	if opts.TotalSize != "" {
		fmt.Printf("  Mode:    size-constrained (%s)\n", opts.TotalSize)
	} else {
		fmt.Printf("  Mode:    count-constrained (%d files)\n", opts.FileCount)
	}
	fmt.Printf("  Output:  %s\n", opts.OutputPath)
	fmt.Printf("  Files:   %s - %s\n", opts.FileSizeMin, opts.FileSizeMax)
	fmt.Printf("  PII:     %.0f%% (%s)\n", opts.PIIPercent, opts.PIIType)
	fmt.Printf("  Filler:  %.0f%%\n", opts.FillerPercent)

	if diskInfo != nil {
		fmt.Printf("\nDisk Space:\n")
		fmt.Printf("  Available: %s\n", models.FormatBytes(diskInfo.Available))
		fmt.Printf("  Required:  ~%s\n", models.FormatBytes(diskInfo.RequiredSpace))
	}

	fmt.Print("\nStart generation? [Y/n]: ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm == "n" || confirm == "N" {
		fmt.Println("Cancelled.")
		return nil
	}

	watch, _ := cmd.Flags().GetBool("watch")

	fmt.Println("\nStarting generation...")

	jsonData, err := json.Marshal(opts)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	// Open WebSocket before firing the request so we don't miss early events
	var wsConn *websocket.Conn
	if watch {
		wsURL := "ws://" + strings.TrimPrefix(apiURL, "http://") + "/ws/v1/activity"
		wsConn, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not connect to WebSocket (%v) — proceeding without watch\n", err)
			watch = false
		}
	}

	resp, err := http.Post(apiURL+"/api/v1/generate/enhanced", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		if wsConn != nil {
			wsConn.Close()
		}
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		if wsConn != nil {
			wsConn.Close()
		}
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Println("Generation started.")

	if !watch {
		fmt.Printf("Monitor: ws://%s/ws/v1/activity\n", strings.TrimPrefix(apiURL, "http://"))
		return nil
	}

	defer wsConn.Close()
	fmt.Println("Streaming progress (Ctrl+C to detach)...")
	fmt.Println()

	for {
		_, msg, err := wsConn.ReadMessage()
		if err != nil {
			break
		}

		var event map[string]interface{}
		if err := json.Unmarshal(msg, &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)
		if !strings.HasPrefix(eventType, "enhanced_generation_") {
			continue
		}

		switch eventType {
		case "enhanced_generation_progress":
			current, _ := event["current"].(float64)
			total, _ := event["total"].(float64)
			pct, _ := event["percent"].(float64)
			file, _ := event["current_file"].(string)
			fmt.Printf("\r  [%d/%d] %.1f%%  %s          ",
				int(current), int(total), pct, file)
		case "enhanced_generation_completed":
			filesCreated, _ := event["files_created"].(float64)
			bytesWritten, _ := event["bytes_written"].(float64)
			durationMs, _ := event["duration_ms"].(float64)
			fmt.Printf("\n\nComplete: %d files, %s in %.1fs\n",
				int(filesCreated),
				models.FormatBytes(int64(bytesWritten)),
				durationMs/1000)
			return nil
		case "enhanced_generation_failed":
			errMsg, _ := event["error"].(string)
			fmt.Println()
			return fmt.Errorf("generation failed: %s", errMsg)
		}
	}

	return nil
}

// ==============================================

// --- Profile command implementations ---

func createProfile(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	totalSize, _ := cmd.Flags().GetString("total-size")
	fileCount, _ := cmd.Flags().GetInt("file-count")
	fileSizeMin, _ := cmd.Flags().GetString("file-size-min")
	fileSizeMax, _ := cmd.Flags().GetString("file-size-max")
	piiPercent, _ := cmd.Flags().GetFloat64("pii-percent")
	piiType, _ := cmd.Flags().GetString("pii-type")
	fillerPercent, _ := cmd.Flags().GetFloat64("filler-percent")
	output, _ := cmd.Flags().GetString("output")
	seed, _ := cmd.Flags().GetInt("seed")

	if totalSize == "" && fileCount == 0 {
		return fmt.Errorf("specify either --total-size or --file-count")
	}

	reqBody := map[string]interface{}{
		"name":        name,
		"description": description,
		"options": map[string]interface{}{
			"name":           name,
			"output_path":    output,
			"total_size":     totalSize,
			"file_count":     fileCount,
			"file_size_min":  fileSizeMin,
			"file_size_max":  fileSizeMax,
			"pii_percent":    piiPercent,
			"pii_type":       piiType,
			"filler_percent": fillerPercent,
			"formats":        []string{"csv", "json", "txt"},
			"seed":           seed,
		},
	}

	body, err := apiPost("/api/v1/profiles", reqBody)
	if err != nil {
		return err
	}

	if outputJSON {
		fmt.Println(string(body))
		return nil
	}

	var p struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.Unmarshal(body, &p)
	fmt.Printf("Created profile: %s (ID: %s)\n", p.Name, p.ID)
	return nil
}

func listProfiles(cmd *cobra.Command, args []string) error {
	body, err := apiGet("/api/v1/profiles")
	if err != nil {
		return err
	}

	if outputJSON {
		fmt.Println(string(body))
		return nil
	}

	var result struct {
		Profiles []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Options     struct {
				TotalSize     string  `json:"total_size"`
				FileCount     int     `json:"file_count"`
				PIIPercent    float64 `json:"pii_percent"`
				FillerPercent float64 `json:"filler_percent"`
				PIIType       string  `json:"pii_type"`
			} `json:"options"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tPII%\tFILLER%\tTYPE")
	for _, p := range result.Profiles {
		desc := p.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%.0f%%\t%.0f%%\t%s\n",
			p.ID[:8], p.Name, desc,
			p.Options.PIIPercent,
			p.Options.FillerPercent,
			p.Options.PIIType)
	}
	w.Flush()
	return nil
}

func getProfile(cmd *cobra.Command, args []string) error {
	body, err := apiGet("/api/v1/profiles/" + args[0])
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func showProfile(cmd *cobra.Command, args []string) error {
	body, err := apiGet("/api/v1/profiles/" + args[0])
	if err != nil {
		return err
	}
	if outputJSON {
		fmt.Println(string(body))
		return nil
	}
	var p struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Options     struct {
			OutputPath    string   `json:"output_path"`
			TotalSize     string   `json:"total_size"`
			FileCount     int      `json:"file_count"`
			FileSizeMin   string   `json:"file_size_min"`
			FileSizeMax   string   `json:"file_size_max"`
			PIIPercent    float64  `json:"pii_percent"`
			PIIType       string   `json:"pii_type"`
			FillerPercent float64  `json:"filler_percent"`
			Formats       []string `json:"formats"`
			Seed          int64    `json:"seed"`
		} `json:"options"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return err
	}
	fmt.Printf("Name:        %s\n", p.Name)
	fmt.Printf("ID:          %s\n", p.ID)
	if p.Description != "" {
		fmt.Printf("Description: %s\n", p.Description)
	}
	fmt.Printf("Created:     %s\n", p.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", p.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()
	fmt.Println("Options:")
	fmt.Printf("  Output:    %s\n", p.Options.OutputPath)
	if p.Options.TotalSize != "" {
		fmt.Printf("  Mode:      size (%s)\n", p.Options.TotalSize)
	} else {
		fmt.Printf("  Mode:      count (%d files)\n", p.Options.FileCount)
	}
	fmt.Printf("  File Size: %s – %s\n", p.Options.FileSizeMin, p.Options.FileSizeMax)
	fmt.Printf("  PII:       %.0f%% (%s)\n", p.Options.PIIPercent, p.Options.PIIType)
	fmt.Printf("  Filler:    %.0f%%\n", p.Options.FillerPercent)
	if len(p.Options.Formats) > 0 {
		fmt.Printf("  Formats:   %s\n", strings.Join(p.Options.Formats, ", "))
	}
	if p.Options.Seed != 0 {
		fmt.Printf("  Seed:      %d\n", p.Options.Seed)
	}
	return nil
}

func updateProfile(cmd *cobra.Command, args []string) error {
	id := args[0]
	updates := map[string]interface{}{}
	options := map[string]interface{}{}

	if cmd.Flags().Changed("name") {
		v, _ := cmd.Flags().GetString("name")
		updates["name"] = v
	}
	if cmd.Flags().Changed("description") {
		v, _ := cmd.Flags().GetString("description")
		updates["description"] = v
	}
	if cmd.Flags().Changed("file-count") {
		v, _ := cmd.Flags().GetInt("file-count")
		options["file_count"] = v
	}
	if cmd.Flags().Changed("total-size") {
		v, _ := cmd.Flags().GetString("total-size")
		options["total_size"] = v
	}
	if cmd.Flags().Changed("pii-percent") {
		v, _ := cmd.Flags().GetFloat64("pii-percent")
		options["pii_percent"] = v
	}
	if cmd.Flags().Changed("filler-percent") {
		v, _ := cmd.Flags().GetFloat64("filler-percent")
		options["filler_percent"] = v
	}
	if cmd.Flags().Changed("pii-type") {
		v, _ := cmd.Flags().GetString("pii-type")
		options["pii_type"] = v
	}
	if cmd.Flags().Changed("output") {
		v, _ := cmd.Flags().GetString("output")
		options["output_path"] = v
	}
	if cmd.Flags().Changed("file-size-min") {
		v, _ := cmd.Flags().GetString("file-size-min")
		options["file_size_min"] = v
	}
	if cmd.Flags().Changed("file-size-max") {
		v, _ := cmd.Flags().GetString("file-size-max")
		options["file_size_max"] = v
	}
	if len(options) > 0 {
		updates["options"] = options
	}
	if len(updates) == 0 {
		return fmt.Errorf("no changes specified — use flags to specify what to update")
	}

	body, err := apiPut("/api/v1/profiles/"+id, updates)
	if err != nil {
		return err
	}
	if outputJSON {
		fmt.Println(string(body))
		return nil
	}
	var p struct {
		Name string `json:"name"`
	}
	json.Unmarshal(body, &p)
	fmt.Printf("Updated profile: %s\n", p.Name)
	return nil
}

func deleteProfile(cmd *cobra.Command, args []string) error {
	id := args[0]

	fmt.Printf("Delete profile %q? This cannot be undone. [y/N]: ", id)
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Cancelled.")
		return nil
	}

	req, err := http.NewRequest("DELETE", apiURL+"/api/v1/profiles/"+id, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(b))
	}
	fmt.Printf("Deleted profile: %s\n", id)
	return nil
}

func generateFromProfile(cmd *cobra.Command, args []string) error {
	id := args[0]
	watch, _ := cmd.Flags().GetBool("watch")

	overrides := map[string]interface{}{}
	if cmd.Flags().Changed("output") {
		output, _ := cmd.Flags().GetString("output")
		overrides["output_path"] = output
	}

	var wsConn *websocket.Conn
	if watch {
		wsURL := "ws://" + strings.TrimPrefix(apiURL, "http://") + "/ws/v1/activity"
		var err error
		wsConn, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not connect to WebSocket (%v) — proceeding without watch\n", err)
			watch = false
		}
	}

	body, err := apiPost("/api/v1/profiles/"+id+"/generate", overrides)
	if err != nil {
		if wsConn != nil {
			wsConn.Close()
		}
		return err
	}

	if outputJSON {
		fmt.Println(string(body))
		if wsConn != nil {
			wsConn.Close()
		}
		return nil
	}

	var genResp struct {
		ScenarioID string `json:"scenario_id"`
	}
	json.Unmarshal(body, &genResp)
	fmt.Printf("Generation started (scenario: %s)\n", genResp.ScenarioID)

	if !watch {
		if wsConn != nil {
			wsConn.Close()
		}
		return nil
	}

	defer wsConn.Close()
	fmt.Println("Streaming progress (Ctrl+C to detach)...")
	fmt.Println()

	for {
		_, msg, err := wsConn.ReadMessage()
		if err != nil {
			break
		}
		var event map[string]interface{}
		if err := json.Unmarshal(msg, &event); err != nil {
			continue
		}
		eventType, _ := event["type"].(string)
		if !strings.HasPrefix(eventType, "enhanced_generation_") {
			continue
		}
		switch eventType {
		case "enhanced_generation_progress":
			current, _ := event["current"].(float64)
			total, _ := event["total"].(float64)
			pct, _ := event["percent"].(float64)
			file, _ := event["current_file"].(string)
			fmt.Printf("\r  [%d/%d] %.1f%%  %s          ",
				int(current), int(total), pct, file)
		case "enhanced_generation_completed":
			filesCreated, _ := event["files_created"].(float64)
			bytesWritten, _ := event["bytes_written"].(float64)
			durationMs, _ := event["duration_ms"].(float64)
			fmt.Printf("\n\nComplete: %d files, %.1f MB in %.1fs\n",
				int(filesCreated), float64(bytesWritten)/1024/1024, durationMs/1000)
			return nil
		case "enhanced_generation_failed":
			errMsg, _ := event["error"].(string)
			fmt.Println()
			return fmt.Errorf("generation failed: %s", errMsg)
		}
	}
	return nil
}
