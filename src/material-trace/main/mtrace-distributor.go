package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	str "strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	sc "github.com/hyperledger/fabric/protos/peer"
)

/*
	method: acceptPo
	Executed when a distributor accepts or rejects a purchase order.
	If po is rejected, the status is set to rejected and process stops.
	Otherwise the lineitems are split based on assignedTo value - items can go to either inventory from distributor,
	manufacturerer 1, or manufacturer 2.
	The split items are stored in private collection databases.
*/
func (s *SmartContract) acceptPo(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 6 {
		return shim.Error("Incorrect number of arguments. Expecting 6. 1. updated lineItems, 2. po acceptance flag 3. timestamp of acceptance 4. Rejection reason 5. manufacturer discounts 6. progressStatus")
	}
	isAccepted, parseErr := strconv.ParseBool(args[1])
	if parseErr != nil {
		return shim.Error("Expecting true or false value for second argument. Found: " + args[1])
	}
	if !isAccepted && len(args[3]) == 0 {
		return shim.Error("Rejection reason is required.")
	}
	acceptanceTimeStamp, parseErr := strconv.ParseInt(args[2], 10, 64)
	if parseErr != nil {
		return shim.Error("Unable to parse timestamp provided - " + args[2] + " Expecting a number.")
	}
	mfrDiscounts := []ManufacturerPricingDiscount{}
	json.Unmarshal([]byte(args[4]), &mfrDiscounts)
	fmt.Println("Discounts length " + strconv.Itoa(len(mfrDiscounts)))
	if len(mfrDiscounts) < 2 {
		return shim.Error("At least 2 manufacturer discount objects expected.")
	}
	progressStatus := ItemStatus{}
	err := json.Unmarshal([]byte(args[5]), &progressStatus)
	if err != nil {
		return shim.Error("Unable to parse progress status data provided - " + args[5])
	}
	discountsMap := make(map[string]ManufacturerPricingDiscount)
	for _, discount := range mfrDiscounts {
		discountsMap[str.ToLower(discount.Name)] = discount
		fmt.Println(" Discount for  " + discountsMap[str.ToLower(discount.Name)].Name + " is " + strconv.Itoa(discountsMap[str.ToLower(discount.Name)].Discount))
	}

	item := PurchaseOrder{}
	json.Unmarshal([]byte(args[0]), &item) // converts the incoming json object into PurchaseOrder Struct
	poId := item.PoId
	// Map of updated items from the UI
	lineItemMap := make(map[int]LineItem)
	for _, lineItem := range item.LineItems {
		// lineItemMap[lineItem.MaterialId] = lineItem
		lineItemMap[lineItem.LineNumber] = lineItem
	}
	// Find existing PO from blockchain
	value, err := stub.GetState(poId)
	if err != nil || value == nil {
		return shim.Error("Not Found")
	}
	po := PurchaseOrder{}
	json.Unmarshal(value, &po)
	if !isAccepted {
		po.PoStatus = STATUS_REJECTED
		po.Comment = args[3]
		if err := stub.PutState(po.PoId, po.ToJson()); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(po.ToJson())
	}
	po.PoStatus = STATUS_ACCEPTED
	po.AcceptanceTimeStamp = acceptanceTimeStamp
	utilityInitialStatus := ItemStatus{
		Owner:     organizationMap["org1msp"],
		Status:    STATUS_OPEN,
		TimeStamp: po.CreatedTimeStamp,
	}
	distributorProgressStatus := progressStatus
	cdMfr1LineItem := LineItemCDPrivateDetails{}
	cdMfr2LineItem := LineItemCDPrivateDetails{}
	cdDistributorLineItem := LineItemCDPrivateDetails{}

	mfrAssignedCount := 0
	mfr2AssignedCount := 0
	distributorAssignedCount := 0
	currentPrivateLineItems := LineItemPrivateDetails{}
	poPrivateDataResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, po.PoId)
	if err1 != nil {
		logger.Info("Unable to get PRIVATE_COLLECTION_CUSTOMER_LINEITEMS data for PO: " + po.PoId)
	} else {
		if poPrivateDataResponse != nil {
			json.Unmarshal(poPrivateDataResponse, &currentPrivateLineItems)
			po.LineItems = currentPrivateLineItems.LineItems
		} else {
			fmt.Println("private lineItems data not found for " + po.PoId)
			return shim.Error("private lineItems data not found for " + po.PoId)
		}

	}
	validMfrs := "manufacturer 1|manufacturer 2"
	sharedItemsMap := make(map[int]LineItem)
	for i, lineItem := range po.LineItems {
		updatedItem := lineItemMap[lineItem.LineNumber] // lineItem.MaterialId]
		lineItem.DeliveryDate = updatedItem.DeliveryDate
		lineItem.AssignedQty = updatedItem.AssignedQty
		orderRequests := make([]OrderRequest, 1)
		orderSplitCount := 0
		distributorQty := 0
		assignedToMfr := ""
		// items that will be fulfilled by distributor
		if updatedItem.AssignedTo == "" || str.ToLower(updatedItem.AssignedTo) == "inventory" {
			// distributorQty := updatedItem.Quantity - updatedItem.AssignedQty
			distributorQty = updatedItem.Quantity
			updatedItem.AssignedTo = "Inventory"
		} else {
			updatedItem.AssignedQty = updatedItem.Quantity
			// updatedItem.AssignToMfr = updatedItem.AssignedTo
			assignedToMfr = updatedItem.AssignedTo
		}
		sharedItemsMap[lineItem.LineNumber] = updatedItem
		if distributorQty > 0 {
			pricingInfo := fillPricingInfo(lineItem, updatedItem.DeliveryDate, updatedItem.AssignedTo, distributorQty, updatedItem.UnitPrice, po.PoNumber, po.PoId, utilityInitialStatus, distributorProgressStatus)
			if distributorAssignedCount == 0 {
				cdDistributorLineItem.LineItems = make([]LineItemPricing, 1)
				cdDistributorLineItem.ObjectType = PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR
				cdDistributorLineItem.PoId = item.PoId
				cdDistributorLineItem.LineItems[0] = pricingInfo
			} else {
				cdDistributorLineItem.LineItems = append(cdDistributorLineItem.LineItems, pricingInfo)
			}

			orderRequest := OrderRequest{}
			orderRequest.LineNumber = lineItem.LineNumber
			orderRequest.MaterialId = lineItem.MaterialId
			orderRequest.Quantity = distributorQty
			orderRequest.Status = STATUS_WIP
			orderRequest.FulfilledBy = organizationMap["org2msp"] //  "Distributor"
			orderRequest.AcknowledgedTimeStamp = progressStatus.TimeStamp
			orderRequest.ProgressStatus = pricingInfo.ProgressStatus
			if orderSplitCount == 0 {
				orderRequests[0] = orderRequest
			} else {
				orderRequests = append(orderRequests, orderRequest)
			}
			orderSplitCount += 1
			distributorAssignedCount += 1
		}
		// items that will be fullfilled by manufacturer
		if updatedItem.AssignedQty > 0 && str.Contains(str.ToLower(validMfrs), str.ToLower(assignedToMfr)) {
			mapKey := str.ToLower(assignedToMfr)
			discountInfo := discountsMap[mapKey]
			// dbug := fmt.Sprintf("got here the key is: %s Name is: %s Discount is: %d", mapKey, discountInfo.Name, discountInfo.Discount)
			// logger.Info(dbug)
			if discountInfo.Name == "" || discountInfo.Discount == 0 {
				return shim.Error("Manufacturer discount missing for " + assignedToMfr)
			}
			discountedPrice := (lineItem.UnitPrice) - (float64(discountInfo.Discount) / 100 * lineItem.UnitPrice)
			updatedItem.MfrUnitPrice = math.Round(discountedPrice)
			pricingInfo := fillPricingInfo(lineItem, updatedItem.DeliveryDate, updatedItem.AssignedTo, updatedItem.AssignedQty, updatedItem.MfrUnitPrice, po.PoNumber, po.PoId, utilityInitialStatus, distributorProgressStatus)
			// who is supplying specific items
			orderRequest := OrderRequest{}
			orderRequest.LineNumber = lineItem.LineNumber
			orderRequest.MaterialId = lineItem.MaterialId
			orderRequest.Quantity = updatedItem.AssignedQty
			orderRequest.Status = STATUS_OPEN        // ORDER_STATUS_SUPPLIER_FULFILLMENT
			orderRequest.FulfilledBy = assignedToMfr // updatedItem.AssignToMfr
			orderRequest.AcknowledgedTimeStamp = progressStatus.TimeStamp
			if orderSplitCount == 0 {
				orderRequests[0] = orderRequest
			} else {
				orderRequests = append(orderRequests, orderRequest)
			}
			orderSplitCount += 1
			if str.ToLower(assignedToMfr) == str.ToLower(organizationMap["org3msp"]) { // "manufacturer 1" {
				if mfrAssignedCount == 0 {
					cdMfr1LineItem.LineItems = make([]LineItemPricing, 1)
					cdMfr1LineItem.ObjectType = PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER1
					cdMfr1LineItem.PoId = item.PoId
					cdMfr1LineItem.LineItems[0] = pricingInfo
				} else {
					cdMfr1LineItem.LineItems = append(cdMfr1LineItem.LineItems, pricingInfo)
				}
				mfrAssignedCount += 1
			} else if str.ToLower(assignedToMfr) == str.ToLower(organizationMap["org4msp"]) { // "manufacturer 2" {
				if mfr2AssignedCount == 0 {
					cdMfr2LineItem.LineItems = make([]LineItemPricing, 1)
					cdMfr2LineItem.ObjectType = PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER2
					cdMfr2LineItem.PoId = item.PoId
					cdMfr2LineItem.LineItems[0] = pricingInfo
				} else {
					cdMfr2LineItem.LineItems = append(cdMfr2LineItem.LineItems, pricingInfo)
				}
				mfr2AssignedCount += 1
			} else {
				fmt.Println("why did i get here ??  org3= " + str.ToLower(organizationMap["org3msp"]) + " org4= " + str.ToLower(organizationMap["org4msp"]))
			}
		}
		lineItem.OrderRequests = orderRequests
		lineItem.AcknowledgedTimeStamp = progressStatus.TimeStamp
		lineItem.ProgressStatus = append(lineItem.ProgressStatus, progressStatus)
		if len(lineItem.OrderRequests) == 1 {
			lineItem.AssignedTo = orderRequests[0].FulfilledBy
			lineItem.Status = orderRequests[0].Status
			lineItem.AssignedQty = orderRequests[0].Quantity
		}
		po.LineItems[i] = lineItem
	}
	if distributorAssignedCount > 0 {
		cdLineItemBytes, err := json.Marshal(cdDistributorLineItem)
		if err != nil {
			return shim.Error(err.Error())
		}
		err = stub.PutPrivateData(PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR, item.PoId, cdLineItemBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
	}
	// update distributor specific line items per original request from customer
	if len(po.LineItems) > 0 {
		currentPrivateLineItems.ObjectType = PRIVATE_COLLECTION_CUSTOMER_LINEITEMS
		currentPrivateLineItems.LineItems = po.LineItems
		itemBytes, err := json.Marshal(currentPrivateLineItems)
		if err != nil {
			fmt.Println(" Error mashalling poLineItems")
		} else {
			err = stub.PutPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, item.PoId, itemBytes)
			if err != nil {
				fmt.Println("unable to write private lineitems data for " + po.PoId)
			}
		}
	} else {
		fmt.Println("PRIVATE_COLLECTION_CUSTOMER_LINEITEMS private data not found for " + po.PoId)
	}
	// if purchase order has been accepted and
	// some items have been assigned to manufacturer add to private
	// collection between distributor and manufacturer
	if mfrAssignedCount > 0 {
		cdLineItemBytes, err := json.Marshal(cdMfr1LineItem)
		if err != nil {
			return shim.Error(err.Error())
		}
		err = stub.PutPrivateData(PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER1, item.PoId, cdLineItemBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
	}
	if mfr2AssignedCount > 0 {
		cdLineItemBytes, err := json.Marshal(cdMfr2LineItem)
		if err != nil {
			return shim.Error(err.Error())
		}
		err = stub.PutPrivateData(PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER2, item.PoId, cdLineItemBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
	}
	// Add progress to shared table
	updateSharedProgressRecord(stub, poId, sharedItemsMap, progressStatus, MODE_DISTRIBUTOR_ACCEPTS)
	// submit changes to purchase order
	poLineItems := po.LineItems
	po.LineItems = make([]LineItem, 1) // remove lineItems from primary db, line items will be stored in priviate collections.
	if err := stub.PutState(po.PoId, po.ToJson()); err != nil {
		return shim.Error(err.Error())
	}

	po.LineItems = poLineItems // return for client consumption
	var event = CustomEvent{Type: "poaccepted", Description: STATUS_ACCEPTED, Status: po.PoStatus, Id: po.PoId, PoNumber: po.PoNumber, LineItems: po.LineItems}
	eventBytes, err := json.Marshal(&event)
	if err != nil {
		fmt.Println("unable to marshal event ", err)
	}
	err = stub.SetEvent("poaccepted", eventBytes)
	if err != nil {
		fmt.Println("Could not set event for Po acceptance ", err)
	} else {
		fmt.Println("Event set - " + event.Description)
	}

	return shim.Success(po.ToJson())
}

/*
	Method: fillPricingInfo
	This is a utility method to fill private collection lineitems
*/
func fillPricingInfo(lineItem LineItem, DeliveryDate string, assignedTo string, quantity int, unitCost float64, poNumber int, poId string, utilityInitialStatus ItemStatus, distributorInitialStatus ItemStatus) LineItemPricing {
	pricingInfo := LineItemPricing{}
	pricingInfo.PoId = poId
	pricingInfo.PoNumber = poNumber
	pricingInfo.MaterialId = lineItem.MaterialId
	pricingInfo.ItemKey = lineItem.ItemKey
	pricingInfo.Currency = lineItem.Currency
	pricingInfo.MaterialGroup = lineItem.MaterialGroup
	pricingInfo.ShipToLocation = lineItem.ShipToLocation
	pricingInfo.LineNumber = lineItem.LineNumber
	pricingInfo.Description = lineItem.Description
	pricingInfo.DeliveryDate = DeliveryDate
	pricingInfo.ProjectId = lineItem.ProjectId
	pricingInfo.UnitOfMeasure = lineItem.UnitOfMeasure
	pricingInfo.Quantity = quantity
	pricingInfo.UnitPrice = unitCost
	pricingInfo.Subtotal = math.Round(float64(quantity) * unitCost)
	pricingInfo.AssignedTo = assignedTo
	pricingInfo.ProgressStatus = make([]ItemStatus, 1)
	pricingInfo.AcknowledgedTimeStamp = distributorInitialStatus.TimeStamp

	if str.ToLower(assignedTo) == "inventory" {
		wipStatus := distributorInitialStatus
		wipStatus.Status = STATUS_WIP
		pricingInfo.Status = STATUS_WIP
		pricingInfo.ProgressStatus[0] = utilityInitialStatus // distributorInitialStatus
		pricingInfo.ProgressStatus = append(pricingInfo.ProgressStatus, wipStatus)
	} else {
		pricingInfo.Status = STATUS_OPEN
		pricingInfo.ProgressStatus[0] = distributorInitialStatus
	}

	return pricingInfo

}

/*
	Method: advanceInTransitItem
	Executed to advance items that are struck in "in-transit mode"
*/
func (s *SmartContract) advanceInTransitItem(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	itemsToUpdate := make(map[string][]LineItem, 1)
	json.Unmarshal([]byte(args[0]), &itemsToUpdate)
	progressStatus := ItemStatus{}
	err := json.Unmarshal([]byte(args[1]), &progressStatus)
	if err != nil {
		return shim.Error("Unable to parse progress status data provided - " + args[1])
	}
	lineItemMap := make(map[string]LineItem)
	for key, lineItems := range itemsToUpdate {
		for _, line := range lineItems {
			logger.Infof("line.ItemKey: %s item %d", line.ItemKey, line.LineNumber)
			lineItemMap[line.ItemKey] = line
		}
		collectionResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, key)
		if err1 != nil {
			logger.Info("Unable to get " + PRIVATE_COLLECTION_CUSTOMER_LINEITEMS + " data for PO: " + key)
			continue
		}
		itemPrivateData := LineItemPrivateDetails{}
		json.Unmarshal(collectionResponse, &itemPrivateData)
		updatedCount := 0
		for i, _ := range itemPrivateData.LineItems {
			item, keyFound := lineItemMap[itemPrivateData.LineItems[i].ItemKey]
			logger.Infof("key: %s item %s keyFound %b ", key, item.ItemKey, keyFound)
			if keyFound && item.ItemKey == itemPrivateData.LineItems[i].ItemKey {
				if str.ToLower(itemPrivateData.LineItems[i].Status) == str.ToLower(STATUS_SHIPPED) {
					itemPrivateData.LineItems[i].ShippingRequestNumber = item.ShippingRequestNumber
					itemPrivateData.LineItems[i].Status = STATUS_RECEIVED
					itemPrivateData.LineItems[i].ProgressStatus = append(itemPrivateData.LineItems[i].ProgressStatus, progressStatus)
					updatedCount += 1
				} else {
					logger.Infof("status doesn't match; expected %s but found %s ", STATUS_SHIPPED, itemPrivateData.LineItems[i].Status)
				}
				markAsDeliveredLogistics(stub, key, item.ItemKey, progressStatus)
			} else {
				logger.Infof("key not found")
			}
		}
		if updatedCount > 0 {
			pdLineItemBytes, err := json.Marshal(itemPrivateData)
			if err != nil {
				logger.Info("Unable to marshal private data data for PO: " + key)
			}
			err = stub.PutPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, key, pdLineItemBytes)
			if err != nil {
				logger.Info("Unable to update for item " + key)
			}
		} else {
			logger.Info("no update")
		}
	}
	return Success(http.StatusOK, "OK", nil)
}

/*
	Method: markAsDelivered
	utility method to mark item as delivered
*/
func markAsDelivered(stub shim.ChaincodeStubInterface, privateCollectionName string, poId string, itemKey string, progressStatus ItemStatus) {

	privateDataResponse, err1 := stub.GetPrivateData(privateCollectionName, poId)
	if err1 != nil {
		return
	}
	pricingInfo := LineItemCDPrivateDetails{}
	json.Unmarshal(privateDataResponse, &pricingInfo)
	updatedCount := 0
	for i, item := range pricingInfo.LineItems {
		if pricingInfo.LineItems[i].Status == STATUS_SHIPPED && item.ItemKey == itemKey {
			pricingInfo.LineItems[i].Status = progressStatus.Status
			updatedCount += 1
		}
	}
	if updatedCount < 1 {
		return
	}
	pricingBytes, err := json.Marshal(pricingInfo)
	if err != nil {
		logger.Error(err.Error())
	}
	err = stub.PutPrivateData(privateCollectionName, poId, pricingBytes)
	if err != nil {
		logger.Error(err.Error())
	}
}

/*
	Method: markAsDeliveredLogistics
	utility method to mark item as in logistics table
*/
func markAsDeliveredLogistics(stub shim.ChaincodeStubInterface, poId string, itemKey string, progressStatus ItemStatus) {

	privateDataResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_LOGISTICS, poId)
	if err1 != nil {
		return
	}
	pricingInfo := LineItemCDPrivateDetails{}
	json.Unmarshal(privateDataResponse, &pricingInfo)
	updatedCount := 0
	for i, item := range pricingInfo.LineItems {
		if pricingInfo.LineItems[i].Status == STATUS_SHIPPED && item.ItemKey == itemKey {
			pricingInfo.LineItems[i].Status = progressStatus.Status
			updatedCount += 1
		}
	}
	if updatedCount < 1 {
		return
	}
	pricingBytes, err := json.Marshal(pricingInfo)
	if err != nil {
		logger.Error(err.Error())
	}
	err = stub.PutPrivateData(PRIVATE_COLLECTION_LOGISTICS, poId, pricingBytes)
	if err != nil {
		logger.Error(err.Error())
	}
}

/*
	Method: updateProgressStatusOnMfrAcknowledgement
	Executed based on event triggered when a manufacturer acknowledges an order request.
	This allows the distributor to update the main private collection shared between customer and distributor
*/
func (s *SmartContract) updateProgressStatusOnMfrAcknowledgement(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 1 {
		return shim.Error("Expecting two arguments 1. event")
	}
	event := CustomEvent{}
	collectionName := PRIVATE_COLLECTION_CUSTOMER_LINEITEMS
	json.Unmarshal([]byte(args[0]), &event)
	logger.Infof("called updateProgressStatusOnMfrAcknowledgement. event %s", event)
	poPrivateDataResponse, err2 := stub.GetPrivateData(collectionName, event.Id)
	if err2 != nil {
		logger.Info("unable to find private data for: " + event.Id + " in collection: " + collectionName + " error: " + err2.Error())
	} else {
		if len(event.LineItems) == 0 {
			eventBytes, _ := json.Marshal(event)
			return shim.Success(eventBytes)
		}
		var itemsToUpdateMap = make(map[string]LineItem)
		for _, lineItem := range event.LineItems {
			itemsToUpdateMap[lineItem.ItemKey] = lineItem
		}
		pItem := LineItemPrivateDetails{}
		json.Unmarshal(poPrivateDataResponse, &pItem)
		updatedCount := 0
		itemStatus := ItemStatus{
			Owner:     event.Custodian,
			Status:    event.Status,
			TimeStamp: event.TimeStamp,
		}
		for i, eachItem := range pItem.LineItems {
			toUpdate := itemsToUpdateMap[eachItem.ItemKey]
			if toUpdate.ItemKey == eachItem.ItemKey {
				pItem.LineItems[i].ProgressStatus = append(pItem.LineItems[i].ProgressStatus, itemStatus)
				updatedCount += 1
			}
		}
		if updatedCount > 0 {
			itemBytes, err2 := json.Marshal(pItem)
			if err2 != nil {
				logger.Info("unable to marshal private data for: " + event.Id + " in collection: " + collectionName + " error: " + err2.Error())
			}
			err2 = stub.PutPrivateData(collectionName, event.Id, itemBytes)
			if err2 != nil {
				logger.Info("unable to commit data for: " + event.Id + " in collection: " + collectionName + " error: " + err2.Error())
			}
		}
	}
	eventBytes, _ := json.Marshal(event)
	return shim.Success(eventBytes)
}

/*
	Method: notifyDistributorOnMfrShipment
	Executed based on event triggered when a manufacturer ships an item.
	This allows the distributor to update the main private collection shared between customer and distributor
*/
func (s *SmartContract) notifyDistributorOnMfrShipment(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting two arguments. 1. PoId 2. lineItem 3. progress status")
	}
	lineItemsToShip := []LineItem{}
	err := json.Unmarshal([]byte(args[1]), &lineItemsToShip)
	if err != nil {
		return shim.Error("Unable to parse lineItem data provided - " + args[1])
	}
	progressStatus := ItemStatus{}
	err = json.Unmarshal([]byte(args[2]), &progressStatus)
	if err != nil {
		return shim.Error("Unable to parse progress status data provided - " + args[2])
	}
	var lineitemToShipMap = make(map[int]LineItem)
	for _, lineItem := range lineItemsToShip {
		lineitemToShipMap[lineItem.LineNumber] = lineItem
	}
	poId := args[0]
	poPrivateDataResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, poId)
	if err1 != nil {
		logger.Info("notifyDistributorOnMfrShipment: Error, Unable to get " + PRIVATE_COLLECTION_CUSTOMER_LINEITEMS + " data for PO: " + poId)
		return shim.Error("Not Found")
	}
	if poPrivateDataResponse == nil {
		logger.Info("notifyDistributorOnMfrShipment: Unable to find records in collection " + PRIVATE_COLLECTION_CUSTOMER_LINEITEMS + " data for PO: " + poId)
		return shim.Error("Not Found")
	}
	itemPrivateData := LineItemPrivateDetails{}
	json.Unmarshal(poPrivateDataResponse, &itemPrivateData)
	for i, eachItem := range itemPrivateData.LineItems {
		lineItem := lineitemToShipMap[eachItem.LineNumber]
		if eachItem.LineNumber != lineItem.LineNumber {
			logger.Infof("lineItem assigned tp %s lineNumber: %d not in the map, continue ", lineItem.AssignedTo, lineItem.LineNumber)
			continue
		}
		if itemPrivateData.LineItems[i].PoNumber == 0 {
			itemPrivateData.LineItems[i].PoNumber = lineItem.PoNumber
		}
		itemPrivateData.LineItems[i].Status = progressStatus.Status // STATUS_SHIPPED
		itemPrivateData.LineItems[i].IotTrackingCode = lineItem.IotTrackingCode
		itemPrivateData.LineItems[i].MaterialCertificate = lineItem.MaterialCertificate
		itemPrivateData.LineItems[i].TimeShipped = lineItem.TimeShipped
		lastStatusEntry := itemPrivateData.LineItems[i].ProgressStatus[len(itemPrivateData.LineItems[i].ProgressStatus)-1]
		if lastStatusEntry.Status != progressStatus.Status {
			logger.Infof("got here ... line item is assigned to %s attempting to set status to %s", lineItem.AssignedTo, lastStatusEntry.Status)
			itemPrivateData.LineItems[i].ProgressStatus = append(itemPrivateData.LineItems[i].ProgressStatus, progressStatus)
		} else {
			logger.Infof("attempted to assign duplicate status value. item is assigned to %s  attempted status %s", lineItem.AssignedTo, lastStatusEntry.Status)
		}
		for j, orderItem := range eachItem.OrderRequests {
			if orderItem.LineNumber != lineItem.LineNumber {
				continue
			}
			eachItem.OrderRequests[j].Status = STATUS_SHIPPED
		}
		if len(eachItem.OrderRequests) > 0 {
			itemPrivateData.LineItems[i].Status = eachItem.OrderRequests[0].Status
		}
	}
	pdLineItemBytes, err := json.Marshal(itemPrivateData)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, poId, pdLineItemBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(pdLineItemBytes)

}

/*
	Method: notifyDistributorOnLogisticsShipment
	Executed based on event triggered when logistics provider accepts a shipment request.
	This allows the distributor to update the main private collection shared between customer and distributor
*/
func (s *SmartContract) notifyDistributorOnLogisticsShipment(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4 arguments. 1. PoId 2. lineItem 3. timeShipped 4. progress status")
	}
	lineItemsToShip := []LineItem{}
	err := json.Unmarshal([]byte(args[1]), &lineItemsToShip)
	if err != nil {
		return shim.Error("Unable to parse lineItem data provided - " + args[1])
	}
	var lineitemToShipMap = make(map[string]LineItem)
	for _, lineItem := range lineItemsToShip {
		lineitemToShipMap[lineItem.ItemKey] = lineItem
	}
	poId := args[0]
	progressStatus := ItemStatus{}
	err = json.Unmarshal([]byte(args[3]), &progressStatus)
	if err != nil {
		return shim.Error("Unable to parse progress status data provided - " + args[3])
	}
	poPrivateDataResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, poId)
	if err1 != nil {
		logger.Info("notifyDistributorOnMfrShipment: Error, Unable to get " + PRIVATE_COLLECTION_CUSTOMER_LINEITEMS + " data for PO: " + poId)
		return shim.Error("Not Found")
	}
	if poPrivateDataResponse == nil {
		logger.Info("notifyDistributorOnMfrShipment: Unable to find records in collection " + PRIVATE_COLLECTION_CUSTOMER_LINEITEMS + " data for PO: " + poId)
		return shim.Error("Not Found")
	}
	itemPrivateData := LineItemPrivateDetails{}
	json.Unmarshal(poPrivateDataResponse, &itemPrivateData)
	for i, eachItem := range itemPrivateData.LineItems {
		lineItem := lineitemToShipMap[eachItem.ItemKey] // eachItem.LineNumber
		if eachItem.LineNumber != lineItem.LineNumber {
			continue
		}
		itemPrivateData.LineItems[i].IotTrackingCode = lineItem.IotTrackingCode
		for j, orderItem := range eachItem.OrderRequests {
			if orderItem.LineNumber != lineItem.LineNumber {
				continue
			}
			if eachItem.OrderRequests[j].Status != STATUS_SHIPPED {
				eachItem.OrderRequests[j].Status = STATUS_SHIPPED
			}
			eachItem.OrderRequests[j].IotTrackingCode = lineItem.IotTrackingCode
		}
	}
	pdLineItemBytes, err := json.Marshal(itemPrivateData)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, poId, pdLineItemBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	// // update distributor fulfillment table
	// updateDistributorFulfilledLineItems(stub, poId, STATUS_SHIPPED, lineitemToShipMap, progressStatus)

	return shim.Success(pdLineItemBytes)

}

/*
	Method: notifyItemDelivered
	Executed based on event triggered when geolocattion calculations indicate the item has reached destination.
	This allows the distributor to update the main private collection shared between customer and distributor
*/
func (s *SmartContract) notifyItemDelivered(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	logger.Info("called notifyItemDelivered")
	if len(args) != 3 {
		return shim.Error("Expecting two arguments 1. delivery event 2. delivery timestamp 3. progressStatus")
	}
	event := ItemDeliveryEvent{}
	collectionName := PRIVATE_COLLECTION_CUSTOMER_LINEITEMS
	json.Unmarshal([]byte(args[0]), &event)
	timeReceived, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return shim.Error("Invalid number format, expecting a numbeer on argument 2")
	}
	logger.Infof("item received timestamp: %d ", timeReceived)
	progressStatus := ItemStatus{}
	err = json.Unmarshal([]byte(args[2]), &progressStatus)
	if err != nil {
		return shim.Error("Unable to parse progress status data provided - " + args[2])
	}
	poPrivateDataResponse, err2 := stub.GetPrivateData(collectionName, event.PoId)
	if err2 != nil {
		logger.Info("unable to find private data for: " + event.PoId + " in collection: " + collectionName + " error: " + err2.Error())

	} else {
		pItem := LineItemPrivateDetails{}
		json.Unmarshal(poPrivateDataResponse, &pItem)
		updatedCount := 0
		logger.Infof("Event Details: %s ", event)
		for i, eachItem := range pItem.LineItems {
			if len(eachItem.OrderRequests) == 0 {
				continue
			}
			// no splits hence each lineitem has one orderRequest
			orderRequest := eachItem.OrderRequests[0]
			if orderRequest.IotTrackingCode != event.TrackingCode {
				continue
			}
			pItem.LineItems[i].Status = STATUS_RECEIVED
			pItem.LineItems[i].ProgressStatus = append(pItem.LineItems[i].ProgressStatus, progressStatus)
			pItem.LineItems[i].IotProperties = event.ItemMap[eachItem.ItemKey]
			pItem.LineItems[i].TimeReceived = progressStatus.TimeStamp // timeReceived
			pItem.LineItems[i].ShippingRequestNumber = event.ShippingRequestNumber
			eachItem.OrderRequests[0].Status = event.Status
			updatedCount += 1
		}
		if updatedCount > 0 {
			itemBytes, err2 := json.Marshal(pItem)
			if err2 != nil {
				logger.Info("unable to marshal private data for: " + event.PoId + " in collection: " + collectionName + " error: " + err2.Error())
			}
			err2 = stub.PutPrivateData(collectionName, event.PoId, itemBytes)
			if err2 != nil {
				logger.Info("unable to commit data for: " + event.PoId + " in collection: " + collectionName + " error: " + err2.Error())
			}
		}
	}
	eventBytes, _ := json.Marshal(event)
	return shim.Success(eventBytes)

}

/*
	Method: updateDistributorFulfilledLineItems
	Updates the shipping status for items that distribotor is shipping from existing inventory
*/
func updateDistributorFulfilledLineItems(stub shim.ChaincodeStubInterface, poId string, status string, lineitemToShipMap map[string]LineItem, progressStatus ItemStatus) {
	distributorPricingResponse, _ := stub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR, poId)
	if distributorPricingResponse == nil {
		logger.Info("distributorPricingResponse is nil")
		return
	}
	distributorPricing := LineItemCDPrivateDetails{}
	json.Unmarshal(distributorPricingResponse, &distributorPricing)
	for i, eachItem := range distributorPricing.LineItems {
		lineItem, keyFound := lineitemToShipMap[eachItem.ItemKey]
		if !keyFound {
			continue
		}
		if eachItem.LineNumber != lineItem.LineNumber {
			continue
		}
		logger.Infof("in updateDistributorFulfilledLineItems updating lineItems for po: %s  status %s", poId, status)
		distributorPricing.LineItems[i].Status = status
		if status == STATUS_SHIPPED {
			distributorPricing.LineItems[i].Status = STATUS_SHIPPED
			lastStatusEntry := distributorPricing.LineItems[i].ProgressStatus[len(distributorPricing.LineItems[i].ProgressStatus)-1]
			if lastStatusEntry.Status != progressStatus.Status {
				distributorPricing.LineItems[i].ProgressStatus = append(distributorPricing.LineItems[i].ProgressStatus, progressStatus)
			}
			distributorPricing.LineItems[i].TimeShipped = lineItem.TimeShipped
			distributorPricing.LineItems[i].MaterialCertificate = lineItem.MaterialCertificate
		}

	}
	lineItemBytes, err3 := json.Marshal(distributorPricing)
	if err3 != nil {
		logger.Warningf("Unable to Marshal distributor pricing object for: %s ", poId)
	} else {
		err3 = stub.PutPrivateData(PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR, poId, lineItemBytes)
		if err3 != nil {
			logger.Warningf("Unable to commit shipping status update for in %s ", PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR)
		}
	}

}

/*
	Method: updateDistributorIncomingIOT
	Executed when new IOT data is recived, this method will update specic line items that
	distributor is responsible for fulfilling.
	This method can emit delivery event if based on incoming iot data we determine that
	the geo location data is within 1 mile of target destination.
*/
func updateDistributorIncomingIOT(stub shim.ChaincodeStubInterface, poId string, iotInput IotProperty, itemStatus ItemStatus, msgKey string) bool {

	privateCollection := PRIVATE_COLLECTION_CUSTOMER_LINEITEMS
	poPrivateDataResponse, err1 := stub.GetPrivateData(privateCollection, poId)
	if err1 != nil {
		// log poId not found
		logger.Info("unable to find data for: " + poId + " in collection: " + privateCollection)
		return false
	}
	itemPrivateData := LineItemPrivateDetails{}
	json.Unmarshal(poPrivateDataResponse, &itemPrivateData)
	emit_on_delivery := false
	itemMap := make(map[string][]IotProperty)
	var itemsToUpdateMap = make(map[int]LineItem)
	sharedItemsMap := make(map[int]LineItem)
	for i, eachItem := range itemPrivateData.LineItems {
		logger.Infof("trying index %d lineNumber %d", i, eachItem.LineNumber)
		if eachItem.IotTrackingCode != iotInput.TrackingCode {
			logger.Infof("tracking code doesn't match skipping index %d linenumber: %d", i, eachItem.LineNumber)
			continue
		}
		if eachItem.Status == STATUS_DELIVERED || eachItem.Status == STATUS_RECEIVED {
			logger.Infof("item status is %s skipping index %d linenumber: %d", eachItem.Status, i, eachItem.LineNumber)
			continue
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
		// calculate distance
		mi := distanceFromProjectSite(eachItem.ShipToLocation, iotInput)
		if mi < 1 {
			itemPrivateData.LineItems[i].Status = STATUS_RECEIVED
			itemPrivateData.LineItems[i].ProgressStatus = append(itemPrivateData.LineItems[i].ProgressStatus, itemStatus)
			emit_on_delivery = true
		}
		itemsToUpdateMap[eachItem.LineNumber] = itemPrivateData.LineItems[i]
		logger.Infof("map length: %d", len(itemsToUpdateMap))
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
	}
	err = stub.PutPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, poId, pdLineItemBytes)
	if err != nil {
		logger.Info("unable to commit data for: " + poId + " in collection: " + privateCollection + " error: " + err.Error())
		return false
	}
	updateDistributorFulmentIotProperties(stub, poId, emit_on_delivery, itemsToUpdateMap, itemStatus)
	if emit_on_delivery {
		// Add progress to shared table
		updateSharedProgressRecord(stub, poId, sharedItemsMap, itemStatus, MODE_ITEM_DELIVERED)
		//event to emit
		logger.Infof("item should now be in delivery status")
		var event = ItemDeliveryEvent{Type: msgKey, Status: STATUS_DELIVERED, SkipDistributor: false, PoId: poId, TrackingCode: iotInput.TrackingCode, ItemMap: itemMap, ProgressStatus: itemStatus}
		event = updateLogisticsDeliveryStatus(stub, poId, event)
	}
	return true
}

/*
	Method: updateDistributorFulmentIotProperties
	A helper method for updating related distributor collection with latest IOT data
*/
func updateDistributorFulmentIotProperties(stub shim.ChaincodeStubInterface, poId string, isDelivered bool, itemsToUpdateMap map[int]LineItem, progressStatus ItemStatus) {

	distributorPricingResponse, _ := stub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR, poId)
	if distributorPricingResponse != nil {
		distributorPricing := LineItemCDPrivateDetails{}
		json.Unmarshal(distributorPricingResponse, &distributorPricing)
		for i, eachItem := range distributorPricing.LineItems {
			lineItem, keyFound := itemsToUpdateMap[eachItem.LineNumber]
			if !keyFound {
				logger.Infof("key not found: %d ", lineItem.LineNumber)
				continue
			}
			distributorPricing.LineItems[i].IotProperties = lineItem.IotProperties
			distributorPricing.LineItems[i].IotTrackingCode = lineItem.IotTrackingCode
			if isDelivered {
				logger.Infof("item is delviered, lineNumber: %d ", lineItem.LineNumber)
				distributorPricing.LineItems[i].Status = STATUS_DELIVERED
				distributorPricing.LineItems[i].ProgressStatus = append(distributorPricing.LineItems[i].ProgressStatus, progressStatus)
			}
		}
		lineItemBytes, err3 := json.Marshal(distributorPricing)
		if err3 != nil {
			logger.Warningf("Unable to Marshal distributor pricing object for: %s ", poId)
		} else {
			logger.Infof("trying to save in distributor pricing")
			err3 = stub.PutPrivateData(PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR, poId, lineItemBytes)
			if err3 != nil {
				logger.Warningf("Unable to commit shipping status update for in %s ", PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR)
			} else {
				logger.Infof("Still here ... and data comitted %s", distributorPricing)
			}
		}
	}
}
