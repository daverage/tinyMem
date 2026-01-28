package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/a-marczewski/tinymem/internal/analytics"
	"github.com/a-marczewski/tinymem/internal/app"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/a-marczewski/tinymem/internal/doctor"
	"github.com/a-marczewski/tinymem/internal/evidence"
	"github.com/a-marczewski/tinymem/internal/inject"
	"github.com/a-marczewski/tinymem/internal/logging"
	"github.com/a-marczewski/tinymem/internal/memory"
	"github.com/a-marczewski/tinymem/internal/recall"
	"github.com/a-marczewski/tinymem/internal/semantic"
	"github.com/a-marczewski/tinymem/internal/server/mcp"
	"github.com/a-marczewski/tinymem/internal/server/proxy"
)

var rootCmd = &cobra.Command{
	Use:   "tinymem",
	Short: "tinyMem - Persistent memory for LLMs",
	Long:  `tinyMem provides persistent memory capabilities for LLMs with evidence-based truth validation.`,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(recentCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(contractCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(writeCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate the autocompletion script for the specified shell",
	Long: `Generate the autocompletion script for tinyMem for the specified shell.
See each command's help for details on how to use the generated script.
	`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		switch args[0] {
		case "bash":
			err = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			err = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			err = cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			err = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating completion script: %v\n", err)
			os.Exit(1)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
}

func runVersionCmd(a *app.App, cmd *cobra.Command, args []string) {
	fmt.Println("tinyMem v0.1.0")
}

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Start the proxy server",
}

func runProxyCmd(a *app.App, cmd *cobra.Command, args []string) {
	// Set the server mode to ProxyMode
	a.ServerMode = doctor.ProxyMode

	// Create and start proxy server
	proxyServer := proxy.NewServer(a) // Pass the app instance directly
	a.Logger.Info("Starting proxy server", zap.Int("port", a.Config.ProxyPort))

	if err := proxyServer.Start(); err != nil {
		a.Logger.Error("Failed to start proxy server", zap.Error(err))
	}
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Model Context Protocol server",
}

func runMcpCmd(a *app.App, cmd *cobra.Command, args []string) {
	// Set the server mode to MCPMode
	a.ServerMode = doctor.MCPMode

	if a.Logger != nil {
		_ = a.Logger.Sync()
	}

	logFile := a.Config.LogFile
	if logFile == "" {
		logDir := filepath.Join(a.Config.TinyMemDir, "logs")
		logFile = filepath.Join(logDir, fmt.Sprintf("tinymem-%s.log", time.Now().Format("2006-01-02")))
	} else if !filepath.IsAbs(logFile) {
		logFile = filepath.Join(a.Config.TinyMemDir, logFile)
	}
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err == nil {
		if logger, err := logging.NewLoggerWithStderr(a.Config.LogLevel, logFile, false); err == nil {
			a.Logger = logger
		}
	}

	// Create and start MCP server
	mcpServer := mcp.NewServer(a) // Pass the app instance directly
	a.Logger.Info("Starting MCP server")

	if err := mcpServer.Start(); err != nil {
		a.Logger.Error("Failed to start MCP server", zap.Error(err))
	}
}

var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Run a command with memory injection",
	Args:  cobra.MinimumNArgs(1),
}

func runRunCmd(a *app.App, cmd *cobra.Command, args []string) {
	// Set up services using the app instance
	evidenceService := evidence.NewService(a.DB, a.Config)
	var recallEngine recall.Recaller
	if a.Config.SemanticEnabled {
		recallEngine = semantic.NewSemanticEngine(a.DB, a.Memory, evidenceService, a.Config, a.Logger)
	} else {
		recallEngine = recall.NewEngine(a.Memory, evidenceService, a.Config, a.Logger, a.DB.GetConnection())
	}
	defer recallEngine.Close()
	injector := inject.NewMemoryInjector(recallEngine)

	// Perform recall based on the command
	commandStr := fmt.Sprintf("Running command: %s", args[0])
	if len(args) > 1 {
		commandStr += fmt.Sprintf(" with arguments: %s", strings.Join(args[1:], " "))
	}

	injectedPrompt, err := injector.InjectMemoriesIntoPrompt(commandStr, a.ProjectID, a.Config.RecallMaxItems, a.Config.RecallMaxTokens)
	if err != nil {
		a.Logger.Warn("Failed to inject memories", zap.Error(err))
		injectedPrompt = commandStr
	}

	cmdToRun := exec.Command(args[0], args[1:]...)
	cmdToRun.Env = append(os.Environ(), fmt.Sprintf("TINYMEM_CONTEXT=%s", injectedPrompt))
	cmdToRun.Stdout = os.Stdout
	cmdToRun.Stderr = os.Stderr
	cmdToRun.Stdin = os.Stdin

	if err := cmdToRun.Run(); err != nil {
		a.Logger.Error("Command failed", zap.Error(err))
	}
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the health of tinyMem services",
}

func runHealthCmd(a *app.App, cmd *cobra.Command, args []string) {
	a.Logger.Info("Checking health...", zap.String("mode", string(a.ServerMode)))

	// Check database connectivity
	if err := a.DB.GetConnection().Ping(); err != nil {
		a.Logger.Error("Database connectivity check failed", zap.Error(err))
		fmt.Printf("❌ Database connectivity: %v\n", err)
	} else {
		a.Logger.Info("Database connectivity: OK")
		fmt.Println("✅ Database connectivity: OK")
	}

	// Check if we can perform a simple query
	if _, err := a.DB.GetConnection().Exec("SELECT 1"); err != nil {
		a.Logger.Error("Database query check failed", zap.Error(err))
		fmt.Printf("❌ Database query: %v\n", err)
	} else {
		a.Logger.Info("Database query: OK")
		fmt.Println("✅ Database query: OK")
	}

	// Mode-specific health checks
	switch a.ServerMode {
	case doctor.MCPMode:
		// For MCP mode, check if we can access memories
		if _, err := a.Memory.GetAllMemories(a.ProjectID); err != nil {
			a.Logger.Error("Memory service health check failed", zap.Error(err))
			fmt.Printf("❌ Memory service: %v\n", err)
		} else {
			a.Logger.Info("Memory service health check passed")
			fmt.Println("✅ Memory service: OK")
		}
	case doctor.ProxyMode:
		// For proxy mode, check if proxy is listening (attempt to connect to proxy port)
		if err := checkProxyListening(a.Config.ProxyPort); err != nil {
			a.Logger.Warn("Proxy not listening", zap.Error(err), zap.Int("port", a.Config.ProxyPort))
			fmt.Printf("! Proxy not listening on port %d: %v\n", a.Config.ProxyPort, err)
		} else {
			a.Logger.Info("Proxy is listening", zap.Int("port", a.Config.ProxyPort))
			fmt.Printf("✅ Proxy listening on port %d\n", a.Config.ProxyPort)
		}
	}

	a.Logger.Info("Health check complete.")
	fmt.Println("Health check complete.")
}

// Helper function to check if proxy is listening on a port
func checkProxyListening(port int) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("invalid port %d", port)
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		return err
	}
	return conn.Close()
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show memory statistics",
}

func runStatsCmd(a *app.App, cmd *cobra.Command, args []string) {
	memories, err := a.Memory.GetAllMemories(a.ProjectID)
	if err != nil {
		a.Logger.Error("Failed to get memories for stats", zap.Error(err))
		fmt.Printf("❌ Error retrieving memories: %v\n", err)
		return
	}

	// Count by type
	typeCounts := make(map[memory.Type]int)
	for _, mem := range memories {
		typeCounts[mem.Type]++
	}

	fmt.Printf("Total memories: %d\n", len(memories))
	fmt.Println("By type:")
	for memType, count := range typeCounts {
		fmt.Printf("  %s: %d\n", string(memType), count)
	}

	// Task-specific statistics using analytics package
	if taskCount, exists := typeCounts[memory.Task]; exists && taskCount > 0 {
		fmt.Printf("\nTask Statistics:\n")

		// Initialize analytics service
		taskAnalytics := analytics.NewTaskAnalytics(a.DB.GetConnection())

		// Get comprehensive task metrics
		metrics, err := taskAnalytics.GetTaskMetrics(a.ProjectID)
		if err != nil {
			fmt.Printf("  Error getting detailed task metrics: %v\n", err)

			// Fallback to basic calculation
			var completedTasks int
			for _, mem := range memories {
				if mem.Type == memory.Task && strings.Contains(mem.Detail, "Completed: true") {
					completedTasks++
				}
			}

			completionRate := 0.0
			if taskCount > 0 {
				completionRate = float64(completedTasks) / float64(taskCount) * 100
			}

			fmt.Printf("  Total Tasks: %d\n", taskCount)
			fmt.Printf("  Completed: %d\n", completedTasks)
			fmt.Printf("  Completion Rate: %.1f%%\n", completionRate)
		} else {
			fmt.Printf("  Total Tasks: %d\n", metrics.TotalTasks)
			fmt.Printf("  Completed: %d\n", metrics.CompletedTasks)
			fmt.Printf("  Incomplete: %d\n", metrics.IncompleteTasks)
			fmt.Printf("  Completion Rate: %.1f%%\n", metrics.CompletionRate)

			if metrics.AverageTimeToComplete > 0 {
				hours := metrics.AverageTimeToComplete.Hours()
				fmt.Printf("  Avg. Time to Complete: %.1f hours\n", hours)
			}

			// Task breakdown by section if available
			if len(metrics.TasksBySection) > 0 {
				fmt.Printf("  By Section:\n")
				for section, sectionMetrics := range metrics.TasksBySection {
					fmt.Printf("    %s: %d/%d (%.1f%%)\n",
						section, sectionMetrics.Completed, sectionMetrics.Total, sectionMetrics.Rate)
				}
			}
		}
	}

	// Mode-specific stats
	switch a.ServerMode {
	case doctor.MCPMode:
		fmt.Println("\nMode: MCP (Model Context Protocol)")
	case doctor.ProxyMode:
		fmt.Println("\nMode: Proxy")
		// Additional proxy-specific stats could be added here
		fmt.Printf("Proxy Port: %d\n", a.Config.ProxyPort)
	}
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostics on tinyMem installation",
}

func runDoctorCmd(a *app.App, cmd *cobra.Command, args []string) {
	// Use the server mode from the app instance
	// This allows the doctor command to behave appropriately based on how tinyMem was started
	doctorRunner := doctor.NewRunnerWithMode(a.Config, a.DB, a.ProjectID, a.Memory, a.ServerMode)
	diagnostics := doctorRunner.RunAll()
	diagnostics.PrintReport()
}

var recentCmd = &cobra.Command{
	Use:   "recent",
	Short: "Show recent memories",
}

func runRecentCmd(a *app.App, cmd *cobra.Command, args []string) {
	// Get recent memories
	memories, err := a.Memory.GetAllMemories(a.ProjectID)
	if err != nil {
		a.Logger.Error("Failed to get recent memories", zap.Error(err))
		fmt.Printf("❌ Error retrieving memories: %v\n", err)
		return
	}

	// Show only the 10 most recent
	limit := 10
	if len(memories) < limit {
		limit = len(memories)
	}

	fmt.Printf("Recent memories (showing %d of %d total):\n\n", limit, len(memories))
	for i := 0; i < limit && i < len(memories); i++ {
		mem := memories[i]
		fmt.Printf("[%d] %s: %s\n", i+1, string(mem.Type), mem.Summary)
		if mem.Detail != "" {
			fmt.Printf("    Details: %s\n", mem.Detail)
		}
		fmt.Printf("    Date: %s\n", mem.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	// Mode-specific info
	switch a.ServerMode {
	case doctor.MCPMode:
		fmt.Println("Mode: MCP (Model Context Protocol)")
	case doctor.ProxyMode:
		fmt.Println("Mode: Proxy")
	}
}

var queryCmd = &cobra.Command{
	Use:   "query [search terms]",
	Short: "Search memories",
	Args:  cobra.MinimumNArgs(1),
}

func runQueryCmd(a *app.App, cmd *cobra.Command, args []string) {
	// Set up services using the app instance
	evidenceService := evidence.NewService(a.DB, a.Config)
	var recallEngine recall.Recaller
	if a.Config.SemanticEnabled {
		recallEngine = semantic.NewSemanticEngine(a.DB, a.Memory, evidenceService, a.Config, a.Logger)
	} else {
		recallEngine = recall.NewEngine(a.Memory, evidenceService, a.Config, a.Logger, a.DB.GetConnection())
	}
	defer recallEngine.Close()

	// Perform search
	query := strings.Join(args, " ")
	results, err := recallEngine.Recall(recall.RecallOptions{
		ProjectID: a.ProjectID,
		Query:     query,
		MaxItems:  a.Config.RecallMaxItems,
	})
	if err != nil {
		a.Logger.Error("Search failed", zap.Error(err))
		fmt.Printf("❌ Search failed: %v\n", err)
		return
	}

	fmt.Printf("Search results for '%s':\n\n", query)
	for i, result := range results {
		mem := result.Memory
		fmt.Printf("[%d] (%.2f) %s: %s\n", i+1, result.Score, string(mem.Type), mem.Summary)
		if mem.Detail != "" {
			fmt.Printf("    Details: %s\n", mem.Detail)
		}
		fmt.Printf("    Date: %s\n", mem.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	// Mode-specific info
	switch a.ServerMode {
	case doctor.MCPMode:
		fmt.Println("Mode: MCP (Model Context Protocol)")
	case doctor.ProxyMode:
		fmt.Println("Mode: Proxy")
	}
}

var writeCmd = &cobra.Command{
	Use:   "write",
	Short: "Write a new memory",
	Long: `Write a new memory to the tinyMem database.

Memory types: fact, claim, plan, decision, constraint, observation, note
Note: Facts require evidence and cannot be created directly via CLI.

Examples:
  tinymem write --type claim --summary "API uses REST" --detail "Based on endpoint patterns"
  tinymem write --type decision --summary "Use SQLite for storage" --source "architecture review"
  tinymem write --type note --summary "TODO: Add unit tests"`,
}

var writeType string
var writeSummary string
var writeDetail string
var writeKey string
var writeSource string

func init() {
	writeCmd.Flags().StringVarP(&writeType, "type", "t", "note", "Memory type: claim, plan, decision, constraint, observation, note (fact requires evidence)")
	writeCmd.Flags().StringVarP(&writeSummary, "summary", "s", "", "Brief summary of the memory (required)")
	writeCmd.Flags().StringVarP(&writeDetail, "detail", "d", "", "Detailed description")
	writeCmd.Flags().StringVarP(&writeKey, "key", "k", "", "Optional unique key for the memory")
	writeCmd.Flags().StringVar(&writeSource, "source", "", "Source of the memory")
	_ = writeCmd.MarkFlagRequired("summary")
}

func runWriteCmd(a *app.App, cmd *cobra.Command, args []string) {
	// Validate memory type
	memType := memory.Type(writeType)
	if !memType.IsValid() {
		fmt.Printf("❌ Invalid memory type: %s\n", writeType)
		fmt.Println("Valid types: fact, claim, plan, decision, constraint, observation, note")
		os.Exit(1)
	}

	// Facts require evidence and can't be created via CLI
	if memType == memory.Fact {
		fmt.Println("❌ Facts cannot be created directly via CLI - they require verified evidence.")
		fmt.Println("Use 'claim' type instead, or use the MCP interface with evidence.")
		os.Exit(1)
	}

	if writeSummary == "" {
		fmt.Println("❌ Summary is required. Use --summary or -s flag.")
		os.Exit(1)
	}

	newMemory := &memory.Memory{
		ProjectID: a.ProjectID,
		Type:      memType,
		Summary:   writeSummary,
		Detail:    writeDetail,
	}

	if writeKey != "" {
		newMemory.Key = &writeKey
	}
	if writeSource != "" {
		newMemory.Source = &writeSource
	}

	if err := a.Memory.CreateMemory(newMemory); err != nil {
		a.Logger.Error("Failed to create memory", zap.Error(err))
		fmt.Printf("❌ Failed to create memory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Memory created successfully!\n")
	fmt.Printf("   Type: %s\n", memType)
	fmt.Printf("   Summary: %s\n", writeSummary)
	if writeDetail != "" {
		fmt.Printf("   Detail: %s\n", writeDetail)
	}
	if writeKey != "" {
		fmt.Printf("   Key: %s\n", writeKey)
	}
	if writeSource != "" {
		fmt.Printf("   Source: %s\n", writeSource)
	}
}

var contractCmd = &cobra.Command{
	Use:   "addContract",
	Short: "Add the MANDATORY TINYMEM CONTROL PROTOCOL to agent markdown files",
}

func runContractCmd(a *app.App, cmd *cobra.Command, args []string) {
	if err := memory.AddContract(); err != nil {
		a.Logger.Error("Failed to add contract", zap.Error(err))
		fmt.Printf("❌ Failed to add contract: %v\n", err)
	}
}

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Show a snapshot dashboard of memory state",
}

// newAppRunner creates a Cobra Run function closure with the app.App instance.
func newAppRunner(a *app.App, runFunc func(*app.App, *cobra.Command, []string)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		runFunc(a, cmd, args)
	}
}

func main() {
	appInstance, err := app.NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}
	defer appInstance.Close()

	// Wrap the Run functions with newAppRunner to pass the app instance
	versionCmd.Run = newAppRunner(appInstance, runVersionCmd)
	proxyCmd.Run = newAppRunner(appInstance, runProxyCmd)
	mcpCmd.Run = newAppRunner(appInstance, runMcpCmd)
	runCmd.Run = newAppRunner(appInstance, runRunCmd)
	healthCmd.Run = newAppRunner(appInstance, runHealthCmd)
	statsCmd.Run = newAppRunner(appInstance, runStatsCmd)
	doctorCmd.Run = newAppRunner(appInstance, runDoctorCmd)
	recentCmd.Run = newAppRunner(appInstance, runRecentCmd)
	queryCmd.Run = newAppRunner(appInstance, runQueryCmd)
	contractCmd.Run = newAppRunner(appInstance, runContractCmd)
	dashboardCmd.Run = newAppRunner(appInstance, runDashboardCmd)
	writeCmd.Run = newAppRunner(appInstance, runWriteCmd)

	if err := rootCmd.Execute(); err != nil {
		appInstance.Logger.Error("Root command execution failed", zap.Error(err))
		os.Exit(1)
	}
}
