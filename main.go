package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type request struct {
	Url string `json:"url"`
}

type response struct {
	ID        int       `json:"id"`
	Url       string    `json:"url"`
	Code      string    `json:"shortCode"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func main() {
	router := gin.Default()
	router.POST("/shorten", postShorten)
	router.GET("/shorten/:code", getShortenByCode)

	router.Run("localhost:8080")
}

func postShorten(c *gin.Context) {
	var newShortenReq request

	if err := c.BindJSON(&newShortenReq); err != nil {
		return
	}

	collection := connect()

	fmt.Println(newShortenReq.Url)

	hash := hashUrl(newShortenReq.Url)

	newShortenRes := response{
		ID:        1,
		Url:       newShortenReq.Url,
		Code:      hash,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	result, err := collection.InsertOne(context.TODO(), newShortenRes)
	if err != nil {
		panic(err)
	}
	fmt.Printf("URL inserted with ID: %s\n", result.InsertedID)

	c.IndentedJSON(http.StatusCreated, newShortenRes)

}

func getShortenByCode(c *gin.Context) {
	code := c.Param("code")

	fmt.Println(code)

	newShortenRes := response{
		ID:        1,
		Url:       "abc",
		Code:      code,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	c.IndentedJSON(http.StatusOK, newShortenRes)

}

func hashUrl(url string) string {
	sha1 := sha1.New()
	io.WriteString(sha1, url)
	return hex.EncodeToString(sha1.Sum(nil))
}

func connect() *mongo.Collection {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %s", err)
	}

	MONGO_URI := os.Getenv("MONGO_URI")

	clientOption := options.Client().ApplyURI(MONGO_URI)
	client, err := mongo.Connect(context.TODO(), clientOption)
	if err != nil {
		panic(err)
	}

	collection := client.Database("go-url-shortener").Collection("urls")

	fmt.Println("Connected to db")
	return collection
}
