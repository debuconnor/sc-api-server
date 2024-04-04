package roomapi

import (
	"github.com/debuconnor/dbcore"
	"github.com/valyala/fasthttp"
)

func NewPlace(place Place) Website {
	return &place
}

func (place *Place) Get() {}

func (place *Place) Save() {
	dml := dbcore.NewDml()
	dml.Insert()
	dml.Into(SCHEMA_PLACE)
	dml.Value(COLUMN_ID, itoa(place.Id))
	dml.Value(COLUMN_ADMIN_ID, itoa(place.Admin.(*Admin).Id))
	dml.Value(COLUMN_PLATFORM_CODE, place.Platform.(*Platform).Code)
	dml.Value(COLUMN_NAME, place.Name)
	dml.Value(COLUMN_ADDRESS, place.Address)
	dml.Value(COLUMN_DESCRIPTION, place.Description)
	dml.Value(COLUMN_STATUS, "0") // TODO: Set status
	dml.Value(COLUMN_URL, place.Url)
	dml.Execute(db.GetDb())
}

func (place *Place) Delete() {}

func (place *Place) Update() {}

func (place *Place) Parse(string) {}

func (place *Place) Scrape() {}

func (place *Place) Retrieve() {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.Header.SetMethod(HEADER_METHOD_GET)
	req.Header.Set(HEADER_AUTHORIZATION, place.Platform.(*Platform).Session[PLATFORM_COLUMN_ACCESS_TOKEN])

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	req.SetRequestURI(place.Url)

	err := fasthttp.Do(req, resp)
	if err != nil {
		Error(err)
	}

	if resp.StatusCode() == fasthttp.StatusOK {
		roomData := decodeJson(string(resp.Body()))

		for _, roomData := range roomData[PLATFORM_COLUMN_ROOM].([]interface{}) {
			roomMap := roomData.(map[string]interface{})
			room := Room{
				Id:    int(roomMap[COLUMN_ID].(float64)),
				Place: place,
				Name:  roomMap[COLUMN_NAME].(string),
				Price: roomMap[COLUMN_PRICE].(float64),
			}
			place.Rooms = append(place.Rooms, room)
		}
	}
}
