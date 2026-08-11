package domain

import "context"

type RoomEvent struct {
	ID        string `db:"id" json:"id"`
	RoomID    string `db:"room_id" json:"room_id"`
	EventType string `db:"event_type" json:"event_type"`
	Payload   string `db:"payload" json:"payload"`
	CreatedAt string `db:"created_at" json:"created_at"`
}

type EventRepository interface {
	Create(ctx context.Context, event *RoomEvent) error
	ListByRoom(ctx context.Context, roomID string, limit int) ([]RoomEvent, error)
}

type EventUsecase interface {
	LogEvent(ctx context.Context, tenantID string, roomID string, eventType string, payload any) (*RoomEvent, error)
	ListEvents(ctx context.Context, tenantID string, roomID string, limit int) ([]RoomEvent, error)
}
