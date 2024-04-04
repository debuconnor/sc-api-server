package roomapi

import (
	"github.com/debuconnor/dbcore"
	"github.com/valyala/fasthttp"
)

func NewPlace(place Place) Website {
	return &place
}

func (place *Place) Get() {
	defer Recover()

	dml := dbcore.NewDml()
	dml.SelectAll()
	dml.From(SCHEMA_PLACE)
	dml.Where("", COLUMN_ID, dbcore.EQUAL, itoa(place.Id))
	queryResult := dml.Execute(db.GetDb())

	if len(queryResult) > 0 {
		place.Admin = NewAdmin(Admin{
			Id: atoi(queryResult[0][COLUMN_ADMIN_ID]),
		})

		place.Platform = NewPlatform(Platform{
			Code: queryResult[0][COLUMN_PLATFORM_CODE],
		})

		place.Name = queryResult[0][COLUMN_NAME]
		place.Address = queryResult[0][COLUMN_ADDRESS]
		place.Description = queryResult[0][COLUMN_DESCRIPTION]
		place.Url = queryResult[0][COLUMN_URL]
		place.Status = Status{
			Id: atoi(queryResult[0][COLUMN_STATUS]),
		}
	} else {
		place.Id = 0

	}
}

func (place *Place) Save() {
	defer Recover()
	placeId := place.Id
	place.Get()

	if place.Id == 0 {
		dml := dbcore.NewDml()
		dml.Insert()
		dml.Into(SCHEMA_PLACE)
		dml.Value(COLUMN_ID, itoa(placeId))
		dml.Value(COLUMN_ADMIN_ID, itoa(place.Admin.(*Admin).Id))
		dml.Value(COLUMN_PLATFORM_CODE, place.Platform.(*Platform).Code)
		dml.Value(COLUMN_NAME, place.Name)
		dml.Value(COLUMN_ADDRESS, place.Address)
		dml.Value(COLUMN_DESCRIPTION, place.Description)
		dml.Value(COLUMN_STATUS, "0") // TODO: Set status
		dml.Value(COLUMN_URL, place.Url)
		dml.Execute(db.GetDb())
		place.Id = placeId
	} else {
		place.Update()
	}
}

func (place *Place) Delete() {}

func (place *Place) Update() {
	dml := dbcore.NewDml()
	dml.Update(SCHEMA_PLACE)
	dml.Set(COLUMN_NAME, place.Name)
	dml.Set(COLUMN_ADDRESS, place.Address)
	dml.Set(COLUMN_DESCRIPTION, place.Description)
	dml.Set(COLUMN_STATUS, itoa(place.Status.Id))
	dml.Set(COLUMN_URL, place.Url)
	dml.Where("", COLUMN_ID, dbcore.EQUAL, itoa(place.Id))
	dml.Execute(db.GetDb())
}

func (place *Place) Parse(string) {}

func (place *Place) Scrape() {}

func (place *Place) Retrieve() {
	defer Recover()

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
