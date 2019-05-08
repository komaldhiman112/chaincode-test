package main

/*
	Defines a structure to emit events from blockchain
*/
type CustomEvent struct {
	Type        string     `json:"type"`
	Description string     `json:"description"`
	Id          string     `json:"id"`
	PoNumber    int        `json:"poNumber"`
	Status      string     `json:"status"`
	Custodian   string     `json:"custodian"`
	LineItems   []LineItem `json:"lineItems"`
	TimeStamp   int64      `json:"timeStamp"`
}

/*
	Defines a structure for event emitted when it's determined an item has
	arrived to specific destination.  This is determined by based on
	geolocation calculations.
*/
type ItemDeliveryEvent struct {
	Type            string `json:"type"`
	Status          string `json:"status"`
	SkipDistributor bool   `json:"skipDistributor"`
	PoId            string `json:"poId"`
	// TimeShipped     int64                    `json:"timeShipped"`
	TrackingCode          string                   `json:"iotTrackingCode"`
	ItemMap               map[string][]IotProperty `json:"itemMap"`
	ShippingRequestNumber int64                    `json:"shippingRequestNumber"`
	ShippedLineItems      []ShippingLineItem       `json:"shippedLineItems"`
	ProgressStatus        ItemStatus               `json:"progressStatus"`
}

/*
  Defines a structure for messages returned to client
*/
type ResponseMessage struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage"`
}

/*
	A utility structure for discount given by a particular
	manufacturer to a distributor, used to calculate line item price
	that a distributor pays to the manufacturer
*/
type ManufacturerPricingDiscount struct {
	Name     string `json:"name"`
	Discount int    `json:"discount"`
}

/*
  Defines a structure for goods receipt used for integration with SAP
*/
type GoodReceipt struct {
	MaterialCertificate []Mtr            `json:"materialCertificate"`
	ShippedLineItem     ShippingLineItem `json:"shippedLineItem"`
}

/*
	Defines a structure for query results for field operator screen
*/
type FieldOperatorReport struct {
	ShippedItemsMap          map[int]ShippingLineItem     `json:"shippedItemsMap"`
	ShippingRequestMap       map[int64][]ShippingLineItem `json:"sippingRequestMap"`
	PmViewItem               PmView                       `json:"pmViewItem"`
	OriginalPo               PurchaseOrder                `json:"po"`
	DistributorLineItemMap   map[int]LineItemPricing      `json:"distributorLineItemMap"`
	Manufacturer1LineItemMap map[int]LineItemPricing      `json:"manufacturer1LineItemMap"`
	Manufacturer2LineItemMap map[int]LineItemPricing      `json:"manufacturer2LineItemMap"`
	ProgressReportMap        map[int]SharedLineDetail     `json:"progressMap"`
}

/*
  Utility structure for query results specific to open order requests
*/
type PricingResults struct {
	PoId                 string            `json:"poId"`
	PoNumber             int               `json:"poNumber"`
	Owner                Company           `json:"owner"`
	PoStatus             string            `json:"poStatus"` // Accepted, rejected etc
	ExpectedDeliveryDate string            `json:"expectedDeliveryDate"`
	LineItems            []LineItemPricing `json:"lineItems"`
}

/*
	Utility structure for query results for logistics screen
*/
type ShippingRequestsResults struct {
	PoId                 string             `json:"poId"`
	PoNumber             int                `json:"poNumber"`
	PoStatus             string             `json:"poStatus"` // Accepted, rejected etc
	Owner                Company            `json:"owner"`
	ExpectedDeliveryDate string             `json:"expectedDeliveryDate"`
	LineItems            []ShippingLineItem `json:"lineItems"`
}

/*
	Defines a structure for ProjectManager screen Data
*/
type PmView struct {
	PoId      string     `json:"poId"`
	PoNumber  int        `json:"poNumber"`
	PoStatus  string     `json:"poStatus"`
	ProjectId string     `json:"projectId"`
	LineItems []LineItem `json:"lineItems"`
}
