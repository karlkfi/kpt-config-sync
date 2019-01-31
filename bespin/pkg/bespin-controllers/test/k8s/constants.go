package k8s

import "time"

// nolint
const (
	ControllerMaxConcurrentReconciles = 5
	ReconcileDeadline                 = 59 * time.Minute
	ReconcileReenqueuePeriod          = 1 * time.Hour
	Creating                          = "Creating"
	CreatingMessage                   = "Creation in progress"
	Created                           = "Created"
	CreatedMessage                    = "Successfully created"
	CreateFailed                      = "CreateFailed"
	CreateFailedMessageTmpl           = "Create call failed: %v"
	Updating                          = "Updating"
	UpdatingMessage                   = "Update in progress"
	Updated                           = "Updated"
	UpdatedMessage                    = "Successfully updated"
	UpdateFailed                      = "UpdateFailed"
	UpdateFailedMessageTmpl           = "Update call failed: %v"
	Deleting                          = "Deleting"
	DeletingMessage                   = "Deletion in progress"
	Deleted                           = "Deleted"
	DeletedMessage                    = "Successfully deleted"
	DeleteFailed                      = "DeleteFailed"
	DeleteFailedMessageTmpl           = "Delete call failed: %v"
)
