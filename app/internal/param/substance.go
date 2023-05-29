package param

type SubstanceTasks struct {
	TaskID int64 `json:"task_id" validate:"gt=0"`
}

type SubstanceNotice struct {
	MinionID int64 `json:"minion_id"`
}
