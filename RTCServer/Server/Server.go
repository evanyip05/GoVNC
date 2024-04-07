package Server

import (
	"math/rand"
	"net/http"
	"strings"	
	"sync"
	"fmt"
	"io"
	"os"

	"github.com/gin-gonic/gin"
)

var clients = make(map[string]chan string)
var clientsLock sync.Mutex

func register(length int) string {
	clientsLock.Lock()
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, length)
	for i := 0; i < length; i++ {
		code[i] = chars[rand.Intn(len(chars))]
	}

	_, ok := clients[string(code)]

	if ok {
		return register(length)
	} else {
		clients[string(code)] = make(chan string)
		clientsLock.Unlock()
		return string(code)
	}
}

func unregister(id string) {
	clientsLock.Lock()
	println("\nDELETING ID:", id, "\n")
	delete(clients, id)
	clientsLock.Unlock()
}

func printStatus() {
	println("\n\n\nCONNECTIONS:")
	for k := range clients {
		println("CONNECTION: ", k)
	}
	println("\n\n")
}

// access this guy via ip on lan
func Serve() {
	router := gin.Default()

	// allow cors
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})
	router.GET("/")

	router.GET("/download", func(c *gin.Context) {
		filePath := c.Query("path") // Replace with the actual path to your file

		file, err := os.Open(filePath)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to open the file")
			return
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to get file info")
			return
		}

		fileName := fileInfo.Name()

		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename="+fileName)
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Length", fmt.Sprint(fileInfo.Size()))

		// Stream the file to the client
		http.ServeContent(c.Writer, c.Request, fileName, fileInfo.ModTime(), file)
	})

	router.GET("/register", func(c *gin.Context) {
		id := c.Query("id")

		if id == "" {
			c.Writer.WriteString(register(2))
		} else {
			clientsLock.Lock()
			clients[id] = make(chan string)
			clientsLock.Unlock()
			c.Writer.WriteString(id)
		}
	})

	router.Any("/unregister", func(c *gin.Context) {
		unregister(c.Query("id"))
	})

	// called from FE, should register a listener for connections
	router.GET("/listen", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		handle := clients[c.Query("id")]

		println("listening to id:", c.Query("id"))

		for {
			select {
			case message := <-handle:
				event := "message" // Event name
				data := message    // Event data

				// Write the SSE event in the correct format
				c.Writer.WriteString(fmt.Sprintf("event: %s\ndata: %s\n\n", event, data))
				c.Writer.Flush()
			case <-c.Writer.CloseNotify():

				println("disconnected :<")
				return // handle disconnect
			}
		}
	})

	router.POST("/call", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return
		}

		//sendMessage(c.Writer, c.Request, string(body))
		handle, ok := clients[strings.ToUpper(c.Query("id"))]
		println("sending ", ok)
		if ok {
			handle <- string(body)
		}
	})

	err := router.Run(":8080")
	if err != nil {
		fmt.Println(err)
	}
}