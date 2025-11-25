package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Document represents a record with a creation timestamp.
type Document struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	CreatedAt time.Time          `bson:"createdAt"`
	Data      string             `bson:"data"`
}

// BatchProcessor handles paginated MongoDB queries with worker dispatch.
type BatchProcessor struct {
	collection *mongo.Collection
	batchSize  int64
	manager    *Manager
}

// NewBatchProcessor creates a processor for the given collection.
func NewBatchProcessor(coll *mongo.Collection, batchSize int, mgr *Manager) *BatchProcessor {
	if batchSize < 1 {
		batchSize = 100
	}
	return &BatchProcessor{
		collection: coll,
		batchSize:  int64(batchSize),
		manager:    mgr,
	}
}

// ProcessFunc is called for each batch of documents.
type ProcessFunc func(ctx context.Context, docs []Document) error

// ProcessBefore fetches all documents with createdAt < cutoff in batches,
// submitting each batch to the worker pool via processFn.
// Returns total documents processed or error.
func (bp *BatchProcessor) ProcessBefore(ctx context.Context, cutoff time.Time, processFn ProcessFunc) (int, error) {
	var (
		total      int
		lastID     primitive.ObjectID
		lastTime   time.Time
		firstBatch = true
	)

	for {
		batch, err := bp.fetchBatch(ctx, cutoff, lastTime, lastID, firstBatch)
		if err != nil {
			return total, err
		}
		if len(batch) == 0 {
			break
		}

		firstBatch = false
		last := batch[len(batch)-1]
		lastTime = last.CreatedAt
		lastID = last.ID

		// capture for closure
		docs := batch
		submitted := bp.manager.Submit(func(ctx context.Context) error {
			return processFn(ctx, docs)
		}, RetryPolicy{MaxAttempts: 1})

		if !submitted {
			break
		}
		total += len(batch)

		if int64(len(batch)) < bp.batchSize {
			break
		}
	}

	return total, nil
}

func (bp *BatchProcessor) fetchBatch(
	ctx context.Context,
	cutoff, lastTime time.Time,
	lastID primitive.ObjectID,
	firstBatch bool,
) ([]Document, error) {
	var filter bson.D

	if firstBatch {
		filter = bson.D{{Key: "createdAt", Value: bson.D{{Key: "$lt", Value: cutoff}}}}
	} else {
		// cursor pagination: createdAt < lastTime OR (createdAt == lastTime AND _id < lastID)
		filter = bson.D{
			{Key: "$or", Value: bson.A{
				bson.D{{Key: "createdAt", Value: bson.D{{Key: "$lt", Value: lastTime}}}},
				bson.D{
					{Key: "createdAt", Value: lastTime},
					{Key: "_id", Value: bson.D{{Key: "$lt", Value: lastID}}},
				},
			}},
		}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(bp.batchSize)

	cursor, err := bp.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []Document
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}
