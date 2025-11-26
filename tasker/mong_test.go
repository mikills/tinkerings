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

func TestBatchProcessorRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	client, cleanup := setupMongo(t, ctx)
	defer cleanup()

	coll := client.Database("testdb").Collection("items")

	var docs []interface{}
	for i := 0; i < 250; i++ {
		docs = append(docs, bson.M{
			"_id":       fmt.Sprintf("doc-%d", i),
			"createdAt": time.Now().UnixMilli(),
			"data":      fmt.Sprintf("data-%d", i),
		})
	}

	if _, err := coll.InsertMany(ctx, docs); err != nil {
		t.Fatalf("failed to insert docs: %v", err)
	}

	mgr := New(ctx, 4)
	defer mgr.Shutdown()

	bp := NewBatchProcessor(coll, 100, mgr)

	var _claim atomic.Int32

	err := bp.Run(ctx, func(doc Document) any {
		_claim.Add(1)
		return map[string]string{"tagged": "yes", "docId": doc.ID}
	})

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if p := _claim.Load(); p != 250 {
		t.Errorf("expected 250 _claim, got %d", p)
	}

	// verify all docs have _claim
	count, err := coll.CountDocuments(ctx, bson.D{{Key: "_claim", Value: bson.D{{Key: "$exists", Value: true}}}})
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}

	if count != 250 {
		t.Errorf("expected 250 with _claim, got %d", count)
	}
}

func TestBatchProcessorConcurrentInstances(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	client, cleanup := setupMongo(t, ctx)
	defer cleanup()

	coll := client.Database("testdb").Collection("concurrent_items")

	// insert 500 docs
	var docs []interface{}
	for i := 0; i < 500; i++ {
		docs = append(docs, bson.M{
			"_id":       fmt.Sprintf("doc-%d", i),
			"createdAt": time.Now().UnixMilli(),
			"data":      fmt.Sprintf("data-%d", i),
		})
	}

	if _, err := coll.InsertMany(ctx, docs); err != nil {
		t.Fatalf("failed to insert docs: %v", err)
	}

	// run 5 concurrent instances, each with own manager
	var wg sync.WaitGroup
	var total_claim atomic.Int32

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(instanceID int) {
			defer wg.Done()

			mgr := New(ctx, 4)
			defer mgr.Shutdown()

			bp := NewBatchProcessor(coll, 100, mgr)
			bp.Run(ctx, func(doc Document) any {
				total_claim.Add(1)
				return map[string]any{
					"instance": instanceID,
					"docId":    doc.ID,
				}
			})
		}(i)
	}

	wg.Wait()

	// verify all docs have _claim (exactly once due to atomic filter)
	count, err := coll.CountDocuments(ctx, bson.D{{Key: "_claim", Value: bson.D{{Key: "$exists", Value: true}}}})
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}

	if count != 500 {
		t.Errorf("expected 500 with _claim, got %d", count)
	}

	// total _claim may be >= 500 (some wasted work due to races)
	// but should be close to 500 with jitter helping
	t.Logf("total processFn calls: %d (ideal: 500)", total_claim.Load())
}

func TestBatchProcessorContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	client, cleanup := setupMongo(t, ctx)
	defer cleanup()

	coll := client.Database("testdb").Collection("cancel_items")

	// insert docs
	var docs []interface{}
	for i := 0; i < 1000; i++ {
		docs = append(docs, bson.M{
			"_id":       fmt.Sprintf("doc-%d", i),
			"createdAt": time.Now().UnixMilli(),
			"data":      fmt.Sprintf("data-%d", i),
		})
	}

	if _, err := coll.InsertMany(ctx, docs); err != nil {
		t.Fatalf("failed to insert docs: %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)

	mgr := New(ctx, 4)
	defer mgr.Shutdown()

	bp := NewBatchProcessor(coll, 50, mgr)

	var _claim atomic.Int32
	done := make(chan error)

	go func() {
		done <- bp.Run(ctx, func(doc Document) any {
			_claim.Add(1)
			time.Sleep(10 * time.Millisecond) // slow processing
			return "tagged"
		})
	}()

	// cancel after a short time
	time.Sleep(200 * time.Millisecond)
	cancel()

	err := <-done
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	p := _claim.Load()
	if p >= 1000 {
		t.Errorf("expected fewer than 1000 _claim after cancel, got %d", p)
	}
	t.Logf("_claim before cancel: %d", p)
}

func TestBatchProcessorEmptyCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	client, cleanup := setupMongo(t, ctx)
	defer cleanup()

	coll := client.Database("testdb").Collection("empty_items")

	mgr := New(ctx, 4)
	defer mgr.Shutdown()

	bp := NewBatchProcessor(coll, 100, mgr)

	var called atomic.Bool
	err := bp.Run(ctx, func(doc Document) any {
		called.Store(true)
		return "tagged"
	})

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if called.Load() {
		t.Error("processFn should not be called for empty collection")
	}
}

func TestBatchProcessorAlready_claim(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	client, cleanup := setupMongo(t, ctx)
	defer cleanup()

	coll := client.Database("testdb").Collection("already__claim")

	// insert docs WITH _claim already set
	var docs []interface{}
	for i := 0; i < 100; i++ {
		docs = append(docs, bson.M{
			"_id":       fmt.Sprintf("doc-%d", i),
			"createdAt": time.Now().UnixMilli(),
			"data":      fmt.Sprintf("data-%d", i),
			"_claim":    "already-tagged",
		})
	}

	if _, err := coll.InsertMany(ctx, docs); err != nil {
		t.Fatalf("failed to insert docs: %v", err)
	}

	mgr := New(ctx, 4)
	defer mgr.Shutdown()

	bp := NewBatchProcessor(coll, 100, mgr)

	var called atomic.Bool
	err := bp.Run(ctx, func(doc Document) any {
		called.Store(true)
		return "new-tag"
	})

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if called.Load() {
		t.Error("processFn should not be called when all docs already have _claim")
	}
}
