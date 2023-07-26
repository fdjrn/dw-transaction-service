package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/fdjrn/dw-transaction-service/internal/db"
	"github.com/fdjrn/dw-transaction-service/internal/db/entity"
	"github.com/fdjrn/dw-transaction-service/internal/utilities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
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

	updateField := bson.D{
		{"transDate", t.Model.TransDate},
		{"transDateNumeric", t.Model.TransDateNumeric},
		{"receiptNumber", t.Model.ReceiptNumber},
		{"status", t.Model.Status},
		{"lastBalance", t.Model.LastBalance},
		{"updatedAt", time.Now().UnixMilli()},
	}

	if transType == utilities.TransTypeDistribution {
		updateField = append(updateField, bson.D{
			{"totalAmount", t.Model.TotalAmount},
			{"items", t.Model.Items},
		}...)
	}

	update := bson.D{{"$set", updateField}}
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

func (t *TransactionRepository) GetTransactionSummary(isPeriod bool) (int64, error) {

	filterData := bson.D{
		{"partnerId", t.Model.PartnerID},
		{"merchantId", t.Model.MerchantID},
		{"terminalId", t.Model.TerminalID},
		{"status", t.Model.Status},
	}

	if isPeriod {
		filterData = append(filterData, bson.D{
			{"transDateNumeric", bson.D{
				{"$gte", t.Model.Periods.StartDate.UnixMilli()},
				{"$lte", t.Model.Periods.EndDate.UnixMilli()},
			}},
		}...)
	}

	matchStages := bson.D{
		{"$match", filterData},
	}

	groupStages := bson.D{
		{"$group", bson.D{
			//{"_id", bson.D{
			//	{"partnerId", "$partnerId"},
			//	{"merchantId", "$merchantId"},
			//}},
			{"_id", primitive.Null{}},
			{"total", bson.D{{"$sum", "$totalAmount"}}},
		}},
	}

	projectStages := bson.D{
		{"$project", bson.D{
			{"_id", 0},
			//{"partnerId", "$_id.partnerId"},
			//{"merchantId", "$_id.merchantId"},
			{"total", 1},
		}},
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
	defer cancel()

	cursor, err := getDefaultCollection(t.Model.TransType).Aggregate(
		ctx,
		mongo.Pipeline{
			matchStages,
			groupStages,
			projectStages,
		})

	if err != nil {
		return 0, err
	}

	var summary map[string]interface{}
	for cursor.Next(context.TODO()) {
		err = cursor.Decode(&summary)
		if err != nil {
			log.Println("error found on cursor decode: ", err.Error())
			continue
		}
	}

	if summary == nil {
		return 0, mongo.ErrNoDocuments
	}

	return summary["total"].(int64), nil

}
