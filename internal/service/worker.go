package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rodrigomsrocha/code-vault/internal/storage"
)

type Worker struct {
	snippetRepo *storage.SnippetRepository
	seaweedfs   *storage.SeaweedFS
	interval    time.Duration
	stopChan    chan struct{}
	wg          sync.WaitGroup
	running     bool
	mu          sync.Mutex
}

func NewWorker(snippetRepo *storage.SnippetRepository, sfs *storage.SeaweedFS, interval time.Duration) *Worker {
	return &Worker{
		snippetRepo: snippetRepo,
		seaweedfs:   sfs,
		interval:    interval,
		stopChan:    make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("worker already running")
	}
	w.running = true
	w.mu.Unlock()

	log.Println("🔄 Expiration worker started (interval:", w.interval, ")")

	w.wg.Add(1)
	go w.run(ctx)

	return nil
}

func (w *Worker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	log.Println("🛑 Stopping expiration worker...")
	close(w.stopChan)

	w.wg.Wait()
	w.running = false
	log.Println("✅ Expiration worker stopped")
}

func (w *Worker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.running
}

func (w *Worker) run(ctx context.Context) {
	defer w.wg.Done()

	w.cleanup(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.cleanup(ctx)
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) cleanup(ctx context.Context) {
	log.Println("🧹 Running cleanup task...")

	expiredSnippets, err := w.snippetRepo.GetExpiredSnippets(ctx)
	if err != nil {
		log.Printf("❌ Error fetching expired snippets: %v\n", err)
		return
	}

	if len(expiredSnippets) == 0 {
		log.Println("✨ No expired snippets found")
		return
	}

	log.Printf("📋 Found %d expired snippet(s)\n", len(expiredSnippets))

	deleted := 0

	for _, snippet := range expiredSnippets {
		if err := w.deleteSnippet(ctx, snippet.ID); err != nil {
			log.Printf("❌ Error deleting snippet %s: %v\n", snippet.ID, err)
			continue
		}

		deleted++
	}

	w.cleanupOrphanedVersions(ctx)
	log.Printf("✅ Cleanup complete: %d snippets deleted\n", deleted)
}

func (w *Worker) deleteSnippet(ctx context.Context, snippetID uuid.UUID) error {
	versions, err := w.snippetRepo.GetAllVersionsForSnippet(ctx, snippetID)
	if err != nil {
		return fmt.Errorf("failed to get versions: %w", err)
	}

	if err := w.snippetRepo.DeleteSnippet(ctx, snippetID); err != nil {
		return err
	}

	for _, version := range versions {
		w.tryDeleteOrphanedFile(ctx, version.ContentHash)
	}

	return nil
}

func (w *Worker) tryDeleteOrphanedFile(ctx context.Context, contentHash string) bool {
	// Conta quantas versões ainda usam este hash
	count, err := w.snippetRepo.GetContentHashUsageCount(ctx, contentHash)
	if err != nil {
		log.Printf("⚠️  Error checking hash usage for %s: %v\n", contentHash[:8], err)
		return false
	}

	// Se ainda tem outras versões usando, não deleta
	if count > 0 {
		log.Printf("♻️  Keeping file %s (still used by %d version(s))\n", contentHash[:8], count)
		return false
	}

	// Nenhuma versão usa mais, pode deletar do SeaweedFS
	if err := w.seaweedfs.Delete(ctx, contentHash); err != nil {
		log.Printf("⚠️  Error deleting file %s from storage: %v\n", contentHash[:8], err)
		return false
	}

	log.Printf("🗑️  Deleted orphaned file: %s\n", contentHash[:8])
	return true
}

func (w *Worker) cleanupOrphanedVersions(ctx context.Context) {
	log.Println("🧹 Cleaning orphaned versions...")

	// Deleta versões cujos snippets foram soft-deleted
	result := w.snippetRepo.DB().WithContext(ctx).Exec(`
		DELETE FROM snippet_versions
		WHERE snippet_id IN (
			SELECT id FROM snippets WHERE deleted_at IS NOT NULL
		)
	`)

	if result.Error != nil {
		log.Printf("❌ Error cleaning orphaned versions: %v\n", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		log.Printf("🗑️  Deleted %d orphaned versions\n", result.RowsAffected)
	}
}

func (w *Worker) RunOnce(ctx context.Context) error {
	w.cleanup(ctx)
	return nil
}
