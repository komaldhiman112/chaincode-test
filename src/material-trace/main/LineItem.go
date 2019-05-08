package main

/*
 Structures related to PO line Items
*/
type LineItem struct {
	PoNumber              int            `json:"poNumber"`
	LineNumber            int            `json:"lineNumber"`
	MaterialId            string         `json:"materialId"`
	MaterialGroup         string         `json:"materialGroup"`
	ItemKey               string         `json:"itemKey"`
	Description           string         `json:"description"`
	Quantity              int            `json:"quantity"`
	UnitOfMeasure         string         `json:"unitOfMeasure"`
	UnitPrice             float64        `json:"unitPrice"`
	Currency              string         `json:"currency"`
	Subtotal              float64        `json:"subtotal"`
	ShipToLocation        Company        `json:"shipToLocation"`
	ProjectId             string         `json:"projectId"`
	DeliveryDate          string         `json:"deliveryDate"`
	AssignedTo            string         `json:"assignedTo"`
	Status                string         `json:"status"`
	AssignedQty           int            `json:"assignedQty"`
	MfrUnitPrice          float64        `json:"mfrUnitPrice"`
	OrderRequests         []OrderRequest `json:"orderRequests"`
	MaterialCertificate   []Mtr          `json:"materialCertificate"`
	IotTrackingCode       string         `json:"iotTrackingCode"`
	IotProperties         []IotProperty  `json:"iotProperties"`
	TimeShipped           int64          `json:"timeShipped"`
	TimeReceived          int64          `json:"timeReceived"`
	ShippingRequestNumber int64          `json:"shippingRequestNumber"`
	ProgressStatus        []ItemStatus   `json:"progressStatus"`
	AcknowledgedTimeStamp int64          `json:"acknowledgedTimeStamp"`
}

/*
	Defines a structure to identify which party is fulfilling
	a specific order lineItem.
*/
type OrderRequest struct {
	MaterialId            string       `json:"materialId"`
	LineNumber            int          `json:"lineNumber"`
	Quantity              int          `json:"quantity"`
	Status                string       `json:"status"`
	FulfilledBy           string       `json:"fulfilledBy"`
	IotTrackingCode       string       `json:"iotTrackingCode"`
	ProgressStatus        []ItemStatus `json:"progressStatus"`
	AcknowledgedTimeStamp int64        `json:"acknowledgedTimeStamp"`
	TimeShipped           int64        `json:"timeShipped"`
}

/*
	A name value structure for material certificate
*/
type Mtr struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

/*
  Defines a structure for private data that is to be shared only by customer and distributor
*/
type LineItemPrivateDetails struct {
	ObjectType string     `json:"docType"` //docType is used to distinguish the various types of objects in state database
	PoId       string     `json:"poId"`
	LineItems  []LineItem `json:"lineItems"`
}

/*
	Defines a structure for private data shared between distributor and manufacturers
*/
type LineItemCDPrivateDetails struct {
	ObjectType string            `json:"docType"` //docType is used to distinguish the various types of objects in state database
	PoId       string            `json:"poId"`
	LineItems  []LineItemPricing `json:"lineItems"`
}

/*
	Defines the structure for line items with pricing information
	specific to a manufacturerer
*/
type LineItemPricing struct {
	PoId                  string        `json:"poId"`
	PoNumber              int           `json:"poNumber"`
	LineNumber            int           `json:"lineNumber"`
	MaterialId            string        `json:"materialId"`
	MaterialGroup         string        `json:"materialGroup"`
	ItemKey               string        `json:"itemKey"`
	Description           string        `json:"description"`
	Manufacturer          string        `json:"manufacturer"`
	Quantity              int           `json:"quantity"`
	UnitOfMeasure         string        `json:"unitOfMeasure"`
	UnitPrice             float64       `json:"unitPrice"`
	Currency              string        `json:"currency"`
	Subtotal              float64       `json:"subtotal"`
	ShipToLocation        Company       `json:"shipToLocation"`
	Status                string        `json:"status"`
	AssignedTo            string        `json:"assignedTo"`
	ProjectId             string        `json:"projectId"`
	DeliveryDate          string        `json:"deliveryDate"`
	MaterialCertificate   []Mtr         `json:"materialCertificate"`
	IotTrackingCode       string        `json:"iotTrackingCode"`
	IotProperties         []IotProperty `json:"iotProperties"`
	AcknowledgedTimeStamp int64         `json:"acknowledgedTimeStamp"`
	TimeShipped           int64         `json:"timeShipped"`
	ProgressStatus        []ItemStatus  `json:"progressStatus"`
}

/*
   Defines structure for private data for logistics
*/
type ShippingPrivateDetails struct {
	ObjectType string             `json:"docType"` //docType is used to distinguish the various types of objects in state database
	PoId       string             `json:"poId"`
	LineItems  []ShippingLineItem `json:"lineItems"`
}

type ShippingRequest struct {
	ShipToLocation        Company            `json:"shipToLocation"`
	RequestedBy           string             `json:"requestedBy"`
	ShippingRequestNumber int64              `json:"shippingRequestNumber"`
	IotTrackingCode       string             `json:"iotTrackingCode"`
	LineItems             []ShippingLineItem `json:"lineItems"`
}

/*
   Defines the lineitem structure for for logistics
*/
type ShippingLineItem struct {
	PoId                  string       `json:"poId"`
	PoNumber              int          `json:"poNumber"`
	LineNumber            int          `json:"lineNumber"`
	Quantity              int          `json:"quantity"`
	UnitOfMeasure         string       `json:"unitOfMeasure"`
	ShippingRequestNumber int64        `json:"shippingRequestNumber"`
	MaterialId            string       `json:"materialId"` // don't think logistics need this, but needed current design
	Description           string       `json:"description"`
	RequestedBy           string       `json:"requestedBy"`
	IotTrackingCode       string       `json:"iotTrackingCode"`
	ShipToLocation        Company      `json:"shipToLocation"`
	Status                string       `json:"status"`
	DeliveryDate          string       `json:"deliveryDate"`
	TimeRequested         int64        `json:"timeRequested"`
	TimeShipped           int64        `json:"timeShipped"`
	ProgressStatus        []ItemStatus `json:"progressStatus"`
}

/*
  Defines a structure for data that need to be shared back with customer such material certificate
  Iot data
*/
type SharedProgressReport struct {
	ObjectType string             `json:"docType"`
	PoId       string             `json:"poId"`
	LineItems  []SharedLineDetail `json:"lineItems"`
}
type SharedLineDetail struct {
	PoNumber              int           `json:"poNumber"`
	LineNumber            int           `json:"lineNumber"`
	ItemKey               string        `json:"itemKey"`
	AssignedTo            string        `json:"assignedTo"`
	MaterialCertificate   []Mtr         `json:"materialCertificate"`
	IotTrackingCode       string        `json:"iotTrackingCode"`
	IotProperties         []IotProperty `json:"iotProperties"`
	TimeShipped           int64         `json:"timeShipped"`
	TimeReceived          int64         `json:"timeReceived"`
	ShippingRequestNumber int64         `json:"shippingRequestNumber"`
	ProgressStatus        []ItemStatus  `json:"progressStatus"`
}
type ItemStatus struct {
	Owner     string `json:"owner"`
	Status    string `json:"status"`
	TimeStamp int64  `json:"timeStamp"`
}
