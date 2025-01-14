package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

// Initialize the database connection
func init() {
	var err error
	email := os.Getenv("MONGO_EMAIL")
	password := os.Getenv("MONGO_PASSWORD")
	if email == "" || password == "" {
		log.Fatal("Environment variables MONGO_EMAIL and MONGO_PASSWORD must be set")
	}
	encodedPassword := url.QueryEscape(password)
	clientOptions := options.Client().ApplyURI("mongodb+srv://" + email + ":" + encodedPassword + "@cluster0.zuzzadg.mongodb.net/?retryWrites=true&w=majority")
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal("Error connecting to MongoDB:", err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal("Error pinging MongoDB:", err)
	}

	fmt.Println("Connected to MongoDB successfully!")
}

// Handler function to return all documents
func getAllDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	collection := client.Database("Telegram").Collection("CrocEn")
	cursor, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		http.Error(w, "Failed to fetch documents", http.StatusInternalServerError)
		log.Println("Error fetching documents:", err)
		return
	}
	defer cursor.Close(context.TODO())

	var results []bson.M
	if err := cursor.All(context.TODO(), &results); err != nil {
		http.Error(w, "Error decoding documents", http.StatusInternalServerError)
		log.Println("Error decoding documents:", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		log.Println("Error encoding JSON:", err)
	}
}

func main() {
	http.HandleFunc("/documents", getAllDocumentsHandler)
	http.HandleFunc("/leaderboard", countIDOccurrencesHandler)

	fmt.Println("Server is running on http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

// Handler function to count occurrences of each ID
func countIDOccurrencesHandler(w http.ResponseWriter, r *http.Request) {
	collection := client.Database("Telegram").Collection("CrocEn")

	// Aggregation pipeline to count occurrences of each ID
	pipeline := mongo.Pipeline{
		{{"$group", bson.D{
			{Key: "_id", Value: "$ID"},                                    // Group by the "ID" field
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},        // Count occurrences
			{Key: "Name", Value: bson.D{{Key: "$first", Value: "$Name"}}}, // Get the first "Name" encountered for the grouped ID
		}}},
		{{"$sort", bson.D{{Key: "count", Value: -1}}}}, // Sort by count in descending order
	}

	// Execute the aggregation query
	cursor, err := collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		http.Error(w, "Failed to count ID occurrences", http.StatusInternalServerError)
		log.Println("Error in aggregation:", err)
		return
	}
	defer cursor.Close(context.TODO())

	var results []bson.M
	if err := cursor.All(context.TODO(), &results); err != nil {
		http.Error(w, "Error decoding results", http.StatusInternalServerError)
		log.Println("Error decoding results:", err)
		return
	}

	// Respond with the JSON result
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		log.Println("Error encoding JSON:", err)
	}
}