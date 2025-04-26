package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	rdb         *redis.Client
	mongoClient *mongo.Client
	ctx         = context.Background()
)

// Prediction represents a single football match prediction
type Prediction struct {
	MatchID           string    `json:"match_id"`
	HomeTeam          string    `json:"home_team"`
	AwayTeam          string    `json:"away_team"`
	OneXTwo           string    `json:"1x2"`                  // "1", "X", or "2"
	OverUnder3_5      string    `json:"over_under_3_5g"`      // "Over" or "Under"
	OverUnder2_5      string    `json:"over_under_2_5g"`      // "Over" or "Under"
	BTTS              string    `json:"btts"`                 // "Yes" or "No"
	AwayOverUnder1_5  string    `json:"away_over_under_1_5"`  // "Over" or "Under"
	AwayScore         string    `json:"away_to_score"`        // "Yes" or "No"
	HomeOverUnder1_5  string    `json:"home_over_under_1_5"`  // "Over" or "Under"
	HomeScore         string    `json:"home_to_score"`        // "Yes" or "No"
	Timestamp         time.Time `json:"timestamp"`
}

func main() {
	// Initialize Redis
	rdb = redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	// Test Redis connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}

	// Initialize MongoDB (used only for background jobs)
	var err error
	mongoURI := getEnv("MONGO_URI", "mongodb://mongodb_server:27017")
	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Could not connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	// Setup HTTP router
	r := chi.NewRouter()

	r.Get("/api/v1/predictions", getTodaysPredictions)

	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s...", port)
	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func getTodaysPredictions(w http.ResponseWriter, r *http.Request) {
	data, err := rdb.Get(ctx, "predictions:today").Result()
	if err == redis.Nil {
		http.Error(w, "Predictions not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var predictions []Prediction
	if err := json.Unmarshal([]byte(data), &predictions); err != nil {
		http.Error(w, "Failed to parse prediction data", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data":    predictions,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getEnv fetches environment variables with a fallback default
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
