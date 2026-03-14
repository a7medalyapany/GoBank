package worker

// Queue priority names — used when enqueuing tasks (distributor side)
// and when configuring the server (processor side).
const (
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueLow      = "low"
)

// Task type constants — one per task.
// Add new constants here as you add new task files.
const (
	TaskSendVerifyEmail = "task:send_verify_email"
)

// PayloadSendVerifyEmail carries the minimum data needed to process the task.
// The worker re-fetches full records from the DB at process time — never
// store the full user object here to avoid stale data.
type PayloadSendVerifyEmail struct {
	Username string `json:"username"`
}