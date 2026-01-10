package messaging

import "github.com/google/uuid"

type NotificationType string

type NotificationPayload struct {
	Type    NotificationType `json:"type"`
	UserID  uuid.UUID        `json:"user_id"`
	Message string           `json:"message"`
	Data    interface{}      `json:"data,omitempty"`
}
