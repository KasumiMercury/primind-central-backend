package task

//go:generate mockgen -destination=mock_service_task.go -package=task github.com/KasumiMercury/primind-central-backend/internal/task/app/task CreateTaskUseCase,GetTaskUseCase,ListActiveTasksUseCase,UpdateTaskUseCase,DeleteTaskUseCase
