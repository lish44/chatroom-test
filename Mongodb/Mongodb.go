package Mongodb

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var DbName string = "chatroom"
var DbCollection = "user"

func LoadDb() bool {
	return initDB()
}

func initDB() bool {
	// 连接配置
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	var err error
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
		return false
	}
	// 检查连接
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
		return false
	}

	fmt.Println("Connected to MongoDB!")
	return true
}

type UserData struct {
	Name         string
	PassWord     string
	RegistryTime int64
}

func insertOne(dbName, dbCollection string, i interface{}) bool {
	collection := client.Database(dbName).Collection(dbCollection)
	insertResult, err := collection.InsertOne(context.TODO(), i)
	if err != nil {
		log.Fatal(err)
		return false
	}
	fmt.Println("Inserted successful ", insertResult.InsertedID)
	return true
}

func Login(userName, passWord string) bool {

	var udt UserData
	filter := bson.D{{"name", userName}}
	collection := client.Database(DbName).Collection(DbCollection)
	err := collection.FindOne(context.TODO(), filter).Decode(&udt)
	if err != nil {
		return false
	}

	return udt.PassWord == passWord

}

func Register(data interface{}) bool {
	return insertOne(DbName, DbCollection, data)
}

func CheckUserNameExist(userName string) bool {
	var udt UserData
	filter := bson.D{{"name", userName}}
	collection := client.Database(DbName).Collection(DbCollection)
	err := collection.FindOne(context.TODO(), filter).Decode(&udt)
	if err != nil {
		return false
	}
	return true

}

func find(dbName, dbCollection string, fitter interface{}) (int, []primitive.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	collection := client.Database(dbName).Collection(dbCollection)
	cur, err := collection.Find(ctx, fitter)
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)
	cnt := 0
	res := []primitive.M{}
	for cur.Next(ctx) {
		var result bson.D
		err := cur.Decode(&result)
		if err != nil {
			log.Fatal(err)
		}
		// fmt.Printf("result: %v\n", result)
		// fmt.Printf("%v\n", result.Map()["name"])
		res = append(res, result.Map())
		cnt++
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}

	return cnt, res
}
