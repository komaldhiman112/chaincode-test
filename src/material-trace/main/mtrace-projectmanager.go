package main

import (
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

const (
	MODE_DISTRIBUTOR_ACCEPTS = "distributoraccepts"
	MODE_MANUFACTURER_ACK    = "manufactureracknowledge"
	MODE_ITEM_SHIPPED        = "itemshipped"
	MODE_ITEM_DELIVERED      = "delivered"
)

/*
	Method: fillSharedInfo
	A utility method to fill shared line items data
*/
func fillSharedInfo(lineItem LineItem, poId string) SharedLineDetail {
	sharedInfo := SharedLineDetail{}
	sharedInfo.PoNumber = lineItem.PoNumber
	sharedInfo.ItemKey = lineItem.ItemKey
	sharedInfo.LineNumber = lineItem.LineNumber
	sharedInfo.ShippingRequestNumber = lineItem.ShippingRequestNumber
	sharedInfo.AssignedTo = lineItem.AssignedTo
	sharedInfo.ProgressStatus = make([]ItemStatus, len(lineItem.ProgressStatus))
	sharedInfo.ProgressStatus = lineItem.ProgressStatus
	sharedInfo.IotProperties = make([]IotProperty, len(lineItem.IotProperties))
	sharedInfo.TimeShipped = lineItem.TimeShipped
	sharedInfo.TimeReceived = lineItem.TimeReceived
	return sharedInfo
}

/*
	Method: addNewShareProgressRecord
	Adds a new shared record based on who is acting on a particular purchase order
*/
func addNewShareProgressRecord(stub shim.ChaincodeStubInterface, poId string, sharedDetails []SharedLineDetail) {
	sharedProgress := SharedProgressReport{}
	sharedProgress.PoId = poId
	sharedProgress.ObjectType = PRIVATE_COLLECTION_GENERAL_PROGRESS
	sharedProgress.LineItems = make([]SharedLineDetail, len(sharedDetails))
	sharedProgress.LineItems = sharedDetails
	lineItemBytes, err := json.Marshal(sharedProgress)
	if err != nil {
		logger.Warningf("Unable to Marshal shared progress object for: %s ", poId)
	} else {
		logger.Infof("trying to save in shared progress object")
		err = stub.PutPrivateData(PRIVATE_COLLECTION_GENERAL_PROGRESS, poId, lineItemBytes)
		if err != nil {
			logger.Warningf("Unable to commit shipping status update for in %s ", PRIVATE_COLLECTION_GENERAL_PROGRESS)
		} else {
			logger.Infof("Still here ... and data comitted %s", sharedProgress)
		}
	}
}

/*
	Method: updateSharedProgressRecord
	Updates specific line items based on who is acting on a particular purchase order
*/
func updateSharedProgressRecord(stub shim.ChaincodeStubInterface, poId string, itemsToUpdateMap map[int]LineItem, itemStatus ItemStatus, mode string) {
	privateDataResponse, _ := stub.GetPrivateData(PRIVATE_COLLECTION_GENERAL_PROGRESS, poId)
	if privateDataResponse != nil {
		sharedProgress := SharedProgressReport{}
		json.Unmarshal(privateDataResponse, &sharedProgress)
		for i, eachItem := range sharedProgress.LineItems {
			lineItem, keyFound := itemsToUpdateMap[eachItem.LineNumber]
			if !keyFound {
				logger.Infof("key not found: %d ", lineItem.LineNumber)
				continue
			}
			sharedProgress.LineItems[i].ProgressStatus = append(sharedProgress.LineItems[i].ProgressStatus, itemStatus)
			switch mode {
			case MODE_DISTRIBUTOR_ACCEPTS:
				sharedProgress.LineItems[i].AssignedTo = lineItem.AssignedTo
			case MODE_ITEM_SHIPPED:
				sharedProgress.LineItems[i].MaterialCertificate = lineItem.MaterialCertificate
				sharedProgress.LineItems[i].TimeShipped = lineItem.TimeShipped
				sharedProgress.LineItems[i].IotTrackingCode = lineItem.IotTrackingCode
				sharedProgress.LineItems[i].ShippingRequestNumber = lineItem.ShippingRequestNumber
			case MODE_ITEM_DELIVERED:
				sharedProgress.LineItems[i].TimeReceived = lineItem.TimeReceived
				sharedProgress.LineItems[i].IotProperties = lineItem.IotProperties
			}
		}
		lineItemBytes, err := json.Marshal(sharedProgress)
		if err != nil {
			logger.Warningf("Unable to Marshal shared progress object for: %s ", poId)
			// return shim.Error(err3.Error())
		} else {
			logger.Infof("trying to save in shared progress object")
			err = stub.PutPrivateData(PRIVATE_COLLECTION_GENERAL_PROGRESS, poId, lineItemBytes)
			if err != nil {
				logger.Warningf("Unable to commit shipping status update for in %s ", PRIVATE_COLLECTION_GENERAL_PROGRESS)
			}
		}
	}
}
