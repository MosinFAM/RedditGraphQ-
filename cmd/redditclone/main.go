package main

import (
	"log"
	"net/http"
	"os"

	"github.com/MosinFAM/graphql-posts/internal/db"
	"github.com/MosinFAM/graphql-posts/internal/graph"
	"github.com/MosinFAM/graphql-posts/internal/storage"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

func main() {
	storeType := os.Getenv("STORAGE_TYPE")
	var store storage.Storage

	if storeType == "postgres" {
		dbConn, err := db.Connect()
		if err != nil {
			log.Fatal("Failed to connect to DB:", err)
		}

		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			log.Fatal("DATABASE_URL is not set")
		}

		store = storage.NewPostgresStorage(dbConn, dsn)
	} else {
		store = storage.NewMemoryStorage()
	}

	resolver := &graph.Resolver{Storage: store}
	schema := graph.NewExecutableSchema(graph.Config{Resolvers: resolver})
	srv := handler.New(schema)

	// Поддержка WebSockets
	srv.AddTransport(transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	})

	srv.AddTransport(transport.POST{})

	// Настройка Gin и CORS
	r := gin.Default()
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
	})

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}
		c.Next()
	})

	r.POST("/query", gin.WrapH(c.Handler(srv)))
	r.GET("/query", gin.WrapH(srv))

	r.GET("/", gin.WrapH(playground.Handler("GraphQL Playground", "/query")))

	log.Println("Server is running on port 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
