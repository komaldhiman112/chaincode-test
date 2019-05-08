package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	sc "github.com/hyperledger/fabric/protos/peer"
)

/*
	Method: manufacturerAcknowledgeOrderRequest
	Executed when manufacturer operator accepts an order request assigned by distributor
*/
func (s *SmartContract) manufacturerAcknowledgeOrderRequest(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 2. 1. poId 2. tiimeStamp 3. progress status 4. msgKey")
	}
	poId := args[0]
	ackTimeStamp, parseErr := strconv.ParseInt(args[1], 10, 64)
	if parseErr != nil {
		return shim.Error("Unable to parse timestamp provided - " + args[1] + " Expecting a number.")
	}
	progressStatus := ItemStatus{}
	parseErr = json.Unmarshal([]byte(args[2]), &progressStatus)
	if parseErr != nil {
		return shim.Error("Unable to parse progress status data provided - " + args[2])
	}
	privateCollection := ""
	owner := ""
	msgKey := args[3]
	switch currentMspId {
	case "org3msp": // "manufacturer 1"
		privateCollection = PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER1
		owner = organizationMap["org3msp"]
	case "org4msp": // "manufacturer 2"
		privateCollection = PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER2
		owner = organizationMap["org4msp"]
	}
	if privateCollection == "" {
		return shim.Error("Unexpected Organization Id - " + currentMspId)
	}
	poPrivateDataResponse, err1 := stub.GetPrivateData(privateCollection, poId)
	if err1 != nil {
		logger.Info("Unable to get " + privateCollection + " data for PO: " + poId)
		return shim.Error("No records found for PO: " + poId)
	}
	itemPrivateData := LineItemCDPrivateDetails{}
	json.Unmarshal(poPrivateDataResponse, &itemPrivateData)
	itemsToUpdate := make([]LineItem, 1)
	sharedItemsMap := make(map[int]LineItem)
	for i, line := range itemPrivateData.LineItems {
		itemPrivateData.LineItems[i].AcknowledgedTimeStamp = progressStatus.TimeStamp
		itemPrivateData.LineItems[i].Status = progressStatus.Status
		itemPrivateData.LineItems[i].ProgressStatus = append(itemPrivateData.LineItems[i].ProgressStatus, progressStatus)
		item := LineItem{ItemKey: itemPrivateData.LineItems[i].ItemKey, LineNumber: line.LineNumber}
		if i == 0 {
			itemsToUpdate[0] = item
		} else {
			itemsToUpdate = append(itemsToUpdate, item)
		}
		sharedItemsMap[item.LineNumber] = item
	}

	// Add progress to shared table
	updateSharedProgressRecord(stub, poId, sharedItemsMap, progressStatus, MODE_MANUFACTURER_ACK)

	pdLineItemBytes, err := json.Marshal(itemPrivateData)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutPrivateData(privateCollection, poId, pdLineItemBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	if msgKey != "" {
		var event = CustomEvent{
			Type:        msgKey,
			Description: "Order Request Acknowledged",
			Status:      STATUS_WIP,
			Id:          itemPrivateData.PoId,
			Custodian:   owner,
			TimeStamp:   ackTimeStamp,
			LineItems:   itemsToUpdate,
		}
		eventBytes, err := json.Marshal(&event)
		if err != nil {
			fmt.Println("unable to marshal event ", err)
		}
		err = stub.SetEvent(event.Type, eventBytes)
		if err != nil {
			fmt.Println("Could not set event for Order Request Acknowledged ", err)
		} else {
			logger.Infof("Event set - type: %s description: %s", event.Type, event.Description)
		}
	}

	return shim.Success(pdLineItemBytes)
}

/*
	Method: updateManufacturerIncomingIOT
	Executed when new IOT data is received and updates specific line items assigned to the manufacturer
*/
func updateManufacturerIncomingIOT(stub shim.ChaincodeStubInterface, privateCollection string, poId string, iotInput IotProperty, itemStatus ItemStatus, msgKey string) (ok bool) {

	poPrivateDataResponse, err1 := stub.GetPrivateData(privateCollection, poId)
	if err1 != nil {
		// log poId not found
		logger.Info("unable to find data for: " + poId + " in collection: " + privateCollection)
		return false
	}
	itemPrivateData := LineItemCDPrivateDetails{}
	json.Unmarshal(poPrivateDataResponse, &itemPrivateData)
	emit_on_delivery := false
	itemMap := make(map[string][]IotProperty)
	sharedItemsMap := make(map[int]LineItem)
	for i, eachItem := range itemPrivateData.LineItems {
		if eachItem.IotTrackingCode != iotInput.TrackingCode {
			continue
		}
		if eachItem.Status == STATUS_DELIVERED {
			emit_on_delivery = true
			// timeShipped = itemPrivateData.LineItems[i].TimeShipped
			break
		}
		iotProperties := itemPrivateData.LineItems[i].IotProperties
		if iotProperties == nil {
			iotProperties = make([]IotProperty, 1)
			iotProperties[0] = iotInput
		} else {
			iotProperties = append(iotProperties, iotInput)
		}
		itemPrivateData.LineItems[i].IotProperties = iotProperties
		itemMap[eachItem.ItemKey] = iotProperties
		mi := distanceFromProjectSite(eachItem.ShipToLocation, iotInput)
		if mi < 1 {
			itemPrivateData.LineItems[i].Status = STATUS_DELIVERED
			itemPrivateData.LineItems[i].ProgressStatus = append(itemPrivateData.LineItems[i].ProgressStatus, itemStatus)
			emit_on_delivery = true
			// timeShipped = itemPrivateData.LineItems[i].TimeShipped
		}
		// add to shared record
		line := LineItem{}
		line.IotProperties = iotProperties
		line.LineNumber = eachItem.LineNumber
		if emit_on_delivery {
			line.TimeReceived = itemStatus.TimeStamp
		}
		sharedItemsMap[eachItem.LineNumber] = line
	}

	pdLineItemBytes, err := json.Marshal(itemPrivateData)
	if err != nil {
		logger.Info("unable to marshal private data for: " + poId + " in collection: " + privateCollection + " error: " + err.Error())
		return false
		//	return shim.Error(err.Error())
	}
	err = stub.PutPrivateData(privateCollection, poId, pdLineItemBytes)
	if err != nil {
		// return shim.Error(err.Error())
		logger.Info("unable to commit data for: " + poId + " in collection: " + privateCollection + " error: " + err.Error())
		return false
	}
	if emit_on_delivery {

		// Add progress to shared table
		updateSharedProgressRecord(stub, poId, sharedItemsMap, itemStatus, MODE_ITEM_DELIVERED)

		var event = ItemDeliveryEvent{Type: msgKey, Status: STATUS_DELIVERED, SkipDistributor: false, PoId: poId, TrackingCode: iotInput.TrackingCode, ItemMap: itemMap, ProgressStatus: itemStatus}
		event = updateLogisticsDeliveryStatus(stub, poId, event)
		eventBytes, err := json.Marshal(&event)
		if err != nil {
			fmt.Println("unable to marshal event ", err)
		}
		err = stub.SetEvent(event.Type, eventBytes)
		if err != nil {
			fmt.Println("Could not set event for items delivered ", err)
		} else {
			logger.Infof("Items delivered event emitted - %s itemMap: %s", event, event.ItemMap)
		}
	}
	return true

}
