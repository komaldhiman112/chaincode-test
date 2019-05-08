package main

/*
IOT specific functions and structures
*/

import (
	"fmt"
	hav "haversine"
)

func distanceFromProjectSite(destination Company, iotInput IotProperty) (mi float64) {
	fmt.Println("lastJourney: ", destination)
	destCoord := hav.Coord{Lat: destination.Latitude, Lon: destination.Longitude}
	iotCoord := hav.Coord{Lat: iotInput.Latitude, Lon: iotInput.Longitude}
	fmt.Println("destCoord", destCoord)
	fmt.Println("iotCoord", iotCoord)
	// km := 0.00
	mi, _ = hav.Distance(iotCoord, destCoord)
	return mi
}

// func distanceFromDestination(lastJourney Journey, iotInput IotProperty) (mi float64) {
// 	fmt.Println("lastJourney", lastJourney)
// 	destCoord := hav.Coord{Lat: lastJourney.Destination.Latitude, Lon: lastJourney.Destination.Longitude}
// 	iotCoord := hav.Coord{Lat: iotInput.Latitude, Lon: iotInput.Longitude}
// 	fmt.Println("destCoord", destCoord)
// 	fmt.Println("iotCoord", iotCoord)
// 	km := 0.00
// 	mi, km = hav.Distance(iotCoord, destCoord)
// 	println("can use km here if want ", km)
// 	return mi
// }
