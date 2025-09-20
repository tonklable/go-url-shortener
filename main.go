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
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var counter struct {
	Count int `bson:"count"`
}

type request struct {
	Url string `json:"url" binding:"required"`
}

type response struct {
	ID        int       `json:"id"`
	Url       string    `json:"url"`
	Code      string    `json:"shortCode"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type errorResponse struct {
	Message string `json:"message"`
}

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

	var inputReq request
	var outputRes response

	if err := c.BindJSON(&inputReq); err != nil {
		var errorRes errorResponse
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusBadRequest, errorRes)
		return
	}

	if err := connect(); err != nil {
		var errorRes errorResponse
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusServiceUnavailable, errorRes)
		return
	}

	err := getInfoByUrl(inputReq.Url, &outputRes)
	if err == nil {
		c.IndentedJSON(http.StatusOK, outputRes)
		return
	}

	hash := hashUrl(inputReq.Url)

	if err := getNewId(); err != nil {
		var errorRes errorResponse
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
		var errorRes errorResponse
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusServiceUnavailable, errorRes)
		return
	}
	fmt.Printf("URL inserted with ID: %s\n", result.InsertedID)

	c.IndentedJSON(http.StatusCreated, outputRes)

}

func getShortenByCode(c *gin.Context) {

	var outputRes response

	code := c.Param("code")

	fmt.Println(code)

	if err := connect(); err != nil {
		var errorRes errorResponse
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusServiceUnavailable, errorRes)
		return
	}

	if err := getInfoByCode(code, &outputRes); err != nil {
		var errorRes errorResponse
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
	var inputReq request
	var outputRes response

	code := c.Param("code")

	if err := c.BindJSON(&inputReq); err != nil {
		var errorRes errorResponse
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusBadRequest, errorRes)
		return
	}

	if err := connect(); err != nil {
		var errorRes errorResponse
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusServiceUnavailable, errorRes)
		return
	}

	err := getInfoByUrl(inputReq.Url, &outputRes)
	if err == nil {
		var errorRes errorResponse
		errorRes.Message = "Already exists this url in the code " + outputRes.Code
		c.IndentedJSON(http.StatusBadRequest, errorRes)
		return
	}

	filter := bson.M{"code": code}
	update := bson.M{"$set": bson.M{"url": inputReq.Url, "updatedat": time.Now()}}

	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After)

	if err := UrlCol.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&outputRes); err != nil {
		var errorRes errorResponse
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
		var errorRes errorResponse
		errorRes.Message = err.Error()
		c.IndentedJSON(http.StatusBadRequest, errorRes)
		return
	}

	if res.DeletedCount == 0 {
		var errorRes errorResponse
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
	client, err := mongo.Connect(clientOption)
	if err != nil {
		return err
	}

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "url", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	CounterCol = client.Database("go-url-shortener").Collection("counter")
	UrlCol = client.Database("go-url-shortener").Collection("urls")
	_, err = UrlCol.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return err
	}

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

func getInfoByCode(code string, outputRes *response) error {
	filter := bson.M{"code": code}
	if err := UrlCol.FindOne(context.TODO(), filter).Decode(&outputRes); err != nil {
		return err
	}

	return nil
}

func getInfoByUrl(url string, outputRes *response) error {
	filter := bson.M{"url": url}
	if err := UrlCol.FindOne(context.TODO(), filter).Decode(&outputRes); err != nil {
		return err
	}
	return nil
}
