package server

import (
	"os"
	"strings"

	"github.com/daverage/tinymem/internal/tasks"
)

// TaskSubtaskInput mirrors the JSON payload used by MCP task tools.
type TaskSubtaskInput struct {
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// BuildSubtaskSpecs converts inputs into TaskManager subtask specs.
func BuildSubtaskSpecs(inputs []TaskSubtaskInput) []tasks.SubtaskSpec {
	var specs []tasks.SubtaskSpec
	for _, sub := range inputs {
		title := strings.TrimSpace(sub.Title)
		if title == "" {
			continue
		}
		specs = append(specs, tasks.SubtaskSpec{
			Title:     title,
			Completed: sub.Completed,
		})
	}
	return specs
}

// BuildNextTaskSignals derives observational metadata for the next unfinished task.
func BuildNextTaskSignals(parsedTasks []*tasks.Task) map[string]interface{} {
	for _, task := range parsedTasks {
		if task.Completed {
			continue
		}

		return map[string]interface{}{
			"steps_total":      task.StepsTotal,
			"steps_done":       task.StepsDone,
			"has_subtasks":     task.StepsTotal > 0,
			"title_word_count": len(strings.Fields(task.Title)),
			"section_title":    task.Section,
			"is_flat_task":     task.StepsTotal == 0,
		}
	}

	return nil
}

// BuildTaskAuthorityPayload produces the response body for memory_check_task_authority.
func BuildTaskAuthorityPayload(taskManager *tasks.TaskManager) (map[string]interface{}, error) {
	exists, err := taskManager.Exists()
	if err != nil {
		return nil, err
	}

	uncheckedTasks := []string{}
	authorization := "create_allowed"
	var taskSignals map[string]interface{}

	if exists {
		taskList, err := taskManager.List()
		if err != nil {
			if os.IsNotExist(err) {
				exists = false
			} else {
				return nil, err
			}
		}
		if exists {
			for _, task := range taskList {
				if !task.Completed {
					uncheckedTasks = append(uncheckedTasks, task.Title)
				}
			}

			if len(uncheckedTasks) > 0 {
				authorization = "authorized"
				taskSignals = BuildNextTaskSignals(taskList)
			} else {
				authorization = "unauthorized"
			}
		}
	}

	result := map[string]interface{}{
		"exists":          exists,
		"unchecked_tasks": uncheckedTasks,
		"authorization":   authorization,
		"task_count":      len(uncheckedTasks),
	}

	if exists && authorization == "authorized" && len(uncheckedTasks) > 0 {
		result["next_task"] = uncheckedTasks[0]
	}
	if taskSignals != nil {
		result["task_signals"] = taskSignals
	}

	return result, nil
}
