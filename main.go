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
	"github.com/go-chi/cors"
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
		Addr:     getEnv("REDIS_URL", "redis:6379"),
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

	// Add CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"}, // Allow all origins
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-RapidAPI-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/api/v1/predictions", getTodaysPredictions)
	r.Post("/api/v1/predictions", savePredictions)

	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s...", port)
	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func getTodaysPredictions(w http.ResponseWriter, r *http.Request) {
	proxySecret := r.Header.Get("X-RapidAPI-Proxy-Secret")
	expectedSecret := getEnv("RAPIDAPI_PROXY_SECRET", "")
	if proxySecret == "" || proxySecret != expectedSecret {
		http.Error(w, "404", http.StatusUnauthorized)
		return
	}
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

func savePredictions(w http.ResponseWriter, r *http.Request) {
	// Read and parse the incoming predictions
	var predictions []Prediction
	if err := json.NewDecoder(r.Body).Decode(&predictions); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Save predictions to MongoDB
	collection := mongoClient.Database("matchpredictions").Collection("predictions")
	for _, prediction := range predictions {
		_, err := collection.InsertOne(ctx, prediction)
		if err != nil {
			http.Error(w, "Failed to insert prediction into MongoDB", http.StatusInternalServerError)
			return
		}
	}

	// Cache predictions in Redis
	data, err := json.Marshal(predictions)
	if err != nil {
		http.Error(w, "Failed to serialize predictions", http.StatusInternalServerError)
		return
	}
	if err := rdb.Set(ctx, "predictions:today", data, 30*time.Hour).Err(); err != nil {
		http.Error(w, "Failed to cache predictions in Redis", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Predictions saved successfully",
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
