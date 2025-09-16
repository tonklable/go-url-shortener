package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var counter struct {
	Count int `bson:"count"`
}

type request struct {
	Url string `json:"url"`
}

var inputReq request

type response struct {
	ID        int       `json:"id"`
	Url       string    `json:"url"`
	Code      string    `json:"shortCode"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

var outputRes response

type errorResponse struct {
	Message string `json:"message"`
}

var errorRes errorResponse

func main() {
	router := gin.Default()
	router.POST("/shorten", postShorten)
	router.GET("/shorten/:code", getShortenByCode)
	router.PUT("/shorten/:code", putShortenByCode)
	router.DELETE("/shorten/:code", deleteShortenByCode)

	router.Run("localhost:8080")
}

var (
	CounterCol *mongo.Collection
	UrlCol     *mongo.Collection
)

func postShorten(c *gin.Context) {

	if err := c.BindJSON(&inputReq); err != nil {
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusBadRequest, errorRes)
		return
	}

	hash := hashUrl(inputReq.Url)

	if err := connect(); err != nil {
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusServiceUnavailable, errorRes)
		return
	}

	if err := getNewId(); err != nil {
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusServiceUnavailable, errorRes)
		return
	}

	outputRes = response{
		ID:        counter.Count,
		Url:       inputReq.Url,
		Code:      hash[:4],
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	result, err := UrlCol.InsertOne(context.TODO(), outputRes)
	if err != nil {
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusServiceUnavailable, errorRes)
	}
	fmt.Printf("URL inserted with ID: %s\n", result.InsertedID)

	c.IndentedJSON(http.StatusCreated, outputRes)

}

func getShortenByCode(c *gin.Context) {
	code := c.Param("code")

	fmt.Println(code)

	if err := connect(); err != nil {
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusServiceUnavailable, errorRes)
		return
	}

	if err := getOriginalUrl(code); err != nil {
		errorRes.Message = err.Error()
		if err == mongo.ErrNoDocuments {
			c.IndentedJSON(http.StatusNotFound, errorRes)
			return
		}
		c.IndentedJSON(http.StatusBadRequest, errorRes)
		return
	}
	c.IndentedJSON(http.StatusOK, outputRes)

}

func putShortenByCode(c *gin.Context) {
	code := c.Param("code")

	if err := c.BindJSON(&inputReq); err != nil {
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusBadRequest, errorRes)
		return
	}

	fmt.Println(code)

	if err := connect(); err != nil {
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusServiceUnavailable, errorRes)
		return
	}

	filter := bson.M{"code": code}
	update := bson.M{"$set": bson.M{"url": inputReq.Url, "updatedat": time.Now()}}

	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After)

	if err := UrlCol.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&outputRes); err != nil {
		errorRes.Message = err.Error()
		if err == mongo.ErrNoDocuments {
			c.IndentedJSON(http.StatusNotFound, errorRes)
			return
		}
		c.IndentedJSON(http.StatusBadRequest, errorRes)
		return
	}

	c.IndentedJSON(http.StatusOK, outputRes)
}

func deleteShortenByCode(c *gin.Context) {
	code := c.Param("code")

	fmt.Println(code)

	if err := connect(); err != nil {
		c.Error(err)
		return
	}

	filter := bson.M{"code": code}

	res, err := UrlCol.DeleteOne(context.TODO(), filter)
	if err != nil {
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusBadRequest, errorRes)
		return
	}

	if res.DeletedCount == 0 {
		errorRes.Message = "No deletion due to document not found"
		c.IndentedJSON(http.StatusNotFound, errorRes) // No matching document
		return
	}

	c.Status(http.StatusNoContent)
}

func hashUrl(url string) string {
	sha1 := sha1.New()
	io.WriteString(sha1, url)
	return hex.EncodeToString(sha1.Sum(nil))
}

func connect() error {
	err := godotenv.Load(".env")
	if err != nil {
		return err
	}

	MONGO_URI := os.Getenv("MONGO_URI")

	clientOption := options.Client().ApplyURI(MONGO_URI)
	client, err := mongo.Connect(context.TODO(), clientOption)
	if err != nil {
		return err
	}

	CounterCol = client.Database("go-url-shortener").Collection("counter")
	UrlCol = client.Database("go-url-shortener").Collection("urls")

	fmt.Println("Connected to db")

	return nil
}

func getNewId() error {
	filter := bson.M{"_id": "counter"}
	update := bson.M{"$inc": bson.M{"count": 1}}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	if err := CounterCol.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&counter); err != nil {
		return err
	}

	return nil
}

func getOriginalUrl(code string) error {
	filter := bson.M{"code": code}
	if err := UrlCol.FindOne(context.TODO(), filter).Decode(&outputRes); err != nil {
		return err
	}

	return nil
}
