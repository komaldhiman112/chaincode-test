package main

/* Imports
* utility libraries for handling bytes, reading and writing JSON,
formatting, and string manipulation
* 2 specific Hyperledger Fabric specific libraries for Smart Contracts
* 1 specific (haversine) for gps calcuations
*
* //
*/

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	str "strings"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/msp"
	sc "github.com/hyperledger/fabric/protos/peer"
)

// Define the Smart Contract structure
type SmartContract struct {
}

var logger = shim.NewLogger("chaincode")
var currentMspId = ""

// var organizationMap = make(map[string]string)
var organizationMap = map[string]string{
	"org1msp": "Utility",
	"org2msp": "Distributor",
	"org3msp": "Manufacturer 1",
	"org4msp": "Manufacturer 2",
	"org5msp": "Logistics",
}

func main() {

	err := shim.Start(new(SmartContract))
	if err != nil {
		fmt.Printf("Error creating new Material Trace Smart Contract: %s", err)
	}

	// organizationMap["org1msp"] = "Utility"
	// organizationMap["org2msp"] = "Distributor"
	// organizationMap["org3msp"] = "Manufacturer 1"
	// organizationMap["org4msp"] = "Manufacturer 2"
	// organizationMap["org5msp"] = "Logistics"
	// organizationMap["org6msp"] = "Warehouse 1"
	// organizationMap["org7msp"] = "Warehouse 2"

}

const (
	PRIVATE_COLLECTION_CUSTOMER_LINEITEMS        = "collectionCustomerLineItems"
	PRIVATE_COLLECTION_CUSTOMER_DISTRIBUTOR      = "collectionCustomerDistributor"
	PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER1 = "collectionDistributorManufacturer1"
	PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER2 = "collectionDistributorManufacturer2"
	PRIVATE_COLLECTION_LOGISTICS                 = "collectionLogistics"
	PRIVATE_COLLECTION_MTR_MFR1                  = "collectionMtrManufacturer1"
	PRIVATE_COLLECTION_MTR_MFR2                  = "collectionMtrManufacturer2"
	PRIVATE_COLLECTION_GENERAL_PROGRESS          = "collectionGeneralProgress"
	DEFAULT_CURRENCY                             = "USD"
	DEFAULT_MATERIAL_GROUP                       = "pipe"
	DEFAULT_UNIT_OF_MEASURE                      = "each"
	ORDER_STATUS_DISTRIBUTOR_FULFILLMENT         = "distributor fulfillment"
	STATUS_OPEN                                  = "open"
	STATUS_REJECTED                              = "rejected"
	STATUS_WIP                                   = "wip"
	STATUS_ACCEPTED                              = "accepted"
	STATUS_SHIPPED                               = "shipped"
	STATUS_IN_TRANSIT                            = "in-transit"
	STATUS_DELIVERED                             = "delivered"
	STATUS_RECEIVED                              = "received"
	STATUS_VERIFIED                              = "verified"
)

// handleValidateOrderRequest
/*
 * The Init method *
 called when the Smart Contract materialtrace chaincode is instantiated by the network
 * Best practice is to have any Ledger initialization in separate function
 -- see initLedger()
*/
func (s *SmartContract) Init(stub shim.ChaincodeStubInterface) sc.Response {
	if _, args := stub.GetFunctionAndParameters(); len(args) > 0 {
		return Error(http.StatusBadRequest, "Init: Incorrect number of arguments; no arguments were expected.")
	}
	return Success(http.StatusOK, "OK", nil)
}

func (s *SmartContract) Invoke(stub shim.ChaincodeStubInterface) sc.Response {
	// Route call to the correct function
	function, args := stub.GetFunctionAndParameters()
	function = str.ToLower(function)
	creatorByte, err := stub.GetCreator()
	if err != nil {
		return Error(http.StatusBadRequest, "Invoke: Membership details missing.")
	}
	si := &msp.SerializedIdentity{}
	err2 := proto.Unmarshal(creatorByte, si)
	if err2 != nil {
		errMsg := fmt.Sprintf("Unable to determine Organization %s \n", err2.Error())
		fmt.Println(errMsg)
		return shim.Error(errMsg)
	}
	currentMspId = str.ToLower(si.Mspid)
	fmt.Printf("si.Mspid %s Creator  %s target function %s \n", si.Mspid, currentMspId, function)
	switch function {
	case "initledger":
		validMsps := "org1msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, only customer organization can initialize PO.  Found:  " + si.Mspid)
		}
		// return Success(http.StatusOK, "OK", nil)
		return s.initLedger(stub, args)
	case "createpo":
		validMsps := "org1msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, only the customer org, can create a PO")
		}
		return s.createPo(stub, args)
	case "acceptpo":
		// distributor or one of the two manufacturers
		validMsps := "org2msp" // |org3msp|org4msp
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, only the distributor or manufacturerer org, can accept a PO")
		}
		return s.acceptPo(stub, args)
	case "open-order-requests":
		validMsps := "org2msp|org3msp|org4msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, expecting org2,org3, or org4")
		}
		return s.queryOpenOrderItems(stub, args)
	case "acknowledge-order-request":
		validMsps := "org3msp|org4msp" //org2msp|
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, expecting org2,org3, or org4")
		}
		return s.manufacturerAcknowledgeOrderRequest(stub, args)
		// if str.ToLower(si.Mspid) == "org2msp" {
		// 	return s.distributorAcknowledgeOrderRequest(stub, args)
		// } else {
		// 	return s.manufacturerAcknowledgeOrderRequest(stub, args)
		// }
	case "notifyshiptocustomer":
		// distributor or one of the two manufacturers
		validMsps := "org2msp|org3msp|org4msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, only the distributor or manufacturerer org can request ship to customer")
		}
		return s.notifyShipToCustomer(stub, args)
	case "manufactureracknowledgment":
		validMsps := "org2msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, distributor expected.")
		}
		return s.updateProgressStatusOnMfrAcknowledgement(stub, args)
	case "notifyitemdelivered":
		validMsps := "org2msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, distributor expected.")
		}
		return s.notifyItemDelivered(stub, args)
	case "receiveditemsverified":
		validMsps := "org1msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, utility / customer expected")
		}
		return s.handleValidateOrderRequest(stub, args)
	case "logistics-order-requests":
		validMsps := "org5msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			resMsg := ResponseMessage{}
			resMsg.Success = false
			resMsg.ErrorMessage = "Unexpected organization, expecting logistics but got " + organizationMap[str.ToLower(si.Mspid)]
			msgBytes, _ := json.Marshal(resMsg)
			return shim.Error(string(msgBytes))
		}
		return s.queryLogisticsOpenOrderItems(stub, args)
	case "field-operator-list":
		validMsps := "org1msp|org2msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, expecting org1 or org2 but got " + si.Mspid)
		}
		return s.queryFieldOperatorItems(stub, args)
	case "shipped-to-customer":
		validMsps := "org1msp|org2msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, expecting org1 or org2")
		}
		return s.queryAllCustomer(stub)
	case "acceptandshiptocustomer":
		// shipping organization only
		validMsps := "org5msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, only the shipper organization can ship a package")
		}
		return s.logisticsAcceptAndShipsToCustomer(stub, args)
	case "onlogisticsacceptance":
		validMsps := "org2msp|org3msp|org4msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, only the distributor or manufacturerer org can request ship to customer")
		}
		return s.notifyDistributorOnLogisticsShipment(stub, args)
	case "advanceintransititems":
		validMsps := "org2msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, expecting the distributor")
		}
		return s.advanceInTransitItem(stub, args)
	case "shippeditemslist":
		return s.fetchAllShippedItems(stub, args)
	case "incomingiot":
		return s.incomingIOT2(stub, args)
	case "getall":
		return s.queryAll(stub)
	case "querypo":
		return s.queryPo(stub, args)
	case "queryprivatecollection":
		return s.queryPrivateCollection(stub, args)
	case "lineitemprogressstatus":
		return s.queryLineItemStatus(stub, args)
	case "addmaterialcertificate":
		validMsps := "org3msp|org4msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, only manufacturer can upload mtr")
		}
		return s.addMaterialCertificate(stub, args)
	case "onmanufacturershipmentnotification":
		validMsps := "org2msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, only the distributor or manufacturerer org can request ship to customer")
		}
		return s.notifyDistributorOnMfrShipment(stub, args)
	case "customerorderrecevied":
		// customer only
		validMsps := "org1msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Expecteed customer organization")
		}
		return s.customerReceivingDeptment(stub, args)
	case "history":
		return s.getHistoryForSpecificPO(stub, args)
	case "mtr-list":
		validMsps := "org2msp|org3msp|org4msp"
		if !str.Contains(str.ToLower(validMsps), str.ToLower(si.Mspid)) {
			return shim.Error("Unexpected organization, expecting org2,org3, or org4")
		}
		return s.queryMtrItems(stub, args)
	default:
		logger.Warningf("Invoke('%s') invalid!", function)
		return Error(http.StatusNotImplemented, "Invalid method! Valid methods are 'createpo|'!")
	}
}

func (s *SmartContract) createPo(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}
	item := PurchaseOrder{}
	json.Unmarshal([]byte(args[0]), &item) // converts the json object into PurchaseOrder Struct
	progressStatus := ItemStatus{}
	err := json.Unmarshal([]byte(args[1]), &progressStatus)
	if err != nil {
		return shim.Error("Unable to parse progress status data provided - " + args[1])
	}
	return s.handleCreatePoRequest(stub, item, progressStatus)
}
func generateItemKey(poId string, lineItem LineItem) string {
	itemProps := []string{poId, strconv.Itoa(lineItem.LineNumber), lineItem.MaterialId}
	return str.Join(itemProps, "|")
}

func (s *SmartContract) addMaterialCertificate(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	materialCert := MaterialCertificate{}
	json.Unmarshal([]byte(args[0]), &materialCert)
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1.")
	}
	// materialCert.TrackingId = args[0]
	// materialCert.MaterialGroup = args[1]
	// materialCert.Certificate = args[1]
	// materialCert.Summary = make([]MtrDetails, 2)
	// materialCert.Summary[0] = MtrDetails{
	// 	Label: "Heat Number",
	// 	Value: args[2],
	// }
	// materialCert.Summary[1] = MtrDetails{
	// 	Label: "Description",
	// 	Value: args[3],
	// }
	privateCollection := ""
	switch currentMspId {
	case "org3msp": // "":
		privateCollection = PRIVATE_COLLECTION_MTR_MFR1
		materialCert.ObjectType = PRIVATE_COLLECTION_MTR_MFR1
	case "org4msp": // "manufacturer 2":
		privateCollection = PRIVATE_COLLECTION_MTR_MFR2
		materialCert.ObjectType = PRIVATE_COLLECTION_MTR_MFR2
	}
	mtrBytes, err := json.Marshal(materialCert)
	err = stub.PutPrivateData(privateCollection, materialCert.TrackingId, mtrBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(mtrBytes)
}

func (s *SmartContract) notifyShipToCustomer(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 5 {
		return shim.Error("Incorrect number of arguments. Expecting four arguments. 1. PoId 2. lineItems 3. shippingRequestNumber 4. progress status 5. logistics initial progress status")
	}
	lineItemsToShip := []LineItem{}
	err := json.Unmarshal([]byte(args[1]), &lineItemsToShip)
	if err != nil {
		return shim.Error("Unable to parse lineItem data provided - " + args[1])
	}
	var lineitemToShipMap = make(map[string]LineItem)
	for _, lineItem := range lineItemsToShip {
		// lineitemToShipMap[lineItem.LineNumber] = lineItem
		lineitemToShipMap[lineItem.ItemKey] = lineItem
	}
	poId := args[0]
	shippingRequestNumber, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return shim.Error("Unable to parse shippingRequestNumber provided - " + args[2] + " Expecting an int64 number.")
	}
	progressStatus := ItemStatus{}
	err = json.Unmarshal([]byte(args[3]), &progressStatus)
	if err != nil {
		return shim.Error("Unable to parse progress status data provided - " + args[3])
	}
	logisticsInitialStatus := make([]ItemStatus, 1)
	err = json.Unmarshal([]byte(args[4]), &logisticsInitialStatus)
	if err != nil {
		return shim.Error("Unable to parse progress status data provided - " + args[4])
	}

	// iotTrackingCode := args[1]
	privateCollection := ""
	isDistributor := false
	shippingRequestedBy := ""
	switch currentMspId {
	case "org3msp": // "manufacturer 2"::
		privateCollection = PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER1
		shippingRequestedBy = organizationMap["org3msp"] // "Manufacturer 1"
	case "org4msp": // "manufacturer 2":
		privateCollection = PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER2
		shippingRequestedBy = organizationMap["org4msp"] // "Manufacturer 2"
	default:
		privateCollection = PRIVATE_COLLECTION_CUSTOMER_LINEITEMS
		isDistributor = true
		shippingRequestedBy = organizationMap["org2msp"] // "Distributor"
	}

	poPrivateDataResponse, err1 := stub.GetPrivateData(privateCollection, poId)
	if err1 != nil {
		logger.Info("Unable to get " + privateCollection + " data for PO: " + poId)
	} else {
		shippingPd := ShippingPrivateDetails{}
		shippingPrivateDataResponse, _ := stub.GetPrivateData(PRIVATE_COLLECTION_LOGISTICS, poId)
		hasExistingData := false
		if shippingPrivateDataResponse != nil {
			json.Unmarshal(shippingPrivateDataResponse, &shippingPd)
			hasExistingData = true
		} else {
			shippingPd.LineItems = make([]ShippingLineItem, 1)
		}
		shippingPd.ObjectType = PRIVATE_COLLECTION_LOGISTICS
		shippingPd.PoId = poId
		// ShippingRequest
		sharedItemsMap := make(map[int]LineItem)
		if poPrivateDataResponse != nil && isDistributor {
			itemPrivateData := LineItemPrivateDetails{}
			json.Unmarshal(poPrivateDataResponse, &itemPrivateData)
			shippingItemCount := 0
			for i, eachItem := range itemPrivateData.LineItems {
				// lineItem := lineitemToShipMap[eachItem.LineNumber]
				lineItem := lineitemToShipMap[eachItem.ItemKey]
				if eachItem.LineNumber != lineItem.LineNumber {
					continue
				}
				itemPrivateData.LineItems[i].MaterialCertificate = lineItem.MaterialCertificate
				itemPrivateData.LineItems[i].TimeShipped = lineItem.TimeShipped
				itemPrivateData.LineItems[i].IotTrackingCode = lineItem.IotTrackingCode
				itemPrivateData.LineItems[i].ShippingRequestNumber = shippingRequestNumber
				itemPrivateData.LineItems[i].ProgressStatus = append(itemPrivateData.LineItems[i].ProgressStatus, progressStatus)
				lineItem.ShippingRequestNumber = shippingRequestNumber
				sharedItemsMap[lineItem.LineNumber] = lineItem
				if shippingItemCount == 0 {
					timeStamp := itemPrivateData.LineItems[i].ProgressStatus[0].TimeStamp
					logisticsInitialStatus[0].TimeStamp = timeStamp
				}
				for j, orderItem := range eachItem.OrderRequests {
					if orderItem.LineNumber != lineItem.LineNumber {
						continue
					}
					eachItem.OrderRequests[j].Status = STATUS_SHIPPED // "readyforshipment"
					eachItem.OrderRequests[j].IotTrackingCode = lineItem.IotTrackingCode
					eachItem.OrderRequests[j].TimeShipped = lineItem.TimeShipped
					// shippingPd = fillShippingLineItems(poId, lineItem.PoNumber, lineItem.LineNumber, lineItem.ShipToLocation, shippingRequestedBy, hasExistingData, shippingItemCount, shippingPd, lineItem)
					shippingPd = fillShippingLineItems(poId, shippingRequestNumber, lineItem, shippingRequestedBy, hasExistingData, shippingItemCount, shippingPd, logisticsInitialStatus, progressStatus)
					shippingItemCount += 1
				}
			}

			// ADD TO SHARED RECORD
			updateSharedProgressRecord(stub, poId, sharedItemsMap, progressStatus, MODE_ITEM_SHIPPED)

			pdLineItemBytes, err := json.Marshal(itemPrivateData)
			if err != nil {
				return shim.Error(err.Error())
			}
			err = stub.PutPrivateData(PRIVATE_COLLECTION_CUSTOMER_LINEITEMS, poId, pdLineItemBytes)
			if err != nil {
				return shim.Error(err.Error())
			}
			commitShippingPrivateData(stub, poId, shippingPd)
			// // update distributor
			updateDistributorFulfilledLineItems(stub, poId, STATUS_SHIPPED, lineitemToShipMap, progressStatus)
			return shim.Success(pdLineItemBytes)

		} else if poPrivateDataResponse != nil {
			itemPrivateData := LineItemCDPrivateDetails{}
			json.Unmarshal(poPrivateDataResponse, &itemPrivateData)
			shippingItemCount := 0

			for i, priceInfo := range itemPrivateData.LineItems {
				// lineItem := lineitemToShipMap[priceInfo.LineNumber]
				lineItem := lineitemToShipMap[priceInfo.ItemKey]
				if priceInfo.LineNumber != lineItem.LineNumber {
					continue
				}
				itemPrivateData.LineItems[i].Status = progressStatus.Status // STATUS_SHIPPED
				itemPrivateData.LineItems[i].ProgressStatus = append(itemPrivateData.LineItems[i].ProgressStatus, progressStatus)
				itemPrivateData.LineItems[i].IotTrackingCode = lineItem.IotTrackingCode
				itemPrivateData.LineItems[i].MaterialCertificate = lineItem.MaterialCertificate
				itemPrivateData.LineItems[i].TimeShipped = lineItem.TimeShipped
				lineItem.ShippingRequestNumber = shippingRequestNumber
				sharedItemsMap[lineItem.LineNumber] = lineItem
				// shippingPd = fillShippingLineItems(poId, priceInfo.PoNumber, priceInfo.LineNumber, priceInfo.ShipToLocation, shippingRequestedBy, hasExistingData, shippingItemCount, shippingPd, lineItem)
				shippingPd = fillShippingLineItems(poId, shippingRequestNumber, lineItem, shippingRequestedBy, hasExistingData, shippingItemCount, shippingPd, logisticsInitialStatus, progressStatus)
				shippingItemCount += 1
			}

			// ADD TO SHARED RECORD
			updateSharedProgressRecord(stub, poId, sharedItemsMap, progressStatus, MODE_ITEM_SHIPPED)

			pdLineItemBytes, err := json.Marshal(itemPrivateData)
			if err != nil {
				return shim.Error(err.Error())
			}
			err = stub.PutPrivateData(privateCollection, poId, pdLineItemBytes)
			if err != nil {
				return shim.Error(err.Error())
			}
			commitShippingPrivateData(stub, poId, shippingPd)
			return shim.Success(pdLineItemBytes)
		}
	}
	return shim.Error("Not Found")
}

func (s *SmartContract) incomingIOT2(stub shim.ChaincodeStubInterface, args []string) sc.Response {

	fmt.Println("length of incoming IOT: ", len(args))
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4")
	}
	poId := args[0]
	iotInput := IotProperty{}
	json.Unmarshal([]byte(args[1]), &iotInput)
	itemStatus := ItemStatus{}
	json.Unmarshal([]byte(args[2]), &itemStatus)
	messageKey := args[3]
	// itemStatus := ItemStatus{
	// 	Owner:     organizationMap["org1msp"], // "Utility",
	// 	Status:    STATUS_RECEIVED,
	// 	TimeStamp: iotInput.Timestamp,
	// }

	privateCollection := ""
	isDistributor := false
	switch currentMspId {
	case "org3msp": // "":
		privateCollection = PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER1
	case "org4msp": // "manufacturer 2":
		privateCollection = PRIVATE_COLLECTION_DISTRIBUTOR_MANUFACTURER2
	default:
		privateCollection = PRIVATE_COLLECTION_CUSTOMER_LINEITEMS
		isDistributor = true
	}
	ok := false
	if !isDistributor {
		ok = updateManufacturerIncomingIOT(stub, privateCollection, poId, iotInput, itemStatus, messageKey)
		if !ok {
			return shim.Error("Unable to update Manufacturer IOT data for " + currentMspId)
		}
		// ok = updateDistributorIncomingIOT(stub, poId, iotInput)
	} else {
		ok = updateDistributorIncomingIOT(stub, poId, iotInput, itemStatus, messageKey)
	}
	if !ok {
		return shim.Error("Unable to update IOT data for " + currentMspId)
	}
	// return Success(http.StatusOK, "Sucessfully recorded IOT data", nil)
	poArrayAsBytes, _ := json.Marshal(iotInput)
	return shim.Success(poArrayAsBytes)

}

/*
 * Utility Methods
 */

/*
* Given a json object convert to PurchaseOrder struct
 */
func (po *PurchaseOrder) FromJson(input []byte) *PurchaseOrder {
	json.Unmarshal(input, po)
	return po
}

/*
* Given a PurchaseOrder object, convert to json object
 */
func (po *PurchaseOrder) ToJson() []byte {
	jsonPo, _ := json.Marshal(po)
	return jsonPo
}

/*
*	Handle success response
 */
func Success(rc int32, doc string, payload []byte) sc.Response {
	return sc.Response{
		Status:  rc,
		Message: doc,
		Payload: payload,
	}
}

/*
* Handle error response
 */
func Error(rc int32, doc string) sc.Response {
	logger.Errorf("Error %d = %s", rc, doc)
	return sc.Response{
		Status:  rc,
		Message: doc,
	}
}
