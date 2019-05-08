package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	str "strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	sc "github.com/hyperledger/fabric/protos/peer"
)

/*
	Method: queryOpenOrderItems
	Returns a list of of all lineitems from specific private collections based on the current logged in user org
	Data displayed in invetory manager and manufacturer 1 & 2 screens
*/
func (s *SmartContract) queryOpenOrderItems(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	queryString := "{\"selector\":{\"lineItems\": {\"$gt\": null }}}"
	collectionName := PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER1 // "org2msp|org3msp|org4msp"
	if str.ToLower(currentMspId) == "org4msp" {
		collectionName = PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER2
	} else if str.ToLower(currentMspId) == "org2msp" {
		collectionName = PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR
	}
	queryResults, err := getOpenOrderItemsPrivateDataQueryResults(stub, collectionName, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)

}

/*
	Method: queryLogisticsOpenOrderItems
	Returns all open order items for logistics operator screen
*/
func (s *SmartContract) queryLogisticsOpenOrderItems(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	queryString := "{\"selector\":{\"lineItems\": {\"$gt\": null }}}"
	collectionName := PRIVATE_COLLECTION_LOGISTICS
	queryResults, err := getLogisticsPrivateDataQueryResults(stub, collectionName, queryString)
	// getPrivateDataQueryResults
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

/*
	Method: queryFieldOperatorItems
	Given a specific "Mango formatted " query string execute query and return results.
*/
func (s *SmartContract) queryFieldOperatorItems(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	queryString := args[0] // "{\"selector\":{\"lineItems\": {\"$gt\": null }}}"
	queryResults, err := getFieldOperatorListResults(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

/*
	Method: queryMtrItems
	Returns a list of mtr records for either of the manufacturer
*/
func (s *SmartContract) queryMtrItems(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	queryString := "{\"selector\":{}}}"
	collectionName := PRIVATE_COLLECTION_MTR_MFR1
	if str.ToLower(currentMspId) == "org4msp" {
		collectionName = PRIVATE_COLLECTION_MTR_MFR2
	}
	queryResults, err := getMtrItemsResults(stub, collectionName, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

/*
	Method: queryPrivateCollection
	Returns a list of line items from a specific collection based and a specific poId
*/
func (s *SmartContract) queryPrivateCollection(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2; 1. private collection name 2. querystring ")
	}
	privateCollectionName := args[0]
	queryString := args[1]
	switch privateCollectionName {
	case PRIVATE_COLLECTION_MTR_MFR1, PRIVATE_COLLECTION_MTR_MFR2:
		queryResults, err := getMtrQueryResults(APIstub, privateCollectionName, queryString)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(queryResults)
	default:
		queryResults, err := getLineItemQueryResults(APIstub, privateCollectionName, queryString)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(queryResults)
	}

}

func (s *SmartContract) queryAll(APIstub shim.ChaincodeStubInterface) sc.Response {

	startKey := "" //leave key empty to retrive all data
	endKey := ""
	resultsIterator, err := APIstub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")
	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		po := fillPoFromQueryResponse(APIstub, queryResponse.Value)

		// po := PurchaseOrder{}
		// json.Unmarshal(queryResponse.Value, &po)
		// indexMap := make(map[string]int)
		// poPrivateDataResponse, err1 := APIstub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, po.PoId)
		// if err1 != nil {
		// 	logger.Info("Unable to get PRIVATE_COLLECTION_CUSTOMER_LINEITEMS data for PO: " + po.PoId)
		// } else {
		// 	if poPrivateDataResponse != nil {
		// 		pLineItem := LineItemPrivateDetails{}
		// 		json.Unmarshal(poPrivateDataResponse, &pLineItem)
		// 		po.LineItems = pLineItem.LineItems
		// 		for i, lineItem := range po.LineItems {
		// 			indexMap[lineItem.ItemKey] = i
		// 		}
		// 	}
		// }

		// if len(po.LineItems) > 0 {

		// 	poPrivateDataResponse, err1 := APIstub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR, po.PoId)
		// 	if err1 != nil {
		// 		logger.Info("Unable to get PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR data for PO: " + po.PoId)
		// 	} else {
		// 		if poPrivateDataResponse != nil {
		// 			privateData := LineItemCDPrivateDetails{}
		// 			json.Unmarshal(poPrivateDataResponse, &privateData)
		// 			for _, priceInfo := range privateData.LineItems {
		// 				index, found := indexMap[priceInfo.ItemKey]
		// 				if !found {
		// 					continue
		// 				}
		// 				po.LineItems[index].UnitPrice = priceInfo.UnitPrice
		// 				po.LineItems[index].Subtotal = priceInfo.Subtotal
		// 				po.LineItems[index].Quantity = priceInfo.Quantity
		// 			}
		// 		} else {
		// 			fmt.Println("private data not found for " + po.PoId)
		// 		}
		// 	}

		// 	poPrivateDataResponse, err1 = APIstub.GetPrivateData(PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER1, po.PoId)
		// 	if err1 != nil {
		// 		logger.Info("Unable to get PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER data for PO: " + po.PoId)
		// 	} else {
		// 		if poPrivateDataResponse != nil {
		// 			privateData := LineItemCDPrivateDetails{}
		// 			json.Unmarshal(poPrivateDataResponse, &privateData)
		// 			for _, priceInfo := range privateData.LineItems {
		// 				index, found := indexMap[priceInfo.ItemKey]
		// 				if !found {
		// 					continue
		// 				}
		// 				po.LineItems[index].MfrUnitPrice = priceInfo.UnitPrice
		// 				po.LineItems[index].AssignedQty = priceInfo.Quantity
		// 			}
		// 		}
		// 	}
		// }
		// Add comma before array members,suppress it for the first array member

		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		poAsBytes, _ := json.Marshal(po)
		buffer.WriteString(string(poAsBytes))
		// buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")
	return shim.Success(buffer.Bytes())
}
func (s *SmartContract) queryAllCustomer(APIstub shim.ChaincodeStubInterface) sc.Response {

	startKey := "" //leave key empty to retrive all data
	endKey := ""
	resultsIterator, err := APIstub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	// var logger = shim.NewLogger("material-trace")
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		po := fillPoFromQueryResponse(APIstub, queryResponse.Value)
		// po := PurchaseOrder{}
		// json.Unmarshal(queryResponse.Value, &po)
		// indexMap := make(map[string]int)
		// poPrivateDataResponse, err1 := APIstub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, po.PoId)
		// if err1 != nil {
		// 	logger.Info("Unable to get PRIVATE_COLLECTION_CUSTOMER_LINEITEMS data for PO: " + po.PoId)
		// } else {
		// 	if poPrivateDataResponse != nil {
		// 		pLineItem := LineItemPrivateDetails{}
		// 		json.Unmarshal(poPrivateDataResponse, &pLineItem)
		// 		po.LineItems = pLineItem.LineItems
		// 		for i, lineItem := range po.LineItems {
		// 			indexMap[lineItem.ItemKey] = i
		// 		}
		// 	}
		// }
		// // len(currentLineItems) > 0
		// if len(po.LineItems) > 0 {

		// 	poPrivateDataResponse, err1 := APIstub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR, po.PoId)
		// 	if err1 != nil {
		// 		logger.Info("Unable to get PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR data for PO: " + po.PoId)
		// 	} else {
		// 		if poPrivateDataResponse != nil {
		// 			privateData := LineItemCDPrivateDetails{}
		// 			json.Unmarshal(poPrivateDataResponse, &privateData)
		// 			for _, priceInfo := range privateData.LineItems {
		// 				index, found := indexMap[priceInfo.ItemKey]
		// 				if !found {
		// 					continue
		// 				}
		// 				po.LineItems[index].UnitPrice = priceInfo.UnitPrice
		// 				po.LineItems[index].Subtotal = priceInfo.Subtotal
		// 				po.LineItems[index].Quantity = priceInfo.Quantity
		// 			}
		// 		} else {
		// 			fmt.Println("private data not found for " + po.PoId)
		// 		}
		// 	}
		// }

		// Add comma before array members,suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		poAsBytes, _ := json.Marshal(po)
		buffer.WriteString(string(poAsBytes))
		// buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")
	return shim.Success(buffer.Bytes())
}

/*
	Method: fetchAllShippedItems
	Returns a list of shipped lineitems
*/
func (s *SmartContract) fetchAllShippedItems(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	shippingRequestsResults, err := shippedItemsList(stub)
	if err != nil {
		return shim.Error("Error fetching data")
	}
	itemAsBytes, _ := json.Marshal(shippingRequestsResults)
	return shim.Success(itemAsBytes)
}

/*
	Method: queryLineItemStatus
	Returns status result for a specific ponumber and lineItem
*/
func (s *SmartContract) queryLineItemStatus(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2; 1. poNumber 2. lineNumber")
	}
	poNumber, parseErr := strconv.Atoi(args[0])
	if parseErr != nil {
		return shim.Error("Invalid first argument, expected value must a number")
	}
	lineNumber, parseErr := strconv.Atoi(args[1])
	if parseErr != nil {
		return shim.Error("Invalid second argument, expected value must a number")
	}
	queryString := `{"selector":{"lineItems":{"$elemMatch":{"poNumber":{"$eq":` + strconv.Itoa(poNumber) + `}}}}}`
	logger.Infof("QueryString Passed: %s", queryString)
	resultsIterator, err := stub.GetPrivateDataQueryResult(PRIVATE_COLLECTION_GENERAL_PROGRESS, queryString)
	if err != nil {
		return shim.Error("Not Found")
	}
	defer resultsIterator.Close()
	targetLineItem := LineItem{}
	found := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error("Fetching results")
		}
		pLineItem := LineItemPrivateDetails{}
		json.Unmarshal(queryResponse.Value, &pLineItem)
		for _, item := range pLineItem.LineItems {
			if item.PoNumber != poNumber {
				continue
			}
			if item.LineNumber != lineNumber {
				continue
			}
			targetLineItem = item
			found = true
			break
		}
		if found {
			break
		}
	}
	itemAsBytes, _ := json.Marshal(targetLineItem)
	return shim.Success(itemAsBytes)
}

/*
	Method: queryPo
	Execute specified query for a po
*/
func (s *SmartContract) queryPo(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	queryString := args[0]
	queryResults, err := getMergedResultForQueryString(APIstub, queryString) // getQueryResultForQueryString(APIstub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

/*
	Utility method to merge private collection data into one view for distributor and customer views
*/
func getMergedResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {

	poItems, err := getMergedPoList(stub, queryString)
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	buffer.WriteString("[")
	bArrayMemberAlreadyWritten := false
	for _, po := range poItems {
		// Add a comma before array members, suppress it for the first array member
		if po.PoNumber == 0 {
			continue
		}
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(po.PoId)
		buffer.WriteString("\"")
		buffer.WriteString(", \"Record\":")
		resultAsBytes, _ := json.Marshal(po)
		buffer.WriteString(string(resultAsBytes))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil

	// resultsIterator, err := stub.GetQueryResult(queryString)
	// if err != nil {
	// 	return nil, err
	// }
	// defer resultsIterator.Close()

	// // buffer is a JSON array containing QueryRecords
	// var buffer bytes.Buffer
	// buffer.WriteString("[")
	// bArrayMemberAlreadyWritten := false
	// for resultsIterator.HasNext() {
	// 	queryResponse, err := resultsIterator.Next()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	// Add a comma before array members, suppress it for the first array member
	// 	if bArrayMemberAlreadyWritten == true {
	// 		buffer.WriteString(",")
	// 	}
	// 	buffer.WriteString("{\"Key\":")
	// 	buffer.WriteString("\"")
	// 	buffer.WriteString(queryResponse.Key)
	// 	buffer.WriteString("\"")

	// 	buffer.WriteString(", \"Record\":")

	// 	po := PurchaseOrder{}
	// 	json.Unmarshal(queryResponse.Value, &po)
	// 	indexMap := make(map[string]int)
	// 	poPrivateDataResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, po.PoId)
	// 	if err1 != nil {
	// 		logger.Info("Unable to get PRIVATE_COLLECTION_CUSTOMER_LINEITEMS data for PO: " + po.PoId)
	// 	} else {

	// 		if poPrivateDataResponse != nil {
	// 			pLineItem := LineItemPrivateDetails{}
	// 			json.Unmarshal(poPrivateDataResponse, &pLineItem)
	// 			po.LineItems = pLineItem.LineItems
	// 			for i, lineItem := range po.LineItems {
	// 				indexMap[lineItem.ItemKey] = i
	// 			}
	// 		}
	// 	}
	// 	if len(po.LineItems) > 0 {

	// 		poPrivateDataResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR, po.PoId)
	// 		if err1 != nil {
	// 			logger.Info("Unable to get PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR data for PO: " + po.PoId)
	// 		} else {
	// 			if poPrivateDataResponse != nil {
	// 				privateData := LineItemCDPrivateDetails{}
	// 				json.Unmarshal(poPrivateDataResponse, &privateData)
	// 				for _, priceInfo := range privateData.LineItems {
	// 					index, found := indexMap[priceInfo.ItemKey]
	// 					if !found {
	// 						continue
	// 					}
	// 					po.LineItems[index].UnitPrice = priceInfo.UnitPrice
	// 					po.LineItems[index].Subtotal = priceInfo.Subtotal
	// 					po.LineItems[index].Quantity = priceInfo.Quantity
	// 					po.LineItems[index].MaterialCertificate = priceInfo.MaterialCertificate
	// 					po.LineItems[index].IotTrackingCode = priceInfo.IotTrackingCode
	// 					po.LineItems[index].IotProperties = priceInfo.IotProperties
	// 					if len(po.LineItems[index].OrderRequests) > 0 {
	// 						po.LineItems[index].OrderRequests[0].ProgressStatus = priceInfo.ProgressStatus
	// 						po.LineItems[index].OrderRequests[0].AcknowledgedTimeStamp = po.AcceptanceTimeStamp
	// 						po.LineItems[index].OrderRequests[0].TimeShipped = priceInfo.TimeShipped
	// 					}
	// 				}
	// 			} else {
	// 				logger.Info("PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR no private data not found for " + po.PoId)
	// 			}
	// 		}
	// 		poPrivateDataResponse, err1 = stub.GetPrivateData(PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER1, po.PoId)
	// 		if err1 != nil {
	// 			logger.Info("Unable to get PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER data for PO: " + po.PoId)
	// 		} else {
	// 			if poPrivateDataResponse != nil {
	// 				privateData := LineItemCDPrivateDetails{}
	// 				json.Unmarshal(poPrivateDataResponse, &privateData)
	// 				for _, priceInfo := range privateData.LineItems {
	// 					index, found := indexMap[priceInfo.ItemKey]
	// 					if !found {
	// 						continue
	// 					}
	// 					po.LineItems[index].MaterialCertificate = priceInfo.MaterialCertificate
	// 					po.LineItems[index].IotTrackingCode = priceInfo.IotTrackingCode
	// 					po.LineItems[index].IotProperties = priceInfo.IotProperties
	// 					if len(po.LineItems[index].OrderRequests) > 0 {
	// 						po.LineItems[index].OrderRequests[0].ProgressStatus = priceInfo.ProgressStatus
	// 						po.LineItems[index].OrderRequests[0].AcknowledgedTimeStamp = priceInfo.AcknowledgedTimeStamp
	// 						po.LineItems[index].OrderRequests[0].TimeShipped = priceInfo.TimeShipped
	// 					}
	// 				}
	// 			}
	// 		}
	// 		poPrivateDataResponse, err1 = stub.GetPrivateData(PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER2, po.PoId)
	// 		if err1 != nil {
	// 			logger.Info("Unable to get PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER data for PO: " + po.PoId)
	// 		} else {
	// 			if poPrivateDataResponse != nil {
	// 				privateData := LineItemCDPrivateDetails{}
	// 				json.Unmarshal(poPrivateDataResponse, &privateData)
	// 				for _, priceInfo := range privateData.LineItems {
	// 					index, found := indexMap[priceInfo.ItemKey]
	// 					if !found {
	// 						continue
	// 					}
	// 					po.LineItems[index].MaterialCertificate = priceInfo.MaterialCertificate
	// 					po.LineItems[index].IotTrackingCode = priceInfo.IotTrackingCode
	// 					po.LineItems[index].IotProperties = priceInfo.IotProperties
	// 					if len(po.LineItems[index].OrderRequests) > 0 {
	// 						po.LineItems[index].OrderRequests[0].ProgressStatus = priceInfo.ProgressStatus
	// 						po.LineItems[index].OrderRequests[0].AcknowledgedTimeStamp = priceInfo.AcknowledgedTimeStamp
	// 						po.LineItems[index].OrderRequests[0].TimeShipped = priceInfo.TimeShipped
	// 					}
	// 				}
	// 			}
	// 		}
	// 	}

	// 	poAsBytes, _ := json.Marshal(po)
	// 	buffer.WriteString(string(poAsBytes))
	// 	buffer.WriteString("}")
	// 	bArrayMemberAlreadyWritten = true
	// }
	// buffer.WriteString("]")
	// return buffer.Bytes(), nil
}
func getMergedPoList(stub shim.ChaincodeStubInterface, queryString string) ([]PurchaseOrder, error) {

	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}

	defer resultsIterator.Close()
	purchaseOrders := make([]PurchaseOrder, 1)
	count := 0
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		po := fillPoFromQueryResponse(stub, queryResponse.Value)
		if count == 0 {
			purchaseOrders[0] = po
		} else {
			purchaseOrders = append(purchaseOrders, po)
		}
		count += 1
	}
	logger.Infof(" querystring %s about to return poItems length %d ", queryString, len(purchaseOrders))
	logger.Infof(" about to return first po lineItems length %d ", len(purchaseOrders[0].LineItems))

	return purchaseOrders, nil
}
func fillPoFromQueryResponse(stub shim.ChaincodeStubInterface, responseValue []byte) PurchaseOrder {
	po := PurchaseOrder{}
	json.Unmarshal(responseValue, &po)
	indexMap := make(map[string]int)
	indexByLineNumberMap := make(map[int]int)
	poPrivateDataResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, po.PoId)
	if err1 != nil {
		logger.Info("Unable to get PRIVATE_COLLECTION_CUSTOMER_LINEITEMS data for PO: " + po.PoId)
	} else {
		if poPrivateDataResponse != nil {
			pLineItem := LineItemPrivateDetails{}
			json.Unmarshal(poPrivateDataResponse, &pLineItem)
			po.LineItems = pLineItem.LineItems
			for i, lineItem := range po.LineItems {
				indexMap[lineItem.ItemKey] = i
				indexByLineNumberMap[lineItem.LineNumber] = i
				logger.Infof("initial map - key: %s index: %d ", lineItem.ItemKey, i)
			}
		}
	}

	if len(po.LineItems) > 0 {

		poPrivateDataResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR, po.PoId)
		if err1 != nil {
			// logger.Info("Unable to get PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR data for PO: " + po.PoId)
		} else {
			if poPrivateDataResponse != nil {
				privateData := LineItemCDPrivateDetails{}
				json.Unmarshal(poPrivateDataResponse, &privateData)
				for _, priceInfo := range privateData.LineItems {
					index, found := indexMap[priceInfo.ItemKey]
					if !found {
						continue
					}
					logger.Infof("in distributor section - key: %s index: %d ", priceInfo.ItemKey, index)
					po.LineItems[index].UnitPrice = priceInfo.UnitPrice
					po.LineItems[index].Subtotal = priceInfo.Subtotal
					po.LineItems[index].Quantity = priceInfo.Quantity
					po.LineItems[index].MaterialCertificate = priceInfo.MaterialCertificate
					po.LineItems[index].IotTrackingCode = priceInfo.IotTrackingCode
					po.LineItems[index].IotProperties = priceInfo.IotProperties
					if len(po.LineItems[index].OrderRequests) > 0 {
						po.LineItems[index].OrderRequests[0].ProgressStatus = priceInfo.ProgressStatus
						po.LineItems[index].OrderRequests[0].AcknowledgedTimeStamp = po.AcceptanceTimeStamp
						po.LineItems[index].OrderRequests[0].TimeShipped = priceInfo.TimeShipped
					}
				}
			} else {
				logger.Info("PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR no private data not found for " + po.PoId)
			}
		}
		poPrivateDataResponse, err1 = stub.GetPrivateData(PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER1, po.PoId)
		if err1 != nil {
			logger.Info("Unable to get PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER1 for PO: " + po.PoId)
		} else {
			if poPrivateDataResponse != nil {
				privateData := LineItemCDPrivateDetails{}
				json.Unmarshal(poPrivateDataResponse, &privateData)
				for _, priceInfo := range privateData.LineItems {
					index, found := indexMap[priceInfo.ItemKey]
					if !found {
						continue
					}
					logger.Infof("in manufacturer 1 section - key: %s index: %d ", priceInfo.ItemKey, index)
					if len(po.LineItems) < index {
						logger.Infof("in manufacturer 1 section something is wrong with this - key: %s index: %d  po.LineItems length: %d", priceInfo.ItemKey, index, len(po.LineItems))
					} else {
						po.LineItems[index].MaterialCertificate = priceInfo.MaterialCertificate
						po.LineItems[index].IotTrackingCode = priceInfo.IotTrackingCode
						po.LineItems[index].IotProperties = priceInfo.IotProperties
						if len(po.LineItems[index].OrderRequests) > 0 {
							po.LineItems[index].OrderRequests[0].ProgressStatus = priceInfo.ProgressStatus
							po.LineItems[index].OrderRequests[0].AcknowledgedTimeStamp = priceInfo.AcknowledgedTimeStamp
							po.LineItems[index].OrderRequests[0].TimeShipped = priceInfo.TimeShipped
						}
					}
				}
			}
		}
		poPrivateDataResponse, err1 = stub.GetPrivateData(PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER2, po.PoId)
		if err1 != nil {
			logger.Info("Unable to get PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER2 data for PO: " + po.PoId)
		} else {
			if poPrivateDataResponse != nil {
				privateData := LineItemCDPrivateDetails{}
				json.Unmarshal(poPrivateDataResponse, &privateData)
				for _, priceInfo := range privateData.LineItems {
					index, found := indexMap[priceInfo.ItemKey]
					if !found {
						continue
					}
					logger.Infof("in manufacturer 2 section - key: %s index: %d ", priceInfo.ItemKey, index)
					if len(po.LineItems) < index {
						logger.Infof("in manufacturer 2 section something is wrong with this - key: %s index: %d  po.LineItems length: %d", priceInfo.ItemKey, index, len(po.LineItems))
					} else {
						po.LineItems[index].MaterialCertificate = priceInfo.MaterialCertificate
						po.LineItems[index].IotTrackingCode = priceInfo.IotTrackingCode
						po.LineItems[index].IotProperties = priceInfo.IotProperties
						if len(po.LineItems[index].OrderRequests) > 0 {
							po.LineItems[index].OrderRequests[0].ProgressStatus = priceInfo.ProgressStatus
							po.LineItems[index].OrderRequests[0].AcknowledgedTimeStamp = priceInfo.AcknowledgedTimeStamp
							po.LineItems[index].OrderRequests[0].TimeShipped = priceInfo.TimeShipped
						}
					}
				}
			}
		}
		poPrivateDataResponse, err1 = stub.GetPrivateData(PRIVATE_COLLECTION_LOGISTICS, po.PoId)
		if err1 != nil {
			// continue

		} else {
			if poPrivateDataResponse != nil {
				privateData := ShippingRequest{}
				json.Unmarshal(poPrivateDataResponse, &privateData)
				for _, shippingInfo := range privateData.LineItems {
					index, found := indexByLineNumberMap[shippingInfo.LineNumber]
					if !found {
						continue
					}
					if po.LineItems[index].ShippingRequestNumber == 0 {
						logger.Infof("set shipping request number from logistics table for line number %d ", po.LineItems[index].LineNumber)
						po.LineItems[index].ShippingRequestNumber = shippingInfo.ShippingRequestNumber
					}
					for _, statusInfo := range shippingInfo.ProgressStatus {
						if statusInfo.Status == "received" && po.LineItems[index].TimeReceived == 0 {
							po.LineItems[index].TimeReceived = statusInfo.TimeStamp
							logger.Infof("set TimeReceived from logistics table for line number %d ", po.LineItems[index].LineNumber)
							break
						}
					}
				}
			}
		}
	}
	return po
}
func getQueryResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {

	fmt.Printf("- getQueryResultForQueryString queryString:\n%s\n", queryString)
	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryRecords
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")

		po := PurchaseOrder{}
		json.Unmarshal(queryResponse.Value, &po)
		poPrivateDataResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, po.PoId)
		if err1 != nil {
			logger.Info("Unable to get PRIVATE_COLLECTION_CUSTOMER_LINEITEMS data for PO: " + po.PoId)
		} else {
			if poPrivateDataResponse != nil {
				pLineItem := LineItemPrivateDetails{}
				json.Unmarshal(poPrivateDataResponse, &pLineItem)
				po.LineItems = pLineItem.LineItems
			}
		}
		poAsBytes, _ := json.Marshal(po)
		buffer.WriteString(string(poAsBytes))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}
func getMtrItemsResults(stub shim.ChaincodeStubInterface, privateCollectionName string, queryString string) ([]byte, error) {
	resultsIterator, err := stub.GetPrivateDataQueryResult(privateCollectionName, queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()
	var buffer bytes.Buffer
	buffer.WriteString("[")
	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString(string(queryResponse.Value))
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}
func getLogisticsPrivateDataQueryResults(stub shim.ChaincodeStubInterface, privateCollectionName string, queryString string) ([]byte, error) {
	fmt.Printf("- collection %s getQueryResultForQueryString queryString: \n%s\n", privateCollectionName, queryString)
	resultsIterator, err := stub.GetPrivateDataQueryResult(privateCollectionName, queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryRecords
	shippingRequestsResults := make([]ShippingRequestsResults, 1)
	prCount := 0
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		shippingPd := ShippingPrivateDetails{}
		json.Unmarshal(queryResponse.Value, &shippingPd)
		srr := ShippingRequestsResults{}

		if value, err := stub.GetState(shippingPd.PoId); err == nil && value != nil {
			po := PurchaseOrder{}
			json.Unmarshal(value, &po)
			srr.PoId = po.PoId
			srr.PoNumber = po.PoNumber
			srr.Owner = po.Owner
			srr.LineItems = shippingPd.LineItems
			srr.ExpectedDeliveryDate = po.ExpectedDeliveryDate
			srr.PoStatus = po.PoStatus
			if prCount == 0 {
				shippingRequestsResults[0] = srr
			} else {
				shippingRequestsResults = append(shippingRequestsResults, srr)
			}
			prCount += 1
		}
	}
	var buffer bytes.Buffer
	buffer.WriteString("[")
	bArrayMemberAlreadyWritten := false
	for _, item := range shippingRequestsResults {
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		resultAsBytes, _ := json.Marshal(item)
		buffer.WriteString(string(resultAsBytes))
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return buffer.Bytes(), nil
}
func getFieldOperatorResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {

	fmt.Printf("- getFieldOperatorResultForQueryString - queryString:\n%s\n", queryString)
	poItems, err := getMergedPoList(stub, queryString)
	if err != nil {
		return nil, err
		//  var buffer bytes.Buffer
		//  buffer.WriteString("[]")
		//  return buffer.Bytes(), nil
	}
	logger.Infof("getFieldOperatorResultForQueryString: getMergedPoListlength: %d ", len(poItems))

	pmViewResults := make([]PmView, 1)
	prCount := 0
	for _, po := range poItems {
		if po.PoStatus != "accepted" {
			logger.Infof("getFieldOperatorResultForQueryString: po.PoStatus : %s poNumber: %d ", po.PoStatus, po.PoNumber)
			continue
		}
		pmview := PmView{}
		filteredLineItems := make([]LineItem, 1)
		count := 0
		for i, item := range po.LineItems {

			ok := item.ShippingRequestNumber > 0 // (item.Status == STATUS_RECEIVED || item.Status == STATUS_VERIFIED || item.Status == STATUS_DELIVERED)
			logger.Infof(" index %d status %s item.ShippingRequestNumber: %d", i, item.Status, item.ShippingRequestNumber)
			if ok {
				logger.Infof(" ok index %d status %s shipping number: %d ", i, item.Status, item.ShippingRequestNumber)
				if count == 0 {
					filteredLineItems[0] = item
				} else {
					filteredLineItems = append(filteredLineItems, item)
				}
				count += 1
			} else {
				logger.Infof(" not okay index %d status %s ", i, item.Status)
			}
			// count += 1
		}
		if count > 0 {
			pmview.PoId = po.PoId
			pmview.PoNumber = po.PoNumber
			pmview.LineItems = filteredLineItems
			pmview.PoStatus = po.PoStatus
			if prCount == 0 {
				pmViewResults[0] = pmview
			} else {
				pmViewResults = append(pmViewResults, pmview)
			}
			prCount += 1
		}
	}

	var buffer bytes.Buffer
	buffer.WriteString("[")
	bArrayMemberAlreadyWritten := false
	if prCount > 0 {
		for _, item := range pmViewResults {
			// Add a comma before array members, suppress it for the first array member
			if bArrayMemberAlreadyWritten == true {
				buffer.WriteString(",")
			}
			resultAsBytes, _ := json.Marshal(item)
			buffer.WriteString(string(resultAsBytes))
			bArrayMemberAlreadyWritten = true
		}
	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil

}
func getOpenOrderItemsPrivateDataQueryResults(stub shim.ChaincodeStubInterface, privateCollectionName string, queryString string) ([]byte, error) {
	fmt.Printf("- collection %s getQueryResultForQueryString queryString: \n%s\n", privateCollectionName, queryString)
	resultsIterator, err := stub.GetPrivateDataQueryResult(privateCollectionName, queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryRecords
	prResults := make([]PricingResults, 1)
	prCount := 0
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		pricing := LineItemCDPrivateDetails{}
		json.Unmarshal(queryResponse.Value, &pricing)
		pr := PricingResults{}

		if value, err := stub.GetState(pricing.PoId); err == nil && value != nil {
			po := PurchaseOrder{}
			json.Unmarshal(value, &po)
			pr.PoId = po.PoId
			pr.PoNumber = po.PoNumber
			pr.Owner = po.Owner
			pr.LineItems = pricing.LineItems
			pr.ExpectedDeliveryDate = po.ExpectedDeliveryDate
			pr.PoStatus = po.PoStatus
			if prCount == 0 {
				prResults[0] = pr
			} else {
				prResults = append(prResults, pr)
			}
			prCount += 1
		}
	}
	var buffer bytes.Buffer
	buffer.WriteString("[")
	bArrayMemberAlreadyWritten := false
	for _, item := range prResults {
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		resultAsBytes, _ := json.Marshal(item)
		buffer.WriteString(string(resultAsBytes))
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return buffer.Bytes(), nil
}
func getMtrQueryResults(stub shim.ChaincodeStubInterface, privateCollectionName string, queryString string) ([]byte, error) {
	if str.ToLower(privateCollectionName) != str.ToLower(PRIVATE_COLLECTION_MTR_MFR1) && str.ToLower(privateCollectionName) != str.ToLower(PRIVATE_COLLECTION_MTR_MFR2) {
		return nil, nil
	}
	resultsIterator, err := stub.GetPrivateDataQueryResult(privateCollectionName, queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryRecords
	var buffer bytes.Buffer
	buffer.WriteString("[")
	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		mtr := MaterialCertificate{}
		json.Unmarshal(queryResponse.Value, &mtr)

		// Add comma before array members,suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		mtrBytes, _ := json.Marshal(mtr)
		buffer.WriteString(string(mtrBytes))
		// buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true

	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil

}
func getLineItemQueryResults(stub shim.ChaincodeStubInterface, privateCollectionName string, queryString string) ([]byte, error) {
	if str.ToLower(privateCollectionName) != str.ToLower(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS) {
		return nil, nil
	}
	resultsIterator, err := stub.GetPrivateDataQueryResult(privateCollectionName, queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()
	// buffer is a JSON array containing QueryRecords
	var buffer bytes.Buffer
	buffer.WriteString("[")
	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		lineItemDetails := LineItemPrivateDetails{}
		json.Unmarshal(queryResponse.Value, &lineItemDetails)

		// Add comma before array members,suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		liBytes, _ := json.Marshal(lineItemDetails)
		buffer.WriteString(string(liBytes))
		// Record is a JSON object, so we write as-is
		// buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true

	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil

}
func (s *SmartContract) getHistoryForSpecificPO(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	historyIterator, err := stub.GetHistoryForKey(args[0])
	if err != nil {
		fmt.Println(err.Error())
		return shim.Error(err.Error())
	}
	itemHistory := make([]PurchaseOrder, 1)
	index := 0
	for historyIterator.HasNext() {
		item, err := historyIterator.Next()
		if err != nil {
			fmt.Println(err.Error())
			return shim.Error(err.Error())
		}
		po := PurchaseOrder{}
		json.Unmarshal(item.Value, &po)
		if index == 0 {
			itemHistory[0] = po
		} else {
			itemHistory = append(itemHistory, po)
		}
		index += 1
	}

	poArrayAsBytes, _ := json.Marshal(itemHistory) // convert array of PO struct into bytes
	return shim.Success(poArrayAsBytes)

}

// field operator fixes
/*
	Utility method for queryFieldOperatorItems, exutes query returns results object
*/
func getFieldOperatorListResults(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {
	reportItems, err := getFieldOperatorList(stub)
	if err != nil {
		return nil, err
	}
	logger.Infof("in method getFieldOperatorResults: poItems Length: %d ", len(reportItems))
	var buffer bytes.Buffer
	buffer.WriteString("[")
	bArrayMemberAlreadyWritten := false
	if len(reportItems) > 0 {
		for _, item := range reportItems {
			// Add a comma before array members, suppress it for the first array member
			if bArrayMemberAlreadyWritten == true {
				buffer.WriteString(",")
			}
			resultAsBytes, _ := json.Marshal(item)
			buffer.WriteString(string(resultAsBytes))
			bArrayMemberAlreadyWritten = true
		}
	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}
func getFieldOperatorList(stub shim.ChaincodeStubInterface) ([]FieldOperatorReport, error) {
	shippingRequestsResults, err := shippedItemsList(stub)
	if err != nil {
		return nil, err
	}
	foReportItems := make([]FieldOperatorReport, 1)
	poCount := 0
	for _, item := range shippingRequestsResults {
		shippedItemsMap := make(map[int]ShippingLineItem)
		shippingRequestMap := make(map[int64][]ShippingLineItem)
		appendMapCount := 0
		for _, line := range item.LineItems {
			if line.Status != "open" {
				shippedItemsMap[line.LineNumber] = line
				if appendMapCount == 0 {
					initialArray := make([]ShippingLineItem, 1)
					initialArray[0] = line
					shippingRequestMap[line.ShippingRequestNumber] = initialArray
				} else {
					shippingRequestMap[line.ShippingRequestNumber] = append(shippingRequestMap[line.ShippingRequestNumber], line)
				}
				appendMapCount += 1
			}
		}

		poPrivateDataResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, item.PoId)
		if err1 != nil {
			logger.Info("Unable to get PRIVATE_COLLECTION_CUSTOMER_LINEITEMS data for PO: " + item.PoId)
		} else {
			if poPrivateDataResponse != nil {
				reportItem := FieldOperatorReport{}
				reportItem.ShippedItemsMap = shippedItemsMap
				reportItem.ShippingRequestMap = shippingRequestMap
				reportItem.DistributorLineItemMap = getLineItemMapForACollection(stub, item.PoId, PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR)
				reportItem.Manufacturer1LineItemMap = getLineItemMapForACollection(stub, item.PoId, PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER1)
				reportItem.Manufacturer2LineItemMap = getLineItemMapForACollection(stub, item.PoId, PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER2)
				reportItem.ProgressReportMap = getGeneralProgressMapForACollection(stub, item.PoId)
				if value, err := stub.GetState(item.PoId); err == nil && value != nil {
					po := PurchaseOrder{}
					json.Unmarshal(value, &po)
					reportItem.OriginalPo = po
				}
				privateData := LineItemPrivateDetails{}
				json.Unmarshal(poPrivateDataResponse, &privateData)
				pmView := PmView{}
				pmView.PoId = item.PoId
				pmView.PoNumber = item.PoNumber
				pmView.PoStatus = item.PoStatus
				pmView.LineItems = make([]LineItem, 1)
				itemCount := 0
				for _, lineItem := range privateData.LineItems {
					shippedLine, found := shippedItemsMap[lineItem.LineNumber]
					if found {
						lineItem.ShippingRequestNumber = shippedLine.ShippingRequestNumber
						if itemCount == 0 {
							pmView.LineItems[0] = lineItem
						} else {
							pmView.LineItems = append(pmView.LineItems, lineItem)
						}
						itemCount += 1
					}
				}
				reportItem.PmViewItem = pmView
				if poCount == 0 {
					foReportItems[0] = reportItem
				} else {
					foReportItems = append(foReportItems, reportItem)
				}
				poCount += 1
			}
		}
	}
	return foReportItems, nil
}

/*
	Method: getLineItemMapForACollection
	Given a private collection name and a poid, return a map of lineItems keyed by lineNUmber
*/
func getLineItemMapForACollection(stub shim.ChaincodeStubInterface, poId string, privateCollectionName string) map[int]LineItemPricing {
	lineItemMap := make(map[int]LineItemPricing)
	poPrivateDataResponse, err1 := stub.GetPrivateData(privateCollectionName, poId)
	if err1 != nil {
		logger.Infof("Unable to get %s data for PO: %s ", privateCollectionName, poId)
	} else {
		if poPrivateDataResponse != nil {
			privateData := LineItemCDPrivateDetails{}
			json.Unmarshal(poPrivateDataResponse, &privateData)
			for _, priceInfo := range privateData.LineItems {
				lineItemMap[priceInfo.LineNumber] = priceInfo
			}
		} else {
			logger.Infof("%s has data for PO: %s ", privateCollectionName, poId)
		}
	}
	return lineItemMap
}

/*
	Method: getGeneralProgressMapForACollection
	A helper method that returns a map of lineitems with a map key been the lineNumber
	for a specified poId
*/
func getGeneralProgressMapForACollection(stub shim.ChaincodeStubInterface, poId string) map[int]SharedLineDetail {
	lineItemMap := make(map[int]SharedLineDetail)
	poPrivateDataResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_GENERAL_PROGRESS, poId)
	if err1 != nil {
		logger.Infof("Unable to get %s data for PO: %s ", PRIVATE_COLLECTION_GENERAL_PROGRESS, poId)
	} else {
		if poPrivateDataResponse != nil {
			privateData := SharedProgressReport{}
			json.Unmarshal(poPrivateDataResponse, &privateData)
			for _, item := range privateData.LineItems {
				lineItemMap[item.LineNumber] = item
			}
		} else {
			logger.Infof("%s has data for PO: %s ", PRIVATE_COLLECTION_GENERAL_PROGRESS, poId)
		}
	}
	return lineItemMap
}

/*
	Method: getSpecificLineItemForACollection
	A helper method that returns a specific lineItem for a po in a private collection
	This will only return if data if the current user org is authorized to have access to the data
*/
func getSpecificLineItemForACollection(stub shim.ChaincodeStubInterface, poId string, lineNumber int, privateCollectionName string) LineItemPricing {
	pricingItem := LineItemPricing{}
	poPrivateDataResponse, err1 := stub.GetPrivateData(privateCollectionName, poId)
	if err1 != nil {
		logger.Infof("Unable to get %s data for PO: %s ", privateCollectionName, poId)
	} else {
		if poPrivateDataResponse != nil {
			privateData := LineItemCDPrivateDetails{}
			json.Unmarshal(poPrivateDataResponse, &privateData)
			for _, priceInfo := range privateData.LineItems {
				if priceInfo.LineNumber == lineNumber {
					pricingItem = priceInfo
				}
			}
		} else {
			logger.Infof("%s has data for PO: %s ", privateCollectionName, poId)
		}
	}
	return pricingItem
}

/*
	Method: shippedItemsList
	A helper method that returns all items in logistics collection
*/
func shippedItemsList(stub shim.ChaincodeStubInterface) ([]ShippingRequestsResults, error) {
	queryString := "{\"selector\":{\"lineItems\": {\"$gt\": null }}}"
	privateCollectionName := PRIVATE_COLLECTION_LOGISTICS
	fmt.Printf("- collection %s getQueryResultForQueryString queryString: \n%s\n", privateCollectionName, queryString)
	resultsIterator, err := stub.GetPrivateDataQueryResult(privateCollectionName, queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()
	shippingRequestsResults := make([]ShippingRequestsResults, 1)
	prCount := 0
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		shippingPd := ShippingPrivateDetails{}
		json.Unmarshal(queryResponse.Value, &shippingPd)
		srr := ShippingRequestsResults{}

		if value, err := stub.GetState(shippingPd.PoId); err == nil && value != nil {
			po := PurchaseOrder{}
			json.Unmarshal(value, &po)
			srr.PoId = po.PoId
			srr.PoNumber = po.PoNumber
			srr.Owner = po.Owner
			srr.LineItems = shippingPd.LineItems
			srr.ExpectedDeliveryDate = po.ExpectedDeliveryDate
			srr.PoStatus = po.PoStatus
			if prCount == 0 {
				shippingRequestsResults[0] = srr
			} else {
				shippingRequestsResults = append(shippingRequestsResults, srr)
			}
			prCount += 1
		}
	}
	return shippingRequestsResults, nil
}
