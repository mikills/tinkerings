package main

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Document struct {
	ID        string `bson:"_id"`
	CreatedAt int64  `bson:"createdAt"`
	Data      string `bson:"data"`
}

type BatchProcessor struct {
	collection *mongo.Collection
	batchSize  int64
	manager    *Manager
}

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

type _claimDoc struct {
	id     string
	result any
}

func (bp *BatchProcessor) Run(ctx context.Context, processFn func(doc Document) any) error {
	jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
	time.Sleep(jitter)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		cursor, err := bp.collection.Find(ctx,
			bson.D{{Key: "_claim", Value: bson.D{{Key: "$exists", Value: false}}}},
			options.Find().SetLimit(bp.batchSize),
		)
		if err != nil {
			return err
		}

		var docs []Document
		if err := cursor.All(ctx, &docs); err != nil {
			cursor.Close(ctx)
			return err
		}
		cursor.Close(ctx)

		if len(docs) == 0 {
			return nil
		}

		results := make(chan _claimDoc, len(docs))
		var wg sync.WaitGroup

		for _, doc := range docs {
			wg.Add(1)
			d := doc
			bp.manager.Submit(func(ctx context.Context) error {
				defer wg.Done()
				result := processFn(d)
				results <- _claimDoc{id: d.ID, result: result}
				return nil
			}, RetryPolicy{MaxAttempts: 1})
		}

		wg.Wait()
		close(results)

		models := make([]mongo.WriteModel, 0, len(docs))
		for r := range results {
			models = append(models, mongo.NewUpdateOneModel().
				SetFilter(bson.D{
					{Key: "_id", Value: r.id},
					{Key: "_claim", Value: bson.D{{Key: "$exists", Value: false}}},
				}).
				SetUpdate(bson.D{
					{Key: "$set", Value: bson.D{{Key: "_claim", Value: r.result}}},
				}),
			)
		}

		_, err = bp.collection.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
		if err != nil {
			return err
		}
	}
}
