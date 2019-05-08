package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	sc "github.com/hyperledger/fabric/protos/peer"
)

func (s *SmartContract) handleCreatePoRequest(stub shim.ChaincodeStubInterface, item PurchaseOrder, progressStatus ItemStatus) sc.Response {

	id := item.PoId
	// Validate that poId does not yet exist. If the key does not exist (nil, nil) is returned.
	if value, err := stub.GetState(item.PoId); !(err == nil && value == nil) {
		msg := fmt.Sprintf("purchase order with id %s exists", id)
		return Error(http.StatusConflict, msg)
	}
	item.PoStatus = STATUS_OPEN
	// item.Custodian = "Customer"
	// item.CurrentJourney = Journey{
	// 	IssuingAgent:       "customer",
	// 	JourneyRole:        "Customer",
	// 	Destination:        item.IssuedTo,
	// 	ReceivingAgentRole: "SupplierMfg",
	// }
	pdLineItem := LineItemPrivateDetails{}
	pdLineItem.LineItems = make([]LineItem, len(item.LineItems))
	pdLineItem.ObjectType = PRIVATE_COLLECTION_CUSTOMER_LINEITEMS
	pdLineItem.PoId = item.PoId
	// progressStatus := ItemStatus{
	// 	Owner:     organizationMap["org1msp"], // "Utility",
	// 	Status:    STATUS_OPEN,
	// 	TimeStamp: item.CreatedTimeStamp,
	// }
	sharedDetails := make([]SharedLineDetail, 1)
	for i, lineItem := range item.LineItems {
		key := generateItemKey(item.PoId, lineItem)
		lineItem.ItemKey = key
		item.LineItems[i].ItemKey = key
		item.LineItems[i].Subtotal = math.Round(float64(lineItem.Quantity) * lineItem.UnitPrice)
		lineItem.Subtotal = math.Round(float64(lineItem.Quantity) * lineItem.UnitPrice)
		if item.LineItems[i].Currency == "" {
			item.LineItems[i].Currency = DEFAULT_CURRENCY
			lineItem.Currency = DEFAULT_CURRENCY
		}
		if item.LineItems[i].MaterialGroup == "" {
			lineItem.MaterialGroup = DEFAULT_MATERIAL_GROUP
			item.LineItems[i].MaterialGroup = DEFAULT_MATERIAL_GROUP
		}
		if item.LineItems[i].UnitOfMeasure == "" {
			lineItem.UnitOfMeasure = DEFAULT_UNIT_OF_MEASURE
			item.LineItems[i].UnitOfMeasure = DEFAULT_UNIT_OF_MEASURE
		}
		lineItem.ProgressStatus = make([]ItemStatus, 1)
		lineItem.ProgressStatus[0] = progressStatus
		sharedInfo := fillSharedInfo(lineItem, item.PoId)
		if i == 0 {
			sharedDetails[0] = sharedInfo
		} else {
			sharedDetails = append(sharedDetails, sharedInfo)
		}
		pdLineItem.LineItems[i] = lineItem
	}

	// will use this object to share the progress status on PM screen
	addNewShareProgressRecord(stub, item.PoId, sharedDetails)

	poLineItems := pdLineItem.LineItems // will be returned to ui client
	item.LineItems = make([]LineItem, 0)
	poAsBytes, _ := json.Marshal(item) // convert PO struct into bytes
	// // add key value
	if err := stub.PutState(id, poAsBytes); err != nil {
		return Error(http.StatusInternalServerError, err.Error())
	}
	pdLineItemBytes, err := json.Marshal(pdLineItem)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, item.PoId, pdLineItemBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	item.LineItems = poLineItems
	var event = CustomEvent{Type: "pocreated", Description: "Po Created", Status: item.PoStatus, Id: item.PoId, PoNumber: item.PoNumber, LineItems: item.LineItems}
	eventBytes, err := json.Marshal(&event)
	if err != nil {
		fmt.Println("unable to marshal event ", err)
	}
	err = stub.SetEvent(event.Type, eventBytes)
	if err != nil {
		fmt.Println("Could not set event for Po created ", err)
	} else {
		fmt.Println("Event set - " + event.Description)
	}

	return shim.Success(item.ToJson())
	// return Success(http.StatusCreated, "PurchaseOrder Created", nil)

}
func (s *SmartContract) handleValidateOrderRequest(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 2 {
		return shim.Error("Expecting two arguments 1. poId 2. ShippingRequestId")
	}
	poId := args[0]
	shippingRequestNumber, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return shim.Error("Invalid number format, expecting a numbeer on argument 2")
	}
	collectionName := PRIVATE_COLLECTION_CUSTOMER_LINEITEMS
	indexMap := make(map[string]int)
	indexByLineNumberMap := make(map[int]int)
	customerDataResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, poId)
	pLineItem := LineItemPrivateDetails{}
	if err1 != nil {
		logger.Info("Unable to get PRIVATE_COLLECTION_CUSTOMER_LINEITEMS data for PO: " + poId)
	} else {
		if customerDataResponse != nil {
			json.Unmarshal(customerDataResponse, &pLineItem)
			for i, lineItem := range pLineItem.LineItems {
				indexMap[lineItem.ItemKey] = i
				indexByLineNumberMap[lineItem.LineNumber] = i
			}
		}
	}
	logisticsResponse, err1 := stub.GetPrivateData(PRIVATE_COLLECTION_LOGISTICS, poId)
	// shippedLineItems := make([]ShippingLineItem, 1)
	updatedCount := 0
	updatedLineItems := make([]LineItem, 1)
	shippedLineItems := make([]GoodReceipt, 1)
	if err1 == nil && logisticsResponse != nil {
		privateData := ShippingRequest{}
		json.Unmarshal(logisticsResponse, &privateData)
		for i, shippingInfo := range privateData.LineItems {
			if shippingInfo.ShippingRequestNumber != shippingRequestNumber {
				continue
			}
			index, found := indexByLineNumberMap[shippingInfo.LineNumber]
			if !found {
				continue
			}
			if pLineItem.LineItems[index].ShippingRequestNumber == 0 {
				logger.Infof("set shipping request number from logistics table for line number %d ", pLineItem.LineItems[index].LineNumber)
				pLineItem.LineItems[index].ShippingRequestNumber = shippingInfo.ShippingRequestNumber
			}
			for _, statusInfo := range shippingInfo.ProgressStatus {
				if statusInfo.Status == STATUS_RECEIVED && pLineItem.LineItems[index].TimeReceived == 0 {
					pLineItem.LineItems[index].TimeReceived = statusInfo.TimeStamp
					logger.Infof("set TimeReceived from logistics table for line number %d ", pLineItem.LineItems[index].LineNumber)
					break
				}
			}
			pLineItem.LineItems[index].Status = STATUS_VERIFIED
			if updatedCount == 0 {
				updatedLineItems[0] = pLineItem.LineItems[i]
			} else {
				updatedLineItems = append(updatedLineItems, pLineItem.LineItems[i])
			}

			goodReciept := GoodReceipt{
				MaterialCertificate: pLineItem.LineItems[index].MaterialCertificate,
				ShippedLineItem:     shippingInfo}
			if updatedCount == 0 {
				shippedLineItems[0] = goodReciept
			} else {
				shippedLineItems = append(shippedLineItems, goodReciept)
			}

			updatedCount += 1
		}
		if updatedCount > 0 {
			itemBytes, err2 := json.Marshal(pLineItem)
			if err2 != nil {
				logger.Info("unable to marshal private data for: " + poId + " in collection: " + collectionName + " error: " + err2.Error())
				// return shim.Error(err.Error())
			}
			err2 = stub.PutPrivateData(collectionName, poId, itemBytes)
			if err2 != nil {
				// return shim.Error(err.Error())
				logger.Info("unable to commit data for: " + poId + " in collection: " + collectionName + " error: " + err2.Error())
				// return shim.Error(err.Error())
			}
		}
	}
	rsBytes, _ := json.Marshal(shippedLineItems)
	return shim.Success(rsBytes)

	// resMsg := ResponseMessage{}
	// // responseMessage
	// updatedCount := 0
	// updatedLineItems := make([]LineItem, 1)
	// for i, eachItem := range pItem.LineItems {
	// 	if eachItem.ShippingRequestNumber == shippingRequestNumber {
	// 		pItem.LineItems[i].Status = STATUS_VERIFIED
	// 		if updatedCount == 0 {
	// 			updatedLineItems[0] = pItem.LineItems[i]
	// 		} else {
	// 			updatedLineItems = append(updatedLineItems, pItem.LineItems[i])
	// 		}
	// 		updatedCount += 1
	// 	}
	// }
	// if updatedCount > 0 {
	// 	// po := PurchaseOrder{}
	// 	// if value, err := stub.GetState(poId); err == nil && value != nil {
	// 	// 	json.Unmarshal(value, &po)
	// 	// 	po.IsFinalized = true
	// 	// 	poBytes, err := json.Marshal(pItem)
	// 	// 	err = stub.PutState(poId, poBytes)
	// 	// 	if err == nil {
	// 	// 		logger.Info("unable to update state for po: " + poId + " error: " + err.Error())
	// 	// 	}
	// 	// }
	// 	itemBytes, err2 := json.Marshal(pItem)
	// 	if err2 != nil {
	// 		logger.Info("unable to marshal private data for: " + poId + " in collection: " + collectionName + " error: " + err2.Error())
	// 		return shim.Error(err.Error())
	// 	}
	// 	err2 = stub.PutPrivateData(collectionName, poId, itemBytes)
	// 	if err2 != nil {
	// 		// return shim.Error(err.Error())
	// 		logger.Info("unable to commit data for: " + poId + " in collection: " + collectionName + " error: " + err2.Error())
	// 		return shim.Error(err.Error())
	// 	}
	// }

	// // resMsg.Success = true
	// // resMsg.ErrorMessage = ""
	// // msgBytes, _ := json.Marshal(resMsg)
	// rsBytes, _ := json.Marshal(updatedLineItems)
	// return shim.Success(rsBytes)
}

func (s *SmartContract) customerReceivingDeptment(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	if value, err := stub.GetState(args[0]); err == nil && value != nil {
		po := PurchaseOrder{}
		json.Unmarshal(value, &po)
		// po.Custodian = "Customer"
		// // lastIndex := len(po.Journeys) - 1
		// // if lastIndex < 1 {
		// // 	return shim.Error("Unexpected index")
		// // }
		// // currentJourney := po.Journeys[lastIndex]
		// // currentJourney.ReceivingAgent = "customer"
		// // currentJourney.ReceivingAgentRole = "AcctReceivable"
		// // currentJourney.ReceivingTimestamp = time.Now()
		// // currentJourney.JourneyRole = "AcctReceivable"
		// // po.Journeys[lastIndex] = currentJourney
		// currentJourney := po.CurrentJourney

		// currentJourney.ReceivingAgent = "AcctReceivable"
		// currentJourney.ReceivingAgentRole = "AcctReceivable"
		// currentJourney.ReceivingTimestamp = time.Now()
		// currentJourney.JourneyRole = "AcctReceivable"
		// po.CurrentJourney = currentJourney
		// po.OrderStatus = ORDER_STATUS_CUSTOMER_RECEIVABLE

		if err := stub.PutState(po.PoId, po.ToJson()); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(po.ToJson())
		// return Success(http.StatusOK, "Items received", nil)
	}
	return shim.Error("Not Found")

}

/*  The initLedger method *
This method will be used for Ledger initialization
Method to invoked as needed after init()
*/
func (s *SmartContract) initLedger(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	id1 := "b0c00193-9dca-444f-9d1e-c3712307d5f1"
	id2 := "f9e7f1cd-595d-4be1-99b6-3e928366fe08"
	timeStamp, err := strconv.ParseInt(args[0], 10, 64)
	companyName := "Element Energy"
	if len(args) > 1 {
		companyName = args[1]
	}
	if err != nil {
		return shim.Error("Invalid number format, expecting a numbeer on argument 1")
	}
	Items := []PurchaseOrder{
		PurchaseOrder{
			PoId:                 id1,
			PoNumber:             10001,
			ExpectedDeliveryDate: "2019-02-28",
			CreatedTimeStamp:     timeStamp,
			// PalletId: "pallet-001-8888",
			Owner: Company{
				CompanyId:     "c-0001",
				CompanyType:   "customer",
				Latitude:      41.8818,
				Longitude:     -87.6231,
				Name:          companyName,
				StreetAddress: "1 North Wacker",
				City:          "Chicago",
				State:         "Il",
				Zipcode:       "60606",
			},
			IssuedTo: Company{
				CompanyId:     "c-44401",
				CompanyType:   "distributor",
				Latitude:      40.440624,
				Longitude:     -79.995888,
				Name:          "Distributor Inc.",
				StreetAddress: "600 Grant Street",
				City:          "Pittsburgh",
				State:         "PA",
				Zipcode:       "15219",
			},
			LineItems: []LineItem{
				LineItem{
					MaterialId:  "12010",
					LineNumber:  1,
					PoNumber:    10001,
					Description: "Pipe 20in x 40ft",
					// Manufacturer: "Manufacturer 1",
					ProjectId: "project1",
					UnitPrice: 99.99,
					Quantity:  3,
					Subtotal:  299.97,
					ShipToLocation: Company{
						CompanyId:     "73a2e705-6e72-474d-9027-994f52ed2576",
						CompanyType:   "utility",
						Latitude:      41.8818,
						Longitude:     -87.6231,
						Name:          companyName,
						StreetAddress: "1 North Wacker",
						City:          "Chicago",
						State:         "IL",
						Zipcode:       "60606",
					},
				},
				LineItem{
					MaterialId:  "12012",
					LineNumber:  2,
					PoNumber:    10001,
					Description: "Pipe 22in x 40ft",
					ProjectId:   "project1",
					UnitPrice:   199.99,
					Quantity:    1,
					Subtotal:    199.99,
					ShipToLocation: Company{
						CompanyId:     "73a2e705-6e72-474d-9027-994f52ed2576",
						CompanyType:   "utility",
						Latitude:      41.8818,
						Longitude:     -87.6231,
						Name:          companyName,
						StreetAddress: "1 North Wacker",
						City:          "Chicago",
						State:         "IL",
						Zipcode:       "60606",
					},
				},
			},
		},
		PurchaseOrder{
			PoId:                 id2,
			PoNumber:             10002,
			ExpectedDeliveryDate: "2019-02-28",
			CreatedTimeStamp:     timeStamp,
			Owner: Company{
				CompanyId:     "73a2e705-6e72-474d-9027-994f52ed2576",
				CompanyType:   "customer",
				Latitude:      41.8818,
				Longitude:     -87.6231,
				Name:          companyName,
				StreetAddress: "300 Madison",
				City:          "New York,",
				State:         "NY",
				Zipcode:       "10017",
			},
			IssuedTo: Company{
				CompanyId:     "69559c68-f7c8-453e-a608-aebf589fc430",
				CompanyType:   "distributor",
				Latitude:      40.440624,
				Longitude:     -79.995888,
				Name:          "Distributor Inc.",
				StreetAddress: "600 Grant Street",
				City:          "Pittsburgh",
				State:         "PA",
				Zipcode:       "15219",
			},
			LineItems: []LineItem{
				LineItem{
					MaterialId:  "13011",
					LineNumber:  1,
					PoNumber:    10002,
					Description: "Pipe 10in x 40ft",
					ProjectId:   "project2",
					UnitPrice:   99.99,
					Quantity:    3,
					Subtotal:    299.97,
					ShipToLocation: Company{
						CompanyType:   "utility",
						Latitude:      40.440624,
						Longitude:     -79.995888,
						Name:          companyName,
						StreetAddress: "300 Madison",
						City:          "New York",
						State:         "NY",
						Zipcode:       "10017",
					},
				},
				LineItem{
					MaterialId:  "14015",
					LineNumber:  2,
					PoNumber:    10002,
					Description: "Pipe 12in x 40ft",
					ProjectId:   "project2",
					UnitPrice:   399.99,
					Quantity:    1,
					Subtotal:    399.99,
					ShipToLocation: Company{
						CompanyId:     "c-44401",
						CompanyType:   "utility",
						Latitude:      40.440624,
						Longitude:     -79.995888,
						Name:          companyName,
						StreetAddress: "300 Madison",
						City:          "New York",
						State:         "NY",
						Zipcode:       "10017",
					},
				},
			},
		},
	}

	i := 0
	for i < len(Items) {
		item := Items[i]
		item.PoStatus = STATUS_OPEN
		progressStatus := ItemStatus{
			Owner:     organizationMap["org1msp"], // "Utility",
			Status:    STATUS_OPEN,
			TimeStamp: item.CreatedTimeStamp,
		}
		s.handleCreatePoRequest(stub, item, progressStatus)
		i = i + 1
	}
	return Success(http.StatusOK, "OK", nil)
}
