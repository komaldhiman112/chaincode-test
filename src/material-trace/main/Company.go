package main

type Company struct {
	CompanyId     string  `json:"companyId"`
	CompanyType   string  `json:"companyType"`
	Name          string  `json:"name"`
	StreetAddress string  `json:"streetAddress"`
	City          string  `json:"city"`
	Zipcode       string  `json:"zipcode"`
	State         string  `json:"state"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
}
