package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupMongo(t *testing.T, ctx context.Context) (*mongo.Client, func()) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "mongo:7",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForListeningPort("27017/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start mongo container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "27017")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	uri := fmt.Sprintf("mongodb://%s:%s", host, port.Port())
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("failed to connect to mongo: %v", err)
	}

	cleanup := func() {
		client.Disconnect(ctx)
		container.Terminate(ctx)
	}

	return client, cleanup
}

func TestBatchProcessorPagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	client, cleanup := setupMongo(t, ctx)
	defer cleanup()

	coll := client.Database("testdb").Collection("items")

	if err := EnsureCreatedAtIndex(ctx, coll); err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// 350 documents: 250 before cutoff, 100 after
	cutoff := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	var docs []interface{}

	for i := 0; i < 250; i++ {
		docs = append(docs, Document{
			ID:        primitive.NewObjectID(),
			CreatedAt: cutoff.AddDate(0, 0, -(i + 1)),
			Data:      fmt.Sprintf("before-%d", i),
		})
	}

	for i := 0; i < 100; i++ {
		docs = append(docs, Document{
			ID:        primitive.NewObjectID(),
			CreatedAt: cutoff.AddDate(0, 0, i+1),
			Data:      fmt.Sprintf("after-%d", i),
		})
	}

	if _, err := coll.InsertMany(ctx, docs); err != nil {
		t.Fatalf("failed to insert docs: %v", err)
	}

	mgr := New(ctx, 4)
	defer mgr.Shutdown()

	bp := NewBatchProcessor(coll, 100, mgr)

	var (
		processedCount atomic.Int32
		batchCount     atomic.Int32
		seenIDs        sync.Map
	)

	total, err := bp.ProcessBefore(ctx, cutoff, func(ctx context.Context, batch []Document) error {
		batchCount.Add(1)
		for _, doc := range batch {
			if !doc.CreatedAt.Before(cutoff) {
				t.Errorf("document %s has createdAt %v >= cutoff %v", doc.ID.Hex(), doc.CreatedAt, cutoff)
			}
			if _, loaded := seenIDs.LoadOrStore(doc.ID.Hex(), true); loaded {
				t.Errorf("duplicate document: %s", doc.ID.Hex())
			}
			processedCount.Add(1)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("ProcessBefore failed: %v", err)
	}

	mgr.Shutdown()

	if total != 250 {
		t.Errorf("expected 250 documents queued, got %d", total)
	}

	if processed := processedCount.Load(); processed != 250 {
		t.Errorf("expected 250 documents processed, got %d", processed)
	}

	// 3 batches: 100 + 100 + 50
	if batches := batchCount.Load(); batches != 3 {
		t.Errorf("expected 3 batches, got %d", batches)
	}
}

func TestBatchProcessorConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	client, cleanup := setupMongo(t, ctx)
	defer cleanup()

	coll := client.Database("testdb").Collection("concurrent_items")

	if err := EnsureCreatedAtIndex(ctx, coll); err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	cutoff := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	var docs []interface{}
	for i := 0; i < 500; i++ {
		docs = append(docs, Document{
			ID:        primitive.NewObjectID(),
			CreatedAt: cutoff.AddDate(0, 0, -(i + 1)),
			Data:      fmt.Sprintf("item-%d", i),
		})
	}

	if _, err := coll.InsertMany(ctx, docs); err != nil {
		t.Fatalf("failed to insert docs: %v", err)
	}

	const workers = 3
	mgr := New(ctx, workers)

	bp := NewBatchProcessor(coll, 100, mgr)

	var (
		concurrent atomic.Int32
		maxSeen    atomic.Int32
	)

	total, err := bp.ProcessBefore(ctx, cutoff, func(ctx context.Context, batch []Document) error {
		cur := concurrent.Add(1)
		defer concurrent.Add(-1)

		for {
			old := maxSeen.Load()
			if cur <= old || maxSeen.CompareAndSwap(old, cur) {
				break
			}
		}

		time.Sleep(50 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Fatalf("ProcessBefore failed: %v", err)
	}

	mgr.Shutdown()

	if total != 500 {
		t.Errorf("expected 500 documents, got %d", total)
	}

	if max := maxSeen.Load(); max > int32(workers) {
		t.Errorf("concurrency exceeded limit: got %d, want <= %d", max, workers)
	}
}

func TestBatchProcessorCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	client, cleanup := setupMongo(t, ctx)
	defer cleanup()

	coll := client.Database("testdb").Collection("cancel_items")

	if err := EnsureCreatedAtIndex(ctx, coll); err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	cutoff := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	var docs []interface{}
	for i := 0; i < 500; i++ {
		docs = append(docs, Document{
			ID:        primitive.NewObjectID(),
			CreatedAt: cutoff.AddDate(0, 0, -(i + 1)),
			Data:      fmt.Sprintf("item-%d", i),
		})
	}

	if _, err := coll.InsertMany(ctx, docs); err != nil {
		t.Fatalf("failed to insert docs: %v", err)
	}

	mgrCtx, cancel := context.WithCancel(ctx)
	mgr := New(mgrCtx, 2)

	bp := NewBatchProcessor(coll, 100, mgr)

	var started atomic.Int32

	go func() {
		bp.ProcessBefore(ctx, cutoff, func(ctx context.Context, docs []Document) error {
			started.Add(1)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				return nil
			}
		})
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	mgr.Wait()

	if s := started.Load(); s >= 5 {
		t.Errorf("expected fewer than 5 batches to start after cancel, got %d", s)
	}
}

func TestBatchProcessorEmptyCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	client, cleanup := setupMongo(t, ctx)
	defer cleanup()

	coll := client.Database("testdb").Collection("empty_items")

	if err := EnsureCreatedAtIndex(ctx, coll); err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	mgr := New(ctx, 2)
	defer mgr.Shutdown()

	bp := NewBatchProcessor(coll, 100, mgr)

	cutoff := time.Now()
	var called atomic.Bool

	total, err := bp.ProcessBefore(ctx, cutoff, func(ctx context.Context, batch []Document) error {
		called.Store(true)
		return nil
	})

	if err != nil {
		t.Fatalf("ProcessBefore failed: %v", err)
	}

	if total != 0 {
		t.Errorf("expected 0 documents, got %d", total)
	}

	if called.Load() {
		t.Error("processFn should not be called for empty collection")
	}
}

func TestBatchProcessorSameDateOrdering(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	client, cleanup := setupMongo(t, ctx)
	defer cleanup()

	coll := client.Database("testdb").Collection("same_date_items")

	if err := EnsureCreatedAtIndex(ctx, coll); err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// 150 documents with identical createdAt to test _id tiebreaker
	sameTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	cutoff := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	var docs []interface{}
	for i := 0; i < 150; i++ {
		docs = append(docs, Document{
			ID:        primitive.NewObjectID(),
			CreatedAt: sameTime,
			Data:      fmt.Sprintf("same-date-%d", i),
		})
	}

	if _, err := coll.InsertMany(ctx, docs); err != nil {
		t.Fatalf("failed to insert docs: %v", err)
	}

	mgr := New(ctx, 2)
	defer mgr.Shutdown()

	bp := NewBatchProcessor(coll, 100, mgr)

	var (
		seenIDs sync.Map
		count   atomic.Int32
	)

	total, err := bp.ProcessBefore(ctx, cutoff, func(ctx context.Context, batch []Document) error {
		for _, doc := range batch {
			if _, loaded := seenIDs.LoadOrStore(doc.ID.Hex(), true); loaded {
				t.Errorf("duplicate document with same createdAt: %s", doc.ID.Hex())
			}
			count.Add(1)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("ProcessBefore failed: %v", err)
	}

	mgr.Shutdown()

	if total != 150 {
		t.Errorf("expected 150 documents, got %d", total)
	}

	if c := count.Load(); c != 150 {
		t.Errorf("expected 150 processed, got %d", c)
	}
}

// EnsureCreatedAtIndex creates a descending index on createdAt.
func EnsureCreatedAtIndex(ctx context.Context, coll *mongo.Collection) error {
	idx := mongo.IndexModel{
		Keys:    bson.D{{Key: "createdAt", Value: -1}},
		Options: options.Index().SetName("idx_createdAt"),
	}
	_, err := coll.Indexes().CreateOne(ctx, idx)
	return err
}
