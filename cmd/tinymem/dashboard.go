package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/daverage/tinymem/internal/analytics"
	"github.com/daverage/tinymem/internal/app"
	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/memory"
	"github.com/daverage/tinymem/internal/tasks"
	"github.com/spf13/cobra"

	_ "github.com/mattn/go-sqlite3"
)

// runDashboardCmd executes the dashboard command
func runDashboardCmd(a *app.App, cmd *cobra.Command, args []string) {
	// Find project root and check for .tinyMem directory
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		fmt.Println("Error finding project root:", err)
		os.Exit(1)
	}

	tinyMemDir := config.GetTinyMemDir(projectRoot)
	if _, err := os.Stat(tinyMemDir); os.IsNotExist(err) {
		fmt.Println("No .tinyMem directory found.")
		fmt.Println("Run `tinymem init` in your project root.")
		os.Exit(1)
	}

	dbPath := filepath.Join(tinyMemDir, "store.sqlite3")

	// Use the existing database connection from the app instance
	dbConn := a.DB.GetConnection()

	// Check if database is accessible
	if err := dbConn.Ping(); err != nil {
		fmt.Printf("Error connecting to database: %v\n", err)
		os.Exit(1)
	}

	// Synchronize tasks from tinyTasks.md if it exists
	taskService := tasks.NewService(a.DB, a.Memory, a.ProjectID)
	taskFilePath := filepath.Join(projectRoot, "tinyTasks.md")
	if _, err := os.Stat(taskFilePath); err == nil {
		if err := taskService.SyncTasksFromFile(taskFilePath); err != nil {
			fmt.Printf("Warning: failed to synchronize tasks from tinyTasks.md: %v\n", err)
		}
	}

	// Initialize analytics
	taskAnalytics := analytics.NewTaskAnalytics(dbConn)

	// Print dashboard header
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│                    tinyMem Dashboard                        │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Section 1: Header / Project Status
	printHeaderSection(projectRoot, tinyMemDir, dbPath, dbConn)

	// Section 2: Integrity Summary
	printIntegritySummary(dbConn)

	// Section 3: Recent Decisions
	printRecentDecisions(dbConn)

	// Section 4: Active Constraints
	printActiveConstraints(dbConn)

	// Section 5: Needs Attention / Suspicious Items
	printNeedsAttention(dbConn)

	// Section 6: Task Analytics with Enhanced Visualizations
	printEnhancedTaskAnalytics(taskAnalytics, a.ProjectID)

	// Section 7: Recall Effectiveness
	printRecallEffectiveness(dbConn)
}

// printHeaderSection prints the header/project status section
func printHeaderSection(projectRoot, tinyMemDir, dbPath string, db *sql.DB) {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 1️⃣  Header / Project Status                                  │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Project Root:\t%s\n", projectRoot)
	fmt.Fprintf(w, ".tinyMem Path:\t%s\n", tinyMemDir)

	// Get DB file size
	if fileInfo, err := os.Stat(dbPath); err == nil {
		size := fileInfo.Size()
		var sizeStr string
		switch {
		case size < 1024:
			sizeStr = fmt.Sprintf("%d B", size)
		case size < 1024*1024:
			sizeStr = fmt.Sprintf("%.2f KB", float64(size)/1024)
		default:
			sizeStr = fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
		}
		fmt.Fprintf(w, "DB File Size:\t%s\n", sizeStr)
	} else {
		fmt.Fprintf(w, "DB File Size:\tError reading size\n")
	}

	// Check if FTS5 is enabled
	var ftsAvailable bool
	err := db.QueryRow("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type = 'table' AND name = 'memories_fts';").Scan(&ftsAvailable)
	if err != nil {
		fmt.Fprintf(w, "SQLite FTS5 Enabled:\terror checking\n")
	} else {
		ftsStatus := "no"
		if ftsAvailable {
			ftsStatus = "yes"
		}
		fmt.Fprintf(w, "SQLite FTS5 Enabled:\t%s\n", ftsStatus)
	}

	// Get last memory activity timestamp and total memory count
	var lastActivityStr sql.NullString
	var totalCount int
	err = db.QueryRow("SELECT MAX(updated_at), COUNT(*) FROM memories;").Scan(&lastActivityStr, &totalCount)
	if err != nil {
		fmt.Fprintf(w, "Last Activity:\terror retrieving\n")
		fmt.Fprintf(w, "Total Memories:\terror retrieving\n")
	} else {
		if lastActivityStr.Valid && lastActivityStr.String != "" {
			// Try parsing different datetime formats that SQLite might store
			var lastActivity time.Time
			formats := []string{
				"2006-01-02 15:04:05.999999-07:00",
				"2006-01-02 15:04:05.999999+00:00",
				"2006-01-02 15:04:05.999999",
				"2006-01-02 15:04:05",
				time.RFC3339,
			}
			parsed := false
			for _, format := range formats {
				if t, parseErr := time.Parse(format, lastActivityStr.String); parseErr == nil {
					lastActivity = t
					parsed = true
					break
				}
			}
			if parsed {
				fmt.Fprintf(w, "Last Activity:\t%s\n", lastActivity.Format("2006-01-02 15:04:05"))
			} else {
				// Fallback: show the raw string if parsing fails
				fmt.Fprintf(w, "Last Activity:\t%s\n", lastActivityStr.String)
			}
		} else {
			fmt.Fprintf(w, "Last Activity:\tnever\n")
		}
		fmt.Fprintf(w, "Total Memories:\t%d\n", totalCount)
	}

	w.Flush()
	fmt.Println()
}

// printIntegritySummary prints the integrity summary section
func printIntegritySummary(db *sql.DB) {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 2️⃣  Integrity Summary                                        │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Count facts with verified evidence
	var factsCount int
	err := db.QueryRow("SELECT COUNT(*) FROM memories WHERE type = 'fact';").Scan(&factsCount)
	if err != nil {
		fmt.Fprintf(w, "Facts:\terror counting\n")
	} else {
		var factsWithVerifiedEvidence int
		err = db.QueryRow(`
			SELECT COUNT(DISTINCT m.id)
			FROM memories m
			INNER JOIN evidence e ON m.id = e.memory_id
			WHERE m.type = 'fact' AND e.verified = 1;
		`).Scan(&factsWithVerifiedEvidence)
		if err != nil {
			fmt.Fprintf(w, "Facts:\t%d (error checking verified evidence)\n", factsCount)
		} else {
			var percent float64
			if factsCount > 0 {
				percent = float64(factsWithVerifiedEvidence) / float64(factsCount) * 100
			}
			fmt.Fprintf(w, "Facts:\t%d (%.1f%% with verified evidence)\n", factsCount, percent)
		}
	}

	// Count other memory types
	types := []memory.Type{memory.Claim, memory.Plan, memory.Decision, memory.Constraint, memory.Observation, memory.Note, memory.Task}
	for _, memType := range types {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM memories WHERE type = ?;", string(memType)).Scan(&count)
		if err != nil {
			fmt.Fprintf(w, "%s:\terror counting\n", string(memType))
		} else {
			fmt.Fprintf(w, "%s:\t%d\n", string(memType), count)
		}
	}

	// Count superseded memories
	var supersededCount int
	err = db.QueryRow("SELECT COUNT(*) FROM memories WHERE superseded_by IS NOT NULL AND superseded_by != 0;").Scan(&supersededCount)
	if err != nil {
		fmt.Fprintf(w, "Superseded:\terror counting\n")
	} else {
		fmt.Fprintf(w, "Superseded:\t%d\n", supersededCount)
	}

	// Count potential conflicts - memories with same key but different content
	var conflictsCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM (
			SELECT key
			FROM memories
			WHERE key IS NOT NULL
			GROUP BY project_id, key
			HAVING COUNT(*) > 1
		);
	`).Scan(&conflictsCount)
	if err != nil {
		fmt.Fprintf(w, "Conflicts:\terror counting\n")
	} else {
		fmt.Fprintf(w, "Conflicts:\t%d (duplicate keys)\n", conflictsCount)
	}

	w.Flush()
	fmt.Println()
}

// printRecentDecisions prints the recent decisions section
func printRecentDecisions(db *sql.DB) {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 3️⃣  Recent Decisions (limit 5)                               │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tUpdated\tSummary\n")
	fmt.Fprintf(w, "--\t-------\t-------\n")

	rows, err := db.Query("SELECT id, updated_at, summary FROM memories WHERE type = 'decision' ORDER BY updated_at DESC LIMIT 5;")
	if err != nil {
		fmt.Fprintf(w, "Error querying decisions: %v\n", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var updatedAt time.Time
			var summary string
			err := rows.Scan(&id, &updatedAt, &summary)
			if err != nil {
				continue
			}

			// Truncate summary if too long
			truncatedSummary := summary
			if len(truncatedSummary) > 50 {
				truncatedSummary = truncatedSummary[:47] + "..."
			}

			fmt.Fprintf(w, "%d\t%s\t%s\n", id, updatedAt.Format("2006-01-02"), truncatedSummary)
		}
	}

	w.Flush()
	fmt.Println()
}

// printActiveConstraints prints the active constraints section
func printActiveConstraints(db *sql.DB) {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 4️⃣  Active Constraints (limit 5)                             │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tSummary\n")
	fmt.Fprintf(w, "--\t-------\n")

	rows, err := db.Query("SELECT id, summary FROM memories WHERE type = 'constraint' AND (superseded_by IS NULL OR superseded_by = 0) ORDER BY updated_at DESC LIMIT 5;")
	if err != nil {
		fmt.Fprintf(w, "Error querying constraints: %v\n", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var summary string
			err := rows.Scan(&id, &summary)
			if err != nil {
				continue
			}

			// Truncate summary if too long
			truncatedSummary := summary
			if len(truncatedSummary) > 60 {
				truncatedSummary = truncatedSummary[:57] + "..."
			}

			fmt.Fprintf(w, "%d\t%s\n", id, truncatedSummary)
		}
	}

	w.Flush()
	fmt.Println()
}

// printNeedsAttention prints the needs attention section
func printNeedsAttention(db *sql.DB) {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 5️⃣  Needs Attention / Suspicious Items (limit 5)             │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Type\tID\tReason\tSummary\n")
	fmt.Fprintf(w, "----\t--\t------\t-------\n")

	// Query for various suspicious items
	query := `
		SELECT m.id, m.type, 'Missing verified evidence' as reason, m.summary
		FROM memories m
		WHERE m.type = 'fact'
		AND NOT EXISTS (
			SELECT 1 FROM evidence e
			WHERE e.memory_id = m.id AND e.verified = 1
		)
		UNION ALL
		SELECT m.id, m.type, 'Claim with fact-like language' as reason, m.summary
		FROM memories m
		WHERE m.type = 'claim'
		AND (m.summary LIKE '%is%' OR m.summary LIKE '%are%' OR m.summary LIKE '%was%' OR m.summary LIKE '%were%')
		AND m.summary LIKE '%fact%'
		UNION ALL
		SELECT m.id, m.type, 'Contradicted by newer memory' as reason, m.summary
		FROM memories m
		WHERE m.superseded_by IS NOT NULL AND m.superseded_by != 0
		LIMIT 5;
	`

	rows, err := db.Query(query)
	if err != nil {
		fmt.Fprintf(w, "Error querying suspicious items: %v\n", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() && count < 5 {
			var id int64
			var memType string
			var reason string
			var summary string
			err := rows.Scan(&id, &memType, &reason, &summary)
			if err != nil {
				continue
			}

			// Truncate summary if too long
			truncatedSummary := summary
			if len(truncatedSummary) > 30 {
				truncatedSummary = truncatedSummary[:27] + "..."
			}

			fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", memType, id, reason, truncatedSummary)
			count++
		}
	}

	w.Flush()
	fmt.Println()
}

// printEnhancedTaskAnalytics prints the enhanced task analytics section with visualizations
func printEnhancedTaskAnalytics(analyticsService *analytics.TaskAnalytics, projectID string) {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 6️⃣  Task Analytics & Visualizations                          │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	// Get comprehensive task metrics using the analytics service
	metrics, err := analyticsService.GetTaskMetrics(projectID)
	if err != nil {
		fmt.Printf("Error getting task metrics: %v\n", err)
		return
	}

	// Print summary statistics
	fmt.Printf("📊 Summary:\n")
	fmt.Printf("   Total Tasks: %d\n", metrics.TotalTasks)
	fmt.Printf("   Completed: %d\n", metrics.CompletedTasks)
	fmt.Printf("   Incomplete: %d\n", metrics.IncompleteTasks)

	if metrics.TotalTasks > 0 {
		fmt.Printf("   Completion Rate: %.1f%%\n", metrics.CompletionRate)
		// Visualize completion rate
		fmt.Printf("   Progress: %s\n", metrics.VisualizeCompletionRate(30))
	}

	// Show average time to complete if available
	if metrics.AverageTimeToComplete > 0 {
		hours := metrics.AverageTimeToComplete.Hours()
		fmt.Printf("   Avg. Time to Complete: %.1f hours\n", hours)
	}

	fmt.Println()

	// Visualize section completion rates
	if len(metrics.TasksBySection) > 0 {
		fmt.Printf("📈 Completion by Section:\n")
		sectionVisuals := metrics.VisualizeSectionRates(20)
		for _, vis := range sectionVisuals {
			fmt.Printf("   %s\n", vis)
		}
		fmt.Println()
	}

	// Show completion trend visualization
	if len(metrics.CompletionTrend) > 0 {
		fmt.Printf("📉 Completion Trend (last 30 days):\n")
		trendLines := metrics.VisualizeCompletionTrend(10)
		for _, line := range trendLines {
			fmt.Printf("   %s\n", line)
		}
		fmt.Println()
	}

	// Show recent tasks
	fmt.Printf("📋 Recent Tasks:\n")
	db := analyticsService.DB
	recentRows, err := db.Query(`
		SELECT summary, detail, updated_at
		FROM memories
		WHERE project_id = ? AND type = 'task'
		ORDER BY updated_at DESC
		LIMIT 5;
	`, projectID)
	if err != nil {
		fmt.Printf("Error querying recent tasks: %v\n", err)
	} else {
		defer recentRows.Close()
		fmt.Printf("   %-30s %-10s %-12s\n", "Summary", "Status", "Updated")
		fmt.Printf("   %-30s %-10s %-12s\n", "-------", "------", "-------")
		for recentRows.Next() {
			var summary, detail, updatedAt string
			err := recentRows.Scan(&summary, &detail, &updatedAt)
			if err != nil {
				continue
			}

			// Determine status from detail
			status := "Incomplete"
			if strings.Contains(detail, "Completed: true") {
				status = "Complete"
			}

			// Truncate summary if too long
			truncatedSummary := summary
			if len(truncatedSummary) > 30 {
				truncatedSummary = truncatedSummary[:27] + "..."
			}

			fmt.Printf("   %-30s %-10s %-12s\n", truncatedSummary, status, updatedAt[:10]) // Just date part
		}
	}
	fmt.Println()
}

// printRecallEffectiveness prints the recall effectiveness section
func printRecallEffectiveness(db *sql.DB) {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 7️⃣  Recall Effectiveness                                     │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	// Check if recall metrics table exists
	var tableExists bool
	err := db.QueryRow("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='recall_metrics';").Scan(&tableExists)
	if err != nil || !tableExists {
		fmt.Println("Recall metrics: not yet available")
		fmt.Println("  (Enable with [metrics] enabled = true in config)")
		fmt.Println()
		return
	}

	// Check if there's any data
	var totalRecalls int
	err = db.QueryRow("SELECT COUNT(*) FROM recall_metrics;").Scan(&totalRecalls)
	if err != nil || totalRecalls == 0 {
		fmt.Println("No recall metrics recorded yet.")
		fmt.Println("  (Run queries with metrics enabled to collect data)")
		fmt.Println()
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Total recalls and 24h recalls
	var recalls24h int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM recall_metrics
		WHERE created_at >= datetime('now', '-24 hours');
	`).Scan(&recalls24h)
	if err != nil {
		recalls24h = 0
	}
	fmt.Fprintf(w, "Total recalls:\t%d (last 24h: %d)\n", totalRecalls, recalls24h)

	// Average memories per recall
	var avgMemories float64
	err = db.QueryRow("SELECT AVG(memory_count) FROM recall_metrics;").Scan(&avgMemories)
	if err != nil {
		fmt.Fprintf(w, "Avg memories/recall:\terror\n")
	} else {
		fmt.Fprintf(w, "Avg memories/recall:\t%.1f\n", avgMemories)
	}

	// Average tokens per recall
	var avgTokens float64
	err = db.QueryRow("SELECT AVG(total_tokens) FROM recall_metrics;").Scan(&avgTokens)
	if err != nil {
		fmt.Fprintf(w, "Avg tokens/recall:\terror\n")
	} else {
		fmt.Fprintf(w, "Avg tokens/recall:\t%.0f\n", avgTokens)
	}

	// Average duration
	var avgDuration float64
	err = db.QueryRow("SELECT AVG(duration_ms) FROM recall_metrics WHERE duration_ms IS NOT NULL;").Scan(&avgDuration)
	if err != nil {
		fmt.Fprintf(w, "Avg duration:\t\terror\n")
	} else {
		fmt.Fprintf(w, "Avg duration:\t\t%.1f ms\n", avgDuration)
	}

	// Query type distribution
	var emptyQueries, searchQueries int
	err = db.QueryRow("SELECT COUNT(*) FROM recall_metrics WHERE query_type = 'empty';").Scan(&emptyQueries)
	if err != nil {
		emptyQueries = 0
	}
	err = db.QueryRow("SELECT COUNT(*) FROM recall_metrics WHERE query_type = 'search';").Scan(&searchQueries)
	if err != nil {
		searchQueries = 0
	}
	fmt.Fprintf(w, "Query types:\t\tempty: %d, search: %d\n", emptyQueries, searchQueries)

	w.Flush()
	fmt.Println()

	// Top 5 most recalled memories
	fmt.Println("Top recalled memories:")
	topW := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(topW, "  ID\tCount\tSummary\n")
	fmt.Fprintf(topW, "  --\t-----\t-------\n")

	// Parse memory_ids JSON to count occurrences
	rows, err := db.Query(`
		SELECT memory_ids FROM recall_metrics WHERE memory_ids IS NOT NULL AND memory_ids != '[]';
	`)
	if err == nil {
		defer rows.Close()
		memoryRecallCounts := make(map[int64]int)
		for rows.Next() {
			var memoryIDsJSON string
			if err := rows.Scan(&memoryIDsJSON); err != nil {
				continue
			}
			var memoryIDs []int64
			if err := json.Unmarshal([]byte(memoryIDsJSON), &memoryIDs); err != nil {
				continue
			}
			for _, id := range memoryIDs {
				memoryRecallCounts[id]++
			}
		}

		// Sort by count and get top 5
		type memCount struct {
			ID    int64
			Count int
		}
		var counts []memCount
		for id, count := range memoryRecallCounts {
			counts = append(counts, memCount{ID: id, Count: count})
		}
		// Sort descending by count
		for i := 0; i < len(counts); i++ {
			for j := i + 1; j < len(counts); j++ {
				if counts[j].Count > counts[i].Count {
					counts[i], counts[j] = counts[j], counts[i]
				}
			}
		}

		shown := 0
		for _, mc := range counts {
			if shown >= 5 {
				break
			}
			// Get memory summary
			var summary string
			err := db.QueryRow("SELECT summary FROM memories WHERE id = ?;", mc.ID).Scan(&summary)
			if err != nil {
				summary = "(deleted)"
			}
			if len(summary) > 40 {
				summary = summary[:37] + "..."
			}
			fmt.Fprintf(topW, "  %d\t%d\t%s\n", mc.ID, mc.Count, summary)
			shown++
		}

		if shown == 0 {
			fmt.Fprintf(topW, "  (no data)\t\t\n")
		}
	}
	topW.Flush()
	fmt.Println()

	// Never-recalled memories count
	var neverRecalledCount int
	// Get all memory IDs that have been recalled
	recalledIDs := make(map[int64]bool)
	rows2, err := db.Query("SELECT memory_ids FROM recall_metrics WHERE memory_ids IS NOT NULL AND memory_ids != '[]';")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var memoryIDsJSON string
			if err := rows2.Scan(&memoryIDsJSON); err != nil {
				continue
			}
			var memoryIDs []int64
			if err := json.Unmarshal([]byte(memoryIDsJSON), &memoryIDs); err != nil {
				continue
			}
			for _, id := range memoryIDs {
				recalledIDs[id] = true
			}
		}
	}

	// Count memories not in the recalled set
	var totalMemories int
	err = db.QueryRow("SELECT COUNT(*) FROM memories;").Scan(&totalMemories)
	if err == nil {
		rows3, err := db.Query("SELECT id FROM memories;")
		if err == nil {
			defer rows3.Close()
			for rows3.Next() {
				var id int64
				if err := rows3.Scan(&id); err != nil {
					continue
				}
				if !recalledIDs[id] {
					neverRecalledCount++
				}
			}
		}
		fmt.Printf("Never recalled: %d of %d memories\n", neverRecalledCount, totalMemories)
	}
	fmt.Println()
}
