package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	sc "github.com/hyperledger/fabric/protos/peer"
)

/*
	Method: logisticsAcceptAndShipsToCustomer
	Executed when logistics operator accepts a shipment request
*/
func (s *SmartContract) logisticsAcceptAndShipsToCustomer(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting one argument. 1. PoId 2. lineItems")
	}
	lineItemsToShip := []LineItem{}
	err := json.Unmarshal([]byte(args[1]), &lineItemsToShip)
	if err != nil {
		return shim.Error("Unable to parse lineItem data provided - " + args[1])
	}
	var lineitemToShipMap = make(map[int]LineItem)
	for _, lineItem := range lineItemsToShip {
		lineitemToShipMap[lineItem.LineNumber] = lineItem
	}
	// progressStatus := ItemStatus{}
	// err = json.Unmarshal([]byte(args[2]), &progressStatus)
	// if err != nil {
	// 	return shim.Error("Unable to parse progress status data provided - " + args[2])
	// }

	poId := args[0]
	shippingPd := ShippingPrivateDetails{}
	shippingPrivateDataResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_LOGISTICS, poId)
	if err1 != nil {
		return shim.Error("Shipping Request not found for - " + args[0])
	}
	event := ItemDeliveryEvent{}
	event.ShippedLineItems = make([]ShippingLineItem, 1)
	if shippingPrivateDataResponse != nil {
		shippingItemCount := 0
		json.Unmarshal(shippingPrivateDataResponse, &shippingPd)
		for i, shipLineItem := range shippingPd.LineItems {
			lineItem := lineitemToShipMap[shipLineItem.LineNumber]
			if shipLineItem.LineNumber != lineItem.LineNumber {
				continue
			}
			if shipLineItem.Status == STATUS_OPEN || shipLineItem.Status == "readyforshipment" {
				shippingPd.LineItems[i].Status = STATUS_IN_TRANSIT
				shippingPd.LineItems[i].TimeShipped = lineItem.TimeShipped
			}
			if shippingItemCount == 0 {
				// shippingPd.LineItems[i].ProgressStatus = append(shippingPd.LineItems[i].ProgressStatus, progressStatus)
				event.ShippedLineItems[0] = shippingPd.LineItems[i]
			} else {
				event.ShippedLineItems = append(event.ShippedLineItems, shippingPd.LineItems[i])
			}
			shippingItemCount += 1
		}
	}
	shippingPd.ObjectType = PRIVATE_COLLECTION_LOGISTICS
	shippingLineItemBytes, err := json.Marshal(shippingPd)
	if err != nil {
		logger.Error("Error json.Marshal(shippingPd): ", err)
		return shim.Error(err.Error())
	}
	err = stub.PutPrivateData(PRIVATE_COLLECTION_LOGISTICS, poId, shippingLineItemBytes)
	if err != nil {
		logger.Error("Unable to update shipping lineitems: ", err)
	}
	// here trigger event notification
	if len(event.ShippedLineItems) > 0 {
		//emit event
		event.Type = "shipmentaccepted"
		event.Status = "accepted"
		event.PoId = poId
		eventBytes, err := json.Marshal(&event)
		if err != nil {
			fmt.Println("unable to marshal event ", err)
		}
		err = stub.SetEvent(event.Type, eventBytes)
		if err != nil {
			fmt.Println("Could not set event for shipment accepted ", err)
		} else {
			fmt.Println("shipment accepted event emitted")
		}
	} else {
		fmt.Println("len(event.ShippedLineItems) is less than 1")
	}
	return shim.Success(shippingLineItemBytes)

}

/*
	Method: fillShippingLineItems
	Utility method to fill line items for addition to logistics privvate collection
*/
func fillShippingLineItems(poId string, shippingRequestNumber int64, lineItem LineItem, shippingRequestedBy string, hasExistingData bool, shippingItemCount int, shippingPd ShippingPrivateDetails, initialStatus []ItemStatus, progressStatus ItemStatus) ShippingPrivateDetails {

	shipLineItem := ShippingLineItem{}
	shipLineItem.PoId = poId
	shipLineItem.PoNumber = lineItem.PoNumber
	shipLineItem.LineNumber = lineItem.LineNumber
	shipLineItem.RequestedBy = shippingRequestedBy
	shipLineItem.ShipToLocation = lineItem.ShipToLocation
	shipLineItem.IotTrackingCode = lineItem.IotTrackingCode // iotTrackingCode
	shipLineItem.MaterialId = lineItem.MaterialId
	shipLineItem.Description = lineItem.Description
	shipLineItem.Status = STATUS_OPEN // "readyforshipment"
	shipLineItem.TimeRequested = lineItem.TimeShipped
	shipLineItem.ShippingRequestNumber = shippingRequestNumber
	shipLineItem.DeliveryDate = lineItem.DeliveryDate
	shipLineItem.Quantity = lineItem.Quantity // needed for sap confirmation call
	shipLineItem.UnitOfMeasure = lineItem.UnitOfMeasure
	shipLineItem.ProgressStatus = make([]ItemStatus, 2)
	shipLineItem.ProgressStatus = initialStatus
	shipLineItem.ProgressStatus = append(shipLineItem.ProgressStatus, progressStatus)
	if shippingItemCount == 0 && !hasExistingData {
		shippingPd.LineItems[shippingItemCount] = shipLineItem
	} else {
		shippingPd.LineItems = append(shippingPd.LineItems, shipLineItem)
	}
	return shippingPd
}

/*
	Method: updateLogisticsDeliveryStatus
	Executed when shippment has reached destination
*/
func updateLogisticsDeliveryStatus(stub shim.ChaincodeStubInterface, poId string, event ItemDeliveryEvent) ItemDeliveryEvent {

	poPrivateDataResponse, err2 := stub.GetPrivateData(PRIVATE_COLLECTION_LOGISTICS, poId)
	if err2 != nil {
		// log error here
		logger.Info("unable to find private data for: " + poId + " in collection: " + PRIVATE_COLLECTION_LOGISTICS + " error: " + err2.Error())
	} else {
		shippingPrivateData := ShippingPrivateDetails{}
		json.Unmarshal(poPrivateDataResponse, &shippingPrivateData)
		updatedCount := 0
		for i, eachItem := range shippingPrivateData.LineItems {
			if eachItem.IotTrackingCode == event.TrackingCode {
				shippingPrivateData.LineItems[i].Status = event.Status
				shippingPrivateData.LineItems[i].ProgressStatus = append(shippingPrivateData.LineItems[i].ProgressStatus, event.ProgressStatus)
				if updatedCount == 0 {
					event.ShippingRequestNumber = shippingPrivateData.LineItems[i].ShippingRequestNumber
				}
				updatedCount += 1
			}
		}
		if updatedCount > 0 {
			sdLineItemBytes, err2 := json.Marshal(shippingPrivateData)
			if err2 != nil {
				logger.Info("unable to marshal private data for: " + poId + " in collection: " + PRIVATE_COLLECTION_LOGISTICS + " error: " + err2.Error())
				//	return shim.Error(err.Error())
			}
			err2 = stub.PutPrivateData(PRIVATE_COLLECTION_LOGISTICS, poId, sdLineItemBytes)
			if err2 != nil {
				// return shim.Error(err.Error())
				logger.Info("unable to commit data for: " + poId + " in collection: " + PRIVATE_COLLECTION_LOGISTICS + " error: " + err2.Error())
			}
		}
	}
	return event

}

/*
	Method: commitShippingPrivateData
	Utility method to commit data into logistics table
*/
func commitShippingPrivateData(stub shim.ChaincodeStubInterface, poId string, shippingPd ShippingPrivateDetails) {
	logger.Info("commitShippingPrivateData: about to record shipment in logistics table for org " + currentMspId)
	shippingLineItemBytes, err1 := json.Marshal(shippingPd)
	if err1 == nil {
		err2 := stub.PutPrivateData(PRIVATE_COLLECTION_LOGISTICS, poId, shippingLineItemBytes)
		if err2 != nil {
			logger.Info("Could not commit shipping lineitems ", err2)
		} else {
			logger.Info("notifyShipmentToCustomer: commited? record shipment in logistics table for org " + currentMspId)
		}
	} else {
		logger.Info("Unable to marshal shipping private data ", err1)
		fmt.Println("Unable to marshal shipping private data ", err1)
	}
}
