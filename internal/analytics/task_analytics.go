package analytics

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/daverage/tinymem/internal/memory"
)

// TaskAnalytics provides methods for analyzing task performance metrics
type TaskAnalytics struct {
	DB *sql.DB
}

// NewTaskAnalytics creates a new TaskAnalytics instance
func NewTaskAnalytics(db *sql.DB) *TaskAnalytics {
	return &TaskAnalytics{
		DB: db,
	}
}

// TaskMetrics represents various task performance metrics
type TaskMetrics struct {
	TotalTasks        int                 `json:"total_tasks"`
	CompletedTasks    int                 `json:"completed_tasks"`
	IncompleteTasks   int                 `json:"incomplete_tasks"`
	CompletionRate    float64             `json:"completion_rate"`
	AverageTimeToComplete time.Duration   `json:"average_time_to_complete"` // in hours
	TasksBySection    map[string]SectionMetrics `json:"tasks_by_section"`
	CompletionTrend   []CompletionTrendPoint `json:"completion_trend"` // for visualization
}

// SectionMetrics represents metrics for a specific section
type SectionMetrics struct {
	Total     int     `json:"total"`
	Completed int     `json:"completed"`
	Rate      float64 `json:"rate"`
}

// CompletionTrendPoint represents a point in the completion trend over time
type CompletionTrendPoint struct {
	Date        string  `json:"date"`
	Completed   int     `json:"completed"`
	Total       int     `json:"total"`
	Percentage  float64 `json:"percentage"`
}

// GetTaskMetrics calculates and returns comprehensive task metrics
func (ta *TaskAnalytics) GetTaskMetrics(projectID string) (*TaskMetrics, error) {
	metrics := &TaskMetrics{
		TasksBySection: make(map[string]SectionMetrics),
	}

	// Get total task count
	err := ta.DB.QueryRow("SELECT COUNT(*) FROM memories WHERE project_id = ? AND type = ?", projectID, memory.Task).Scan(&metrics.TotalTasks)
	if err != nil {
		return nil, fmt.Errorf("failed to get total task count: %w", err)
	}

	// Get completed task count
	err = ta.DB.QueryRow(`
		SELECT COUNT(*)
		FROM memories
		WHERE project_id = ? AND type = ? AND detail LIKE '%Completed: true%';`,
		projectID, memory.Task).Scan(&metrics.CompletedTasks)
	if err != nil {
		return nil, fmt.Errorf("failed to get completed task count: %w", err)
	}

	metrics.IncompleteTasks = metrics.TotalTasks - metrics.CompletedTasks

	if metrics.TotalTasks > 0 {
		metrics.CompletionRate = float64(metrics.CompletedTasks) / float64(metrics.TotalTasks) * 100
	}

	// Calculate average time to complete
	avgHours, err := ta.calculateAverageTimeToComplete(projectID)
	if err != nil {
		// Log the error but don't fail the whole operation
		fmt.Printf("Warning: failed to calculate average time to complete: %v\n", err)
	} else {
		metrics.AverageTimeToComplete = time.Duration(avgHours * float64(time.Hour))
	}

	// Get metrics by section
	sectionMetrics, err := ta.getTasksBySection(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by section: %w", err)
	}
	metrics.TasksBySection = sectionMetrics

	// Get completion trend
	trend, err := ta.getCompletionTrend(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get completion trend: %w", err)
	}
	metrics.CompletionTrend = trend

	return metrics, nil
}

// calculateAverageTimeToComplete calculates the average time it takes to complete tasks
func (ta *TaskAnalytics) calculateAverageTimeToComplete(projectID string) (float64, error) {
	// This is a simplified calculation assuming we can determine completion time
	// from the updated_at timestamp. A more sophisticated implementation would
	// track when tasks were created vs when they were marked complete.

	rows, err := ta.DB.Query(`
		SELECT created_at, updated_at
		FROM memories
		WHERE project_id = ? AND type = ? AND detail LIKE '%Completed: true%'
		AND created_at IS NOT NULL AND updated_at IS NOT NULL
	`, projectID, memory.Task)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var totalDuration float64
	var count int

	for rows.Next() {
		var createdAt, updatedAt string
		err := rows.Scan(&createdAt, &updatedAt)
		if err != nil {
			continue
		}

		// Parse timestamps
		createdTime, err := parseTimestamp(createdAt)
		if err != nil {
			continue
		}

		updatedTime, err := parseTimestamp(updatedAt)
		if err != nil {
			continue
		}

		// Calculate duration in hours
		duration := updatedTime.Sub(createdTime).Hours()
		totalDuration += duration
		count++
	}

	if count == 0 {
		return 0, nil
	}

	return totalDuration / float64(count), nil
}

// getTasksBySection gets task metrics grouped by section
func (ta *TaskAnalytics) getTasksBySection(projectID string) (map[string]SectionMetrics, error) {
	rows, err := ta.DB.Query(`
		SELECT
			COALESCE(source, 'Unknown Section') as section,
			COUNT(*) as total,
			SUM(CASE WHEN detail LIKE '%Completed: true%' THEN 1 ELSE 0 END) as completed
		FROM memories
		WHERE project_id = ? AND type = ?
		GROUP BY COALESCE(source, 'Unknown Section')
	`, projectID, memory.Task)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sectionMetrics := make(map[string]SectionMetrics)

	for rows.Next() {
		var section string
		var total, completed int

		err := rows.Scan(&section, &total, &completed)
		if err != nil {
			continue
		}

		rate := 0.0
		if total > 0 {
			rate = float64(completed) / float64(total) * 100
		}

		sectionMetrics[section] = SectionMetrics{
			Total:     total,
			Completed: completed,
			Rate:      rate,
		}
	}

	return sectionMetrics, nil
}

// getCompletionTrend gets the completion trend over time
func (ta *TaskAnalytics) getCompletionTrend(projectID string) ([]CompletionTrendPoint, error) {
	// Get daily completion counts for the last 30 days
	rows, err := ta.DB.Query(`
		SELECT
			date(updated_at) as day,
			SUM(CASE WHEN detail LIKE '%Completed: true%' THEN 1 ELSE 0 END) as completed,
			COUNT(*) as total
		FROM memories
		WHERE project_id = ? AND type = ?
		AND updated_at >= date('now', '-30 days')
		GROUP BY date(updated_at)
		ORDER BY day
	`, projectID, memory.Task)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trend []CompletionTrendPoint

	for rows.Next() {
		var date string
		var completed, total int

		err := rows.Scan(&date, &completed, &total)
		if err != nil {
			continue
		}

		percentage := 0.0
		if total > 0 {
			percentage = float64(completed) / float64(total) * 100
		}

		trend = append(trend, CompletionTrendPoint{
			Date:       date,
			Completed:  completed,
			Total:      total,
			Percentage: percentage,
		})
	}

	return trend, nil
}

// parseTimestamp parses a timestamp string into time.Time
func parseTimestamp(timestamp string) (time.Time, error) {
	// Try different common timestamp formats
	formats := []string{
		"2006-01-02 15:04:05.999999-07:00",
		"2006-01-02 15:04:05.999999+00:00",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestamp); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", timestamp)
}

// GetTaskSummary provides a high-level summary of task status
func (ta *TaskAnalytics) GetTaskSummary(projectID string) (map[string]interface{}, error) {
	var total, completed, incomplete int

	err := ta.DB.QueryRow(`
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN detail LIKE '%Completed: true%' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN detail LIKE '%Completed: false%' OR detail NOT LIKE '%Completed%' THEN 1 ELSE 0 END) as incomplete
		FROM memories
		WHERE project_id = ? AND type = ?
	`, projectID, memory.Task).Scan(&total, &completed, &incomplete)

	if err != nil {
		return nil, fmt.Errorf("failed to get task summary: %w", err)
	}

	// If the above query doesn't work properly, use individual queries
	if total == 0 {
		err = ta.DB.QueryRow("SELECT COUNT(*) FROM memories WHERE project_id = ? AND type = ?", projectID, memory.Task).Scan(&total)
		if err != nil {
			return nil, fmt.Errorf("failed to get total task count: %w", err)
		}

		err = ta.DB.QueryRow("SELECT COUNT(*) FROM memories WHERE project_id = ? AND type = ? AND detail LIKE '%Completed: true%'", projectID, memory.Task).Scan(&completed)
		if err != nil {
			return nil, fmt.Errorf("failed to get completed task count: %w", err)
		}

		incomplete = total - completed
	}

	overallRate := 0.0
	if total > 0 {
		overallRate = float64(completed) / float64(total) * 100
	}

	summary := map[string]interface{}{
		"total_tasks":       total,
		"completed_tasks":   completed,
		"incomplete_tasks":  incomplete,
		"overall_rate":      overallRate,
		"project_id":        projectID,
	}

	return summary, nil
}

// VisualizeCompletionRate generates a text-based bar chart for completion rate
func (tm *TaskMetrics) VisualizeCompletionRate(width int) string {
	if tm.TotalTasks == 0 {
		return "[No tasks]"
	}

	// Calculate the number of filled positions in the bar
	filled := int((tm.CompletionRate / 100) * float64(width))

	var sb strings.Builder
	sb.WriteString("[")

	// Fill completed portion
	for i := 0; i < filled; i++ {
		sb.WriteString("█")
	}

	// Fill remaining portion
	for i := filled; i < width; i++ {
		sb.WriteString("░")
	}

	sb.WriteString(fmt.Sprintf("] %.1f%% (%d/%d)", tm.CompletionRate, tm.CompletedTasks, tm.TotalTasks))

	return sb.String()
}

// VisualizeSectionRates generates a text-based visualization for section completion rates
func (tm *TaskMetrics) VisualizeSectionRates(width int) []string {
	var visualizations []string

	for section, metrics := range tm.TasksBySection {
		// Calculate the number of filled positions in the bar
		filled := int((metrics.Rate / 100) * float64(width))

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%-20s [", truncateString(section, 18)))

		// Fill completed portion
		for i := 0; i < filled; i++ {
			sb.WriteString("█")
		}

		// Fill remaining portion
		for i := filled; i < width; i++ {
			sb.WriteString("░")
		}

		sb.WriteString(fmt.Sprintf("] %.1f%% (%d/%d)", metrics.Rate, metrics.Completed, metrics.Total))

		visualizations = append(visualizations, sb.String())
	}

	return visualizations
}

// VisualizeCompletionTrend generates a simple text-based line chart for completion trend
func (tm *TaskMetrics) VisualizeCompletionTrend(height int) []string {
	if len(tm.CompletionTrend) == 0 {
		return []string{"No trend data available"}
	}

	// Find min and max percentages for scaling
	minPercent := tm.CompletionTrend[0].Percentage
	maxPercent := tm.CompletionTrend[0].Percentage

	for _, point := range tm.CompletionTrend {
		if point.Percentage < minPercent {
			minPercent = point.Percentage
		}
		if point.Percentage > maxPercent {
			maxPercent = point.Percentage
		}
	}

	// Handle edge case where all values are the same
	if minPercent == maxPercent {
		maxPercent = minPercent + 10 // Add some range for visualization
	}

	var lines []string

	// Generate the chart from top to bottom
	for y := height - 1; y >= 0; y-- {
		var line strings.Builder

		// Add Y-axis labels on the left
		valueAtThisHeight := minPercent + (maxPercent-minPercent)*float64(y)/float64(height-1)
		line.WriteString(fmt.Sprintf("%6.0f |", valueAtThisHeight))

		// Draw the chart points
		for _, point := range tm.CompletionTrend {
			normalizedValue := (point.Percentage - minPercent) / (maxPercent - minPercent)
			normalizedHeight := normalizedValue * float64(height-1)

			// Check if this position should have a character
			if int(normalizedHeight) >= y && int(normalizedHeight) <= y+1 {
				line.WriteString("*")
			} else {
				line.WriteString(" ")
			}
		}

		lines = append(lines, line.String())
	}

	// Add X-axis labels
	var xAxis strings.Builder
	xAxis.WriteString("       ")
	for i, point := range tm.CompletionTrend {
		if i%max(1, len(tm.CompletionTrend)/5) == 0 { // Label roughly every 1/5th of the way
			dateParts := strings.Split(point.Date, "-")
			if len(dateParts) >= 2 {
				xAxis.WriteString(dateParts[1] + "/" + dateParts[2])
			} else {
				xAxis.WriteString(point.Date[len(point.Date)-5:]) // Last 5 chars
			}
		} else {
			xAxis.WriteString(strings.Repeat(" ", len(point.Date)-2)) // Approximate spacing
		}
	}
	lines = append(lines, xAxis.String())

	return lines
}

// truncateString truncates a string to the specified length, adding "..." if truncated
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	if maxLength <= 3 {
		return s[:maxLength]
	}
	return s[:maxLength-3] + "..."
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
