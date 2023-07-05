package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/fdjrn/dw-transaction-service/internal/db"
	"github.com/fdjrn/dw-transaction-service/internal/db/entity"
	"github.com/fdjrn/dw-transaction-service/internal/utilities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type TransactionRepository struct {
	Model *entity.BalanceTransaction
}

func NewTransactionRepository() TransactionRepository {
	return TransactionRepository{Model: new(entity.BalanceTransaction)}
}

func getDefaultCollection(transType int) *mongo.Collection {
	collection := new(mongo.Collection)
	switch transType {
	case utilities.TransTypeTopUp:
		collection = db.Mongo.Collection.BalanceTopup
	case utilities.TransTypePayment:
		collection = db.Mongo.Collection.BalanceDeduct
	case utilities.TransTypeDistribution:
		collection = db.Mongo.Collection.BalanceDistribute
	default:
		collection = nil
	}

	return collection
}

func (t *TransactionRepository) Create(transType int) (interface{}, error) {

	result, err := getDefaultCollection(transType).InsertOne(context.TODO(), t.Model)
	if err != nil {
		return nil, err
	}

	return result.InsertedID, nil
}

func (t *TransactionRepository) FindByID(id interface{}, transType int) (interface{}, error) {
	// filter condition
	filter := bson.D{{"_id", id}}

	//var trans entity.BalanceTransaction
	err := getDefaultCollection(transType).FindOne(context.TODO(), filter).Decode(t.Model)
	if err != nil {
		return nil, err
	}

	return t.Model, nil
}

func (t *TransactionRepository) FindByRefNo() error {
	// filter condition
	filter := bson.D{{"referenceNo", t.Model.ReferenceNo}}

	//var trans entity.BalanceTransaction
	err := getDefaultCollection(t.Model.TransType).FindOne(context.TODO(), filter).Decode(t.Model)
	if err != nil {
		return err
	}

	return nil
}

func (t *TransactionRepository) RemoveByID(id interface{}, transType int) error {

	filter := bson.D{{"_id", id}}
	result, err := getDefaultCollection(transType).DeleteOne(context.TODO(), filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New(fmt.Sprintf("cannot find document with current id: %s", id))
	}

	return nil
}

func (t *TransactionRepository) IsUsedPartnerRefNumber(code string, transType int) bool {

	// filter condition
	filter := bson.D{{"partnerRefNumber", code}}

	var trans entity.BalanceTransaction
	err := getDefaultCollection(transType).FindOne(context.TODO(), filter).Decode(&trans)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false
		}
		utilities.Log.Println(err.Error())
		return true
	}

	return true
}

func (t *TransactionRepository) Update(transType int) error {

	update := bson.D{
		{"$set", bson.D{
			{"transDate", t.Model.TransDate},
			{"receiptNumber", t.Model.ReceiptNumber},
			{"status", t.Model.Status},
			{"lastBalance", t.Model.LastBalance},
			{"updatedAt", time.Now().UnixMilli()},
		}},
	}

	result, err := getDefaultCollection(transType).UpdateOne(
		context.TODO(),
		bson.D{{"referenceNo", t.Model.ReferenceNo}},
		update,
	)

	if err != nil {
		return err
	}

	if result.ModifiedCount == 0 {
		return errors.New(fmt.Sprintf("update failed, cannot find transaction with current referenceNo: %s", t.Model.ReferenceNo))
	}

	return nil
}
