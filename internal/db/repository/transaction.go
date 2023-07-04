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

func (t *TransactionRepository) Create(transType string) (interface{}, error) {

	collection := new(mongo.Collection)
	switch transType {
	case utilities.TransTopUp:
		collection = db.Mongo.Collection.BalanceTopup
	case utilities.TransPayment:
		collection = db.Mongo.Collection.BalanceDeduct
	case utilities.TransDistribute:
		collection = db.Mongo.Collection.BalanceDistribute
	default:

	}

	result, err := collection.InsertOne(context.TODO(), t.Model)
	if err != nil {
		return nil, err
	}

	return result.InsertedID, nil
}

func (t *TransactionRepository) FindByID(id interface{}) (interface{}, error) {
	// filter condition
	filter := bson.D{{"_id", id}}

	var trans entity.BalanceTransaction
	err := db.Mongo.Collection.BalanceTopup.FindOne(context.TODO(), filter).Decode(&trans)
	if err != nil {
		return nil, err
	}

	return trans, nil
}

func (t *TransactionRepository) RemoveByID(id interface{}) error {
	filter := bson.D{{"_id", id}}
	result, err := db.Mongo.Collection.BalanceTopup.DeleteOne(context.TODO(), filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New(fmt.Sprintf("cannot find document with current id: %s", id))
	}

	return nil
}

func (t *TransactionRepository) IsUsedPartnerRefNumber(code string) bool {
	// filter condition
	filter := bson.D{{"partnerRefNumber", code}}

	var trans entity.BalanceTransaction
	err := db.Mongo.Collection.BalanceTopup.FindOne(context.TODO(), filter).Decode(&trans)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false
		}
		return true
	}

	return true
}

func (t *TransactionRepository) Update() error {
	update := bson.D{
		{"$set", bson.D{
			{"transDate", t.Model.TransDate},
			{"receiptNumber", t.Model.ReceiptNumber},
			{"status", t.Model.Status},
			{"lastBalance", t.Model.LastBalance},
			{"updatedAt", time.Now().UnixMilli()},
		}},
	}

	result, err := db.Mongo.Collection.BalanceTopup.UpdateOne(
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
