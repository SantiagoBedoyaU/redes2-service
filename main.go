package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	client *mongo.Client
)

type Record struct {
	IP          string `bson:"ip"`
	Connections uint   `bson:"connections"`
}

func init() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dbURI := "mongodb://localhost:27017"
	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(dbURI))
	if err != nil {
		panic(err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		panic(err)
	}
	fmt.Println("Database successfully connected")
}

func getConnectionsByIP(ip string) uint {
	coll := client.Database("redes2").Collection("connections")
	filter := bson.M{
		"ip": ip,
	}
	record := Record{}
	ctx := context.Background()
	err := coll.FindOne(ctx, filter).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// insert record
			record.Connections = 1
			record.IP = ip
			_, err = coll.InsertOne(ctx, record)
			if err != nil {
				panic(err)
			}
			return record.Connections
		}
		panic(err)
	}
	record.Connections++
	// update record in database
	_, err = coll.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"connections": record.Connections}})
	if err != nil {
		panic(err)
	}

	return record.Connections
}

func handler(ctx *gin.Context) {
	ip := ctx.RemoteIP()
	if realIP, _, err := net.SplitHostPort(ip); err == nil {
		ip = realIP
	}

	if ipHeader := ctx.GetHeader("X-Forwarded-For"); ipHeader != "" {
		ip = ipHeader
	}
	visitas := getConnectionsByIP(ip)
	fmt.Fprintf(ctx.Writer, "Hola, me visitas desde la IP: %s y me has visitado %d veces", ip, visitas)
}

func main() {
	router := gin.Default()
	router.GET("/", handler)
	fmt.Println("Service started at port 8080")
	if err := router.Run("0.0.0.0:8080"); err != nil {
		panic(err)
	}
}
