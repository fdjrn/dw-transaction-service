package db

import (
	"context"
	"github.com/fdjrn/dw-transaction-service/configs"
	"github.com/fdjrn/dw-transaction-service/internal/utilities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type MongoCollection struct {
	BalanceTopup      *mongo.Collection
	BalanceDeduct     *mongo.Collection
	BalanceDistribute *mongo.Collection
}

type MongoInstance struct {
	Client     *mongo.Client
	DB         *mongo.Database
	Collection MongoCollection
}

const (
	BalanceTopupCollection      = "balanceTopup"
	BalanceDeductCollection     = "balanceDeduct"
	BalanceDistributeCollection = "balanceDistribution"
)

var Mongo MongoInstance

func (i *MongoInstance) Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(configs.MainConfig.Database.Mongo.Uri).
		SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)))

	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		return err
	}

	// Send a ping to confirm a successful conn
	if err = client.Database("admin").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		return err
	}

	db := client.Database(configs.MainConfig.Database.Mongo.DBName)
	Mongo = MongoInstance{
		Client: client,
		DB:     db,
		Collection: MongoCollection{
			BalanceTopup:      db.Collection(BalanceTopupCollection),
			BalanceDeduct:     db.Collection(BalanceDeductCollection),
			BalanceDistribute: db.Collection(BalanceDistributeCollection),
		},
	}

	utilities.Log.Println("| database >> connected")
	return nil
}

func (i *MongoInstance) Disconnect() error {
	if Mongo.Client == nil {
		return nil
	}

	err := Mongo.Client.Disconnect(context.TODO())
	if err != nil {
		utilities.Log.Println("error on closing mongodb connection: ", err.Error())
		return err
	}
	return nil
}
