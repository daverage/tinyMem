package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/a-marczewski/tinymem/internal/app"
	"github.com/spf13/cobra"

	"github.com/a-marczewski/tinymem/internal/config"
	"github.com/a-marczewski/tinymem/internal/doctor"
	"github.com/a-marczewski/tinymem/internal/evidence"
	"github.com/a-marczewski/tinymem/internal/extract"
	"github.com/a-marczewski/tinymem/internal/inject"
	"github.com/a-marczewski/tinymem/internal/llm"
	"github.com/a-marczewski/tinymem/internal/memory"
	"github.com/a-marczewski/tinymem/internal/recall"
	"github.com/a-marczewski/tinymem/internal/server/mcp"
	"github.com/a-marczewski/tinymem/internal/server/proxy"
	"github.com/a-marczewski/tinymem/internal/storage"
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
	// Set up services using the app instance
	evidenceService := evidence.NewService(a.DB)
	recallEngine := recall.NewEngine(a.Memory, evidenceService, a.Config) // a.Memory is already memory.Service
	injector := inject.NewMemoryInjector(recallEngine)
	llmClient := llm.NewClient(a.Config)
	extractor := extract.NewExtractor(evidenceService)

	// Create and start proxy server
	proxyServer := proxy.NewServer(a.Config, injector, llmClient, a.Memory, evidenceService, recallEngine, extractor)
	a.Logger.Info("Starting proxy server on port ", a.Config.ProxyPort)

	if err := proxyServer.Start(); err != nil {
		a.Logger.Error("Failed to start proxy server", zap.Error(err))
	}
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Model Context Protocol server",
}

func runMcpCmd(a *app.App, cmd *cobra.Command, args []string) {
	// Set up services using the app instance
	evidenceService := evidence.NewService(a.DB)
	recallEngine := recall.NewEngine(a.Memory, evidenceService, a.Config) // a.Memory is already memory.Service
	extractor := extract.NewExtractor(evidenceService)

	// Create and start MCP server
	mcpServer := mcp.NewServer(a.Config, a.DB, a.Memory, evidenceService, recallEngine, extractor)
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
	evidenceService := evidence.NewService(a.DB)
	recallEngine := recall.NewEngine(a.Memory, evidenceService, a.Config) // a.Memory is already memory.Service
	injector := inject.NewMemoryInjector(recallEngine)

	// Perform recall based on the command
	commandStr := fmt.Sprintf("Running command: %s", args[0])
	if len(args) > 1 {
		commandStr += fmt.Sprintf(" with arguments: %s", strings.Join(args[1:], " "))
	}

	injectedPrompt, err := injector.InjectMemoriesIntoPrompt(commandStr, 10, 2000)
	if err != nil {
		a.Logger.Warn("Failed to inject memories", zap.Error(err))
		injectedPrompt = commandStr
	}

	fmt.Printf("Executing with memory context:\n%s\n", injectedPrompt)
	// In a real implementation, we would execute the command here
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the health of tinyMem services",
}

func runHealthCmd(a *app.App, cmd *cobra.Command, args []string) {
	a.Logger.Info("Checking health...")

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

	a.Logger.Info("Health check complete.")
	fmt.Println("Health check complete.")
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show memory statistics",
}

func runStatsCmd(a *app.App, cmd *cobra.Command, args []string) {
	// Get all memories to calculate stats
	memories, err := a.Memory.GetAllMemories("default_project") // In real impl, get from context
	if err != nil {
		a.Logger.Error("Failed to get memories for stats", zap.Error(err))
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
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostics on tinyMem installation",
}

func runDoctorCmd(a *app.App, cmd *cobra.Command, args []string) {
	// Run diagnostics
	doctorRunner := doctor.NewRunner(a.Config, a.DB)
	diagnostics := doctorRunner.RunAll()
	diagnostics.PrintReport()
}

var recentCmd = &cobra.Command{
	Use:   "recent",
	Short: "Show recent memories",
}

func runRecentCmd(a *app.App, cmd *cobra.Command, args []string) {
	// Get recent memories
	memories, err := a.Memory.GetAllMemories("default_project") // In real impl, get from context
	if err != nil {
		a.Logger.Error("Failed to get recent memories", zap.Error(err))
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
}

var queryCmd = &cobra.Command{
	Use:   "query [search terms]",
	Short: "Search memories",
	Args:  cobra.MinimumNArgs(1),
}

func runQueryCmd(a *app.App, cmd *cobra.Command, args []string) {
	// Set up services using the app instance
	evidenceService := evidence.NewService(a.DB)
	recallEngine := recall.NewEngine(a.Memory, evidenceService, a.Config) // a.Memory is already memory.Service

	// Perform search
	query := strings.Join(args, " ")
	results, err := recallEngine.Recall(recall.RecallOptions{
		Query:    query,
		MaxItems: 10,
	})
	if err != nil {
		a.Logger.Error("Search failed", zap.Error(err))
		return
	}

	fmt.Printf("Search results for '%s':\n\n", query)
	for i, result := range results {
		mem := result.Memory
		fmt.Printf("[%d] (%.2f) %s: %s\n", i+1, string(mem.Type), mem.Summary)
		if mem.Detail != "" {
			fmt.Printf("    Details: %s\n", mem.Detail)
		}
		fmt.Printf("    Date: %s\n", mem.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
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

	if err := rootCmd.Execute(); err != nil {
		appInstance.Logger.Error("Root command execution failed", zap.Error(err))
		os.Exit(1)
	}
}

