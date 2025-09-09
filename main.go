package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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

	fmt.Println(newShortenReq.Url)

	newShortenRes := response{
		ID:        1,
		Url:       newShortenReq.Url,
		Code:      "abc",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	fmt.Println(newShortenRes.CreatedAt)

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
