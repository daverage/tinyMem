package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/a-marczewski/tinymem/internal/app"
	"github.com/a-marczewski/tinymem/internal/config"
	"github.com/a-marczewski/tinymem/internal/memory"
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

	// Print dashboard header
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│                    tinyMem Dashboard                      │")
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

	// Section 6: Recall Effectiveness
	printRecallEffectiveness(dbConn)
}

// printHeaderSection prints the header/project status section
func printHeaderSection(projectRoot, tinyMemDir, dbPath string, db *sql.DB) {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 1️⃣  Header / Project Status                                 │")
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
	var lastActivity sql.NullTime
	var totalCount int
	err = db.QueryRow("SELECT MAX(updated_at), COUNT(*) FROM memories;").Scan(&lastActivity, &totalCount)
	if err != nil {
		fmt.Fprintf(w, "Last Activity:\terror retrieving\n")
		fmt.Fprintf(w, "Total Memories:\terror retrieving\n")
	} else {
		if lastActivity.Valid {
			fmt.Fprintf(w, "Last Activity:\t%s\n", lastActivity.Time.Format("2006-01-02 15:04:05"))
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
	fmt.Println("│ 2️⃣  Integrity Summary                                       │")
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
	types := []memory.Type{memory.Claim, memory.Plan, memory.Decision, memory.Constraint, memory.Observation, memory.Note}
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
	fmt.Println("│ 3️⃣  Recent Decisions (limit 5)                              │")
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
	fmt.Println("│ 4️⃣  Active Constraints (limit 5)                            │")
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
	fmt.Println("│ 5️⃣  Needs Attention / Suspicious Items (limit 5)            │")
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

// printRecallEffectiveness prints the recall effectiveness section
func printRecallEffectiveness(db *sql.DB) {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 6️⃣  Recall Effectiveness                                    │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	// Check if recall metrics table exists
	var tableExists bool
	err := db.QueryRow("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='recall_metrics';").Scan(&tableExists)
	if err != nil || !tableExists {
		fmt.Println("Recall metrics: not enabled")
		fmt.Println()
		return
	}

	// If table exists, try to get some recall metrics
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Top recalled memory IDs
	var topRecalledID int64
	var recallCount int
	err = db.QueryRow(`
		SELECT memory_id, COUNT(*) as recall_count
		FROM recall_metrics
		GROUP BY memory_id
		ORDER BY recall_count DESC
		LIMIT 1;
	`).Scan(&topRecalledID, &recallCount)

	if err != nil {
		fmt.Fprintf(w, "Top recalled memory: \tN/A\n")
	} else {
		fmt.Fprintf(w, "Top recalled memory: \tID %d (%d times)\n", topRecalledID, recallCount)
	}

	// Memories never recalled in last 30 days
	var neverRecalledCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM memories m
		LEFT JOIN recall_metrics rm ON m.id = rm.memory_id
		WHERE rm.memory_id IS NULL;
	`).Scan(&neverRecalledCount)

	if err != nil {
		fmt.Fprintf(w, "Never recalled: \t\tError\n")
	} else {
		fmt.Fprintf(w, "Never recalled: \t\t%d memories\n", neverRecalledCount)
	}

	// Average recall rate
	var avgRecallRate float64
	err = db.QueryRow(`
		SELECT AVG(recall_rate)
		FROM (
			SELECT CAST(COUNT(rm.memory_id) AS REAL) as recall_rate
			FROM memories m
			LEFT JOIN recall_metrics rm ON m.id = rm.memory_id
			GROUP BY m.id
		);
	`).Scan(&avgRecallRate)

	if err != nil {
		fmt.Fprintf(w, "Avg. recall rate: \tError\n")
	} else {
		fmt.Fprintf(w, "Avg. recall rate: \t%.2f%%\n", avgRecallRate*100)
	}

	w.Flush()
	fmt.Println()
}