package main

/*
	Defines purchase order structure
*/
type PurchaseOrder struct {
	ObjectType           string     `json:"docType"` //docType is used to distinguish the various types of objects in state database
	PoId                 string     `json:"poId"`
	PoNumber             int        `json:"poNumber"`
	Owner                Company    `json:"owner"`    // provide sap with shorter ids
	IssuedTo             Company    `json:"issuedTo"` // provide sap with shorter ids
	Comment              string     `json:"comment"`
	PoStatus             string     `json:"poStatus"`  // open, accepted, or rejected
	LineItems            []LineItem `json:"lineItems"` //
	IsFinalized          bool       `json:"isFinalized"`
	AcceptanceTimeStamp  int64      `json:"acceptanceTimeStamp"`
	CreatedTimeStamp     int64      `json:"createdTimeStamp"`
	ExpectedDeliveryDate string     `json:"expectedDeliveryDate"`
	ClientUserAgent      string     `json:"clientUserAgent"`
	ProjectId            string     `json:"projectId"`
}
