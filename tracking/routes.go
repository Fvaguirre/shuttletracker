package tracking

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"

	"fmt"
	"io/ioutil"

	"gopkg.in/mgo.v2/bson"
)

// Coord represents a single lat/lng point used to draw routes
type Coord struct {
	Lat float64 `json:"lat" bson:"lat"`
	Lng float64 `json:"lng" bson:"lng"`
}

// Route represents a set of coordinates to draw a path on our tracking map
type Route struct {
	ID          string     `json:"id"             bson:"_id,omitempty"`
	Name        string     `json:"name"           bson:"name"`
	Description string     `json:"description"    bson:"description"`
	StartTime   string     `json:"startTime"      bson:"startTime"`
	EndTime     string     `json:"endTime"        bson:"endTime"`
	Enabled     bool       `json:"enabled,string" bson:"enabled"`
	Color       string     `json:"color"          bson:"color"`
	Width       int        `json:"width,string"   bson:"width"`
	Coords      []Coord    `json:"coords"         bson:"coords"`
	Duration    []Velocity `json:"duration"      bson:"duration"`
	Created     time.Time  `json:"created"        bson:"created"`
	Updated     time.Time  `json:"updated"        bson:"updated"`
}

// Stop indicates where a tracked object is scheduled to arrive
type Stop struct {
	ID          string `json:"id"             bson:"id"`
	Name        string `json:"name"           bson:"name"`
	Description string `json:"description"    bson:"description"`
	// position on map
	Lat     float64 `json:"lat,string"     bson:"lat"`
	Lng     float64 `json:"lng,string"     bson:"lng"`
	Address string  `json:"address"        bson:"address"`

	StartTime string `json:"startTime"      bson:"startTime"`
	EndTime   string `json:"endTime"        bson:"endTime"`
	Enabled   bool   `json:"enabled,string" bson:"enabled"`
	RouteID   string `json:"routeId"        bson:"routeId"`
}

type MapPoint struct {
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
}
type MapResponsePoint struct {
	Location      MapPoint `json:"location"`
	OriginalIndex int      `json:"originalIndex,omitempty"`
	PlaceID       string   `json:"placeId"`
}
type MapResponse struct {
	SnappedPoints []MapResponsePoint
}

type MapDistanceMatrixDuration struct {
	Value int    `json:"value"`
	Text  string `json:"text"`
}

type MapDistanceMatrixDistance struct {
	Value int    `json:"value"`
	Text  string `json:"text"`
}

type MapDistanceMatrixElement struct {
	Status   string                    `json:"status"`
	Duration MapDistanceMatrixDuration `json:"duration"`
	Distance MapDistanceMatrixDistance `json:"distance"`
}

type MapDistanceMatrixElements struct {
	Elements []MapDistanceMatrixElement `json:"elements"`
}
type MapDistanceMatrixResponse struct {
	Status               string                      `json:"status"`
	OriginAddresses      []string                    `json:"origin_addresses"`
	DestinationAddresses []string                    `json:"destination_addresses"`
	Rows                 []MapDistanceMatrixElements `json:"rows"`
}

type Velocity struct {
	Start    MapPoint `json:"origin"`
	End      MapPoint `json:"destination"`
	Distance float32  `json:"distance"`
	Duration float32  `json:"duration"`
}

// RoutesHandler finds all of the routes in the database
func (App *App) RoutesHandler(w http.ResponseWriter, r *http.Request) {
	// Find all routes in database
	var routes []Route
	err := App.Routes.Find(bson.M{}).All(&routes)
	// Handle query errors
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// Send each route to client as JSON
	WriteJSON(w, routes)
}

// StopsHandler finds all of the route stops in the database
func (App *App) StopsHandler(w http.ResponseWriter, r *http.Request) {
	// Find all stops in database
	var stops []Stop
	err := App.Stops.Find(bson.M{}).All(&stops)
	// Handle query errors
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// Send each stop to client as JSON
	WriteJSON(w, stops)
}

func Interpolate(coords []Coord, key string) []Coord {
	// make request
	prefix := "https://roads.googleapis.com/v1/snapToRoads?"
	var buffer bytes.Buffer
	buffer.WriteString(prefix)
	buffer.WriteString("path=")
	for i, coord := range coords {
		buffer.WriteString(strconv.FormatFloat(coord.Lat, 'f', 10, 64))
		buffer.WriteString(",")
		buffer.WriteString(strconv.FormatFloat(coord.Lng, 'f', 10, 64))
		if i < len(coords)-1 {
			buffer.WriteString("|")
		}
	}
	buffer.WriteString("&interpolate=true&key=")
	buffer.WriteString(key)
	resp, err := http.Get(buffer.String())
	if err != nil {
		fmt.Errorf("Error Not valid response from Google API")
		return nil
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	mapResponse := MapResponse{}
	json.Unmarshal(body, &mapResponse)
	result := []Coord{}
	for _, location := range mapResponse.SnappedPoints {
		currentLocation := Coord{
			Lat: float64(location.Location.Latitude),
			Lng: float64(location.Location.Longitude),
		}
		result = append(result, currentLocation)
	}
	return result
}

func GoogleVelocityCompute(from Coord, to Coord, key string) Velocity {
	prefix := "https://maps.googleapis.com/maps/api/distancematrix/json?units=imperial&"
	var buffer bytes.Buffer
	buffer.WriteString(prefix)
	origin := fmt.Sprintf("origins=%f,%f", from.Lat, from.Lng)
	destination := fmt.Sprintf("destinations=%f,%f", to.Lat, to.Lng)
	buffer.WriteString(origin + "&" + destination + "&key=" + key)
	fmt.Println(buffer.String())
	resp, err := http.Get(buffer.String())
	if err != nil {
		fmt.Errorf("Error Not valid response from Google API")
		return Velocity{}
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	mapResponse := MapDistanceMatrixResponse{}
	json.Unmarshal(body, &mapResponse)
	fmt.Println(mapResponse)
	result := Velocity{
		Start: MapPoint{
			Latitude:  float32(from.Lat),
			Longitude: float32(from.Lng),
		},
		End: MapPoint{
			Latitude:  float32(to.Lat),
			Longitude: float32(to.Lng),
		},
		Distance: float32(mapResponse.Rows[0].Elements[0].Distance.Value),
		Duration: float32(mapResponse.Rows[0].Elements[0].Duration.Value),
	}
	fmt.Println(result)
	return result
}

// compute distance between two coordinates and return a value
func ComputeDistance(c1 Coord, c2 Coord) float32 {
	return float32(math.Sqrt(math.Pow(c1.Lat-c2.Lat, 2) + math.Pow(c1.Lng-c2.Lng, 2)))
}

// Compute the velocity for each segment of the coordinates
func ComputeVelocity(coords []Coord, key string, threshold int) []Velocity {
	result := []Velocity{}
	// only compute the distance greater than some theshold distance and assume all in between has the same velocity
	prev := 0
	index := 1
	// This part could be improved by rewriting with asynchronized call
	for index = 1; index < len(coords); index++ {
		if index%threshold == 0 {
			v := GoogleVelocityCompute(coords[prev], coords[index], key)
			for inner := prev + 1; inner <= index; inner++ {
				result = append(result, Velocity{
					Distance: v.Distance / float32(index-prev),
					Duration: v.Duration / float32(index-prev),
					Start:    MapPoint{Latitude: float32(coords[inner-1].Lat), Longitude: float32(coords[inner-1].Lng)},
					End:      MapPoint{Latitude: float32(coords[inner].Lat), Longitude: float32(coords[inner].Lng)},
				})
			}
			prev = index
		}
	}
	// compute the last segment
	result = append(result, GoogleVelocityCompute(coords[prev], coords[0], key))
	return result
}

// RoutesCreateHandler adds a new route to the database
func (App *App) RoutesCreateHandler(w http.ResponseWriter, r *http.Request) {
	// Create a new route object using request fields
	var routeData map[string]string
	var coordsData []map[string]float64
	// Decode route details
	err := json.NewDecoder(r.Body).Decode(&routeData)
	// Error handling
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// Unmarshal route coordinates
	err = json.Unmarshal([]byte(routeData["coords"]), &coordsData)
	// Error handling
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// Create a Coord from each set of input coordinates
	coords := []Coord{}
	for _, c := range coordsData {
		coord := Coord{c["lat"], c["lng"]}
		coords = append(coords, coord)
	}
	// Here do the interpolation
	coords = Interpolate(coords, App.Config.GoogleMapAPIKey)
	velo := ComputeVelocity(coords, App.Config.GoogleMapAPIKey, App.Config.GoogleMapMinDistance)
	// now we get the velocity for each segment ( this should be stored in database, just store it inside route for god sake)

	fmt.Printf("Size of coordinates = %d", len(coords))
	// Type conversions
	enabled, _ := strconv.ParseBool(routeData["enabled"])
	width, _ := strconv.Atoi(routeData["width"])
	currentTime := time.Now()
	// Create a new route
	route := Route{
		string(bson.NewObjectId()),
		routeData["name"],
		routeData["description"],
		routeData["startTime"],
		routeData["endTime"],
		enabled,
		routeData["color"],
		width,
		coords,
		velo,
		currentTime,
		currentTime}
	// Store new route under routes collection
	err = App.Routes.Insert(&route)
	// Error handling
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//Deletes route from database
func (App *App) RoutesDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("deleting", vars["id"])
	err := App.Routes.Remove(bson.M{"_id": bson.ObjectIdHex(vars["id"])})
	// Error handling
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// StopsCreateHandler adds a new route stop to the database
func (App *App) StopsCreateHandler(w http.ResponseWriter, r *http.Request) {
	// Create a new stop object using request fields
	stop := Stop{}
	err := json.NewDecoder(r.Body).Decode(&stop)
	// Error handling
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// Store new stop under stops collection
	err = App.Stops.Insert(&stop)
	// Error handling
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (App *App) StopsDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("deleting", vars["id"])
	err := App.Stops.Remove(bson.M{"name": vars["id"]})
	// Error handling
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
