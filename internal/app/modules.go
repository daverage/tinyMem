package app

import (
	"context"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/doctor"
	"github.com/daverage/tinymem/internal/memory"
	"github.com/daverage/tinymem/internal/storage"
	"go.uber.org/zap"
)

// CoreModule holds the core application components
type CoreModule struct {
	Config *config.Config
	Logger *zap.Logger
	DB     *storage.DB
}

// ProjectModule holds project-specific information
type ProjectModule struct {
	Path string
	ID   string
}

// ServerModule holds server-specific information
type ServerModule struct {
	Mode doctor.ServerMode
}

// App holds the core components of the application with better separation of concerns.
type App struct {
	Core      CoreModule
	Project   ProjectModule
	Server    ServerModule
	Memory    *memory.Service
	Ctx       context.Context
	Cancel    context.CancelFunc
}