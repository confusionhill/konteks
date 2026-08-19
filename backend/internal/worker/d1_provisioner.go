package worker

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
)

type D1Provisioner struct {
	authRepo domain.AuthRepository
	d1Store  domain.D1Store
}

func NewD1Provisioner(authRepo domain.AuthRepository, d1Store domain.D1Store) *D1Provisioner {
	return &D1Provisioner{
		authRepo: authRepo,
		d1Store:  d1Store,
	}
}

func (w *D1Provisioner) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[D1Provisioner] Shutting down...")
			return
		case <-ticker.C:
			w.processPendingJobs(ctx)
		}
	}
}

func (w *D1Provisioner) processPendingJobs(ctx context.Context) {
	jobs, err := w.authRepo.GetPendingD1Jobs(ctx)
	if err != nil {
		log.Printf("[D1Provisioner] Failed to fetch pending jobs: %v", err)
		return
	}

	for _, job := range jobs {
		w.provisionJob(ctx, job)
	}
}

func (w *D1Provisioner) provisionJob(ctx context.Context, job domain.D1InfrastructureJob) {
	log.Printf("[D1Provisioner] Provisioning project %s (DB Name: %s)", job.ProjectID, job.D1Name)

	dbUUID, err := w.d1Store.CreateDatabase(ctx, job.D1Name)
	if err != nil {
		w.handleProvisioningError(ctx, job, fmt.Errorf("failed to create database: %w", err))
		return
	}

	// Initialize D1 Schema
	queries := `
		CREATE TABLE IF NOT EXISTS rooms (id TEXT PRIMARY KEY, status TEXT NOT NULL, default_policy TEXT, external_user_id TEXT, archive_s3_path TEXT);
		CREATE TABLE IF NOT EXISTS messages (id TEXT PRIMARY KEY, room_id TEXT NOT NULL, role TEXT NOT NULL, content TEXT NOT NULL, sequence INTEGER NOT NULL, created_at TEXT NOT NULL, tokens_input INTEGER, tokens_output INTEGER, cost_total REAL, is_encrypted BOOLEAN, encryption_metadata TEXT);
		CREATE INDEX IF NOT EXISTS idx_messages_room_id_sequence ON messages (room_id, sequence);
		CREATE TABLE IF NOT EXISTS room_state (room_id TEXT PRIMARY KEY, summary TEXT, active_topic TEXT, state TEXT, "references" TEXT, entities TEXT, last_summarized_seq INTEGER, updated_at TEXT);
		CREATE TABLE IF NOT EXISTS memories (id TEXT PRIMARY KEY, project_id TEXT, room_id TEXT, external_user_id TEXT, content TEXT NOT NULL, embedding TEXT, metadata TEXT, created_at TEXT);
	`
	
	_, err = w.d1Store.Query(ctx, dbUUID, queries)
	if err != nil {
		w.handleProvisioningError(ctx, job, fmt.Errorf("failed to initialize schema: %w", err))
		return
	}

	err = w.authRepo.UpdateD1InfrastructureStatus(ctx, job.ProjectID, &dbUUID, "ready", nil, job.RetryCount)
	if err != nil {
		log.Printf("[D1Provisioner] Failed to update status to ready for project %s: %v", job.ProjectID, err)
	} else {
		log.Printf("[D1Provisioner] Successfully provisioned D1 for project %s", job.ProjectID)
	}
}

func (w *D1Provisioner) handleProvisioningError(ctx context.Context, job domain.D1InfrastructureJob, err error) {
	errMsg := err.Error()
	log.Printf("[D1Provisioner] Error for project %s: %v", job.ProjectID, errMsg)

	// If it's a known non-retriable error (e.g. Quota Exceeded from Cloudflare), fail immediately
	if strings.Contains(errMsg, "Quota") || strings.Contains(errMsg, "quota") || strings.Contains(errMsg, "limit") {
		w.authRepo.UpdateD1InfrastructureStatus(ctx, job.ProjectID, nil, "failed", &errMsg, job.RetryCount)
		return
	}

	job.RetryCount++
	if job.RetryCount >= 3 {
		log.Printf("[D1Provisioner] Max retries reached for project %s. Marking as failed.", job.ProjectID)
		w.authRepo.UpdateD1InfrastructureStatus(ctx, job.ProjectID, nil, "failed", &errMsg, job.RetryCount)
	} else {
		// Just update the retry count and keep it pending for the next tick
		w.authRepo.UpdateD1InfrastructureStatus(ctx, job.ProjectID, nil, "pending", &errMsg, job.RetryCount)
	}
}
