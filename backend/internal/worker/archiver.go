package worker

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
)

type ArchivedRoomPayload struct {
	RoomState domain.RoomState `json:"state"`
	Messages  []domain.Message `json:"messages"`
	Memories  []domain.Memory  `json:"memories"`
}

func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func RunArchivalSweep(
	ctx context.Context,
	roomRepo domain.RoomRepository,
	messageRepo domain.MessageRepository,
	memoryRepo domain.MemoryRepository,
	storageProvider domain.StorageProvider,
	inactiveDuration time.Duration,
	roomID string,
) error {
	if roomID != "" {
		log.Printf("[Archiver]: Starting room archival sweep for specific room %s...", roomID)
	} else {
		log.Printf("[Archiver]: Starting room archival sweep for inactive rooms (> %v)...", inactiveDuration)
	}

	// Ensure destination bucket exists first
	if err := storageProvider.EnsureBucketExists(ctx); err != nil {
		return fmt.Errorf("failed to ensure archival S3 bucket: %w", err)
	}

	// 1. Query candidate rooms
	rooms, err := roomRepo.ListRoomsForArchival(ctx, inactiveDuration, roomID)
	if err != nil {
		return fmt.Errorf("failed to list rooms for archival: %w", err)
	}

	if len(rooms) == 0 {
		log.Println("[Archiver]: No inactive rooms found. Sweep complete.")
		return nil
	}

	log.Printf("[Archiver]: Found %d candidate room(s) for archival.", len(rooms))

	for _, room := range rooms {
		log.Printf("[Archiver]: Processing room %s (Tenant: %s)...", room.ID, room.TenantID)

		// 2. Lock room status to archiving
		err := roomRepo.UpdateStatus(ctx, room.ID, domain.RoomStatusArchiving)
		if err != nil {
			log.Printf("[Archiver Error]: failed to update room %s status to archiving: %v", room.ID, err)
			continue
		}

		// 3. Compile Room State
		state, err := roomRepo.GetState(ctx, room.ID)
		if err != nil {
			log.Printf("[Archiver Error]: failed to fetch state for room %s: %v. Rolling back status to active...", room.ID, err)
			_ = roomRepo.UpdateStatus(ctx, room.ID, domain.RoomStatusActive)
			continue
		}
		if state == nil {
			state = &domain.RoomState{RoomID: room.ID, State: json.RawMessage("{}")}
		}

		// 4. Retrieve chronological messages (limit 100,000 for full transcript)
		messages, err := messageRepo.ListByRoom(ctx, room.ID, 100000, 0)
		if err != nil {
			log.Printf("[Archiver Error]: failed to list messages for room %s: %v. Rolling back status to active...", room.ID, err)
			_ = roomRepo.UpdateStatus(ctx, room.ID, domain.RoomStatusActive)
			continue
		}

		// 5. Retrieve memories
		memories, err := memoryRepo.ListByRoom(ctx, room.ID)
		if err != nil {
			log.Printf("[Archiver Error]: failed to list memories for room %s: %v. Rolling back status to active...", room.ID, err)
			_ = roomRepo.UpdateStatus(ctx, room.ID, domain.RoomStatusActive)
			continue
		}

		// 6. Construct payload
		payload := ArchivedRoomPayload{
			RoomState: *state,
			Messages:  messages,
			Memories:  memories,
		}

		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[Archiver Error]: failed to marshal archive payload for room %s: %v. Rolling back status...", room.ID, err)
			_ = roomRepo.UpdateStatus(ctx, room.ID, domain.RoomStatusActive)
			continue
		}

		// 7. Compress JSON payload using gzip
		compressed, err := compressGzip(jsonBytes)
		if err != nil {
			log.Printf("[Archiver Error]: failed to compress archive payload for room %s: %v. Rolling back status...", room.ID, err)
			_ = roomRepo.UpdateStatus(ctx, room.ID, domain.RoomStatusActive)
			continue
		}

		// 8. Upload compressed object to S3 (MinIO)
		s3Key := fmt.Sprintf("tenants/%s/rooms/%s_archive.json.gz", room.TenantID, room.ID)
		path, err := storageProvider.PutObject(ctx, s3Key, compressed)
		if err != nil {
			log.Printf("[Archiver Error]: failed to upload archive to S3 for room %s: %v. Rolling back status...", room.ID, err)
			_ = roomRepo.UpdateStatus(ctx, room.ID, domain.RoomStatusActive)
			continue
		}

		// 9. Execute transaction: purge logs and memories, update room to archived with path
		err = roomRepo.ArchiveRoomTx(ctx, room.ID, path)
		if err != nil {
			log.Printf("[Archiver Error]: database transaction failed to archive room %s: %v. Rolling back status...", room.ID, err)
			_ = roomRepo.UpdateStatus(ctx, room.ID, domain.RoomStatusActive)
			continue
		}

		log.Printf("[Archiver]: Room %s successfully archived to %s and purged from Postgres.", room.ID, path)
	}

	log.Println("[Archiver]: Archival sweep complete.")
	return nil
}
