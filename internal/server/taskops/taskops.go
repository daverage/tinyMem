package taskops

import (
	"fmt"

	"github.com/daverage/tinymem/internal/tasks"
)

// Operator centralizes every mutation to tinyTasks.md through the shared TaskManager.
type Operator struct {
	manager *tasks.TaskManager
	service *tasks.Service
}

// New creates an Operator that uses manager for file mutations and service to sync memory.
func New(manager *tasks.TaskManager, service *tasks.Service) *Operator {
	return &Operator{
		manager: manager,
		service: service,
	}
}

// List returns the parsed tasks in file order.
func (o *Operator) List() ([]*tasks.Task, error) {
	return o.manager.List()
}

// Add wraps TaskManager.AddTask and synchronizes memory afterwards.
func (o *Operator) Add(def tasks.TaskDefinition) (*tasks.Task, error) {
	task, err := o.manager.AddTask(def)
	if err != nil {
		return nil, err
	}
	if err := o.sync(); err != nil {
		return nil, err
	}
	return task, nil
}

// Update wraps TaskManager.UpdateTask and synchronizes memory afterwards.
func (o *Operator) Update(taskID string, update tasks.TaskUpdate) (*tasks.Task, error) {
	task, err := o.manager.UpdateTask(taskID, update)
	if err != nil {
		return nil, err
	}
	if err := o.sync(); err != nil {
		return nil, err
	}
	return task, nil
}

// Complete marks every checkbox within a task as done and syncs memory.
func (o *Operator) Complete(taskID string) (*tasks.Task, error) {
	task, err := o.manager.Complete(taskID)
	if err != nil {
		return nil, err
	}
	if err := o.sync(); err != nil {
		return nil, err
	}
	return task, nil
}

func (o *Operator) sync() error {
	if o.service == nil {
		return nil
	}
	if err := o.service.SyncTasksFromFile(""); err != nil {
		return fmt.Errorf("failed to sync tasks after mutation: %w", err)
	}
	return nil
}
