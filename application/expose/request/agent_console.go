package request

type AgentConsoleWrite struct {
	Message string `json:"message" validate:"required"`
}
