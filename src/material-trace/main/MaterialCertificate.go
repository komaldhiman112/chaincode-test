package main

type MaterialCertificate struct {
	ObjectType    string       `json:"docType"`
	TrackingId    string       `json:"trackingId"`
	MaterialGroup string       `json:"materialGroup"`
	Data          []MtrDetails `json:"data"`
	ReferencedBy  []string     `json:"referencedBy"`
}

type MtrDetails struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
