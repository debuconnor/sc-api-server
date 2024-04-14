package roomapi

import (
	"fmt"

	"github.com/debuconnor/dbcore"
	"github.com/valyala/fasthttp"
)

func NewReservation(reservation Reservation) Website {
	return &reservation
}

func (reservation *Reservation) Get() {
	defer Recover()

	dml := dbcore.NewDml()
	dml.SelectAll()
	dml.From(SCHEMA_RESERVATION)
	dml.Where("", COLUMN_ID, dbcore.EQUAL, itoa(reservation.Id))
	queryResult := dml.Execute(db.GetDb())

	if len(queryResult) > 0 {
		reservation.Id = atoi(queryResult[0][COLUMN_ID])
		reservation.Admin = NewAdmin(Admin{
			Id: atoi(queryResult[0][COLUMN_ADMIN_ID]),
		})
		reservation.Customer = NewCustomer(Customer{
			Id: atoi(queryResult[0][COLUMN_CUSTOMER_ID]),
		})
		reservation.Room = NewRoom(Room{
			Id: atoi(queryResult[0][COLUMN_ROOM_ID]),
		})

		reservation.Date = queryResult[0][COLUMN_DATE]
		reservation.SpendTime = atoi(queryResult[0][COLUMN_SPEND_TIME])
		reservation.PersonCount = atoi(queryResult[0][COLUMN_PERSON_COUNT])
		reservation.Memo = queryResult[0][COLUMN_MEMO]
		reservation.CreatedAt = queryResult[0][COLUMN_CREATED_AT]
		reservation.UpdatedAt = queryResult[0][COLUMN_UPDATED_AT]

		dml.Clear()
		dml.SelectAll()
		dml.From(SCHEMA_PAYMENT)
		dml.Where("", COLUMN_ID, dbcore.EQUAL, queryResult[0][COLUMN_PAYMENT_ID])
		queryResult = dml.Execute(db.GetDb())

		reservation.Payment = Payment{
			Id:         atoi(queryResult[0][COLUMN_ID]),
			Admin:      NewAdmin(Admin{Id: atoi(queryResult[0][COLUMN_ADMIN_ID])}),
			Platform:   NewPlatform(Platform{Code: queryResult[0][COLUMN_PLATFORM_CODE]}),
			Customer:   NewCustomer(Customer{Id: atoi(queryResult[0][COLUMN_CUSTOMER_ID])}),
			Amount:     atof(queryResult[0][COLUMN_AMOUNT]),
			PaidAmount: atof(queryResult[0][COLUMN_PAID_AMOUNT]),
			PaidPoint:  atof(queryResult[0][COLUMN_PAID_POINT]),
			CreatedAt:  queryResult[0][COLUMN_CREATED_AT],
			UpdatedAt:  queryResult[0][COLUMN_UPDATED_AT],
		}
	} else {
		reservation.Id = 0
	}
}

func (reservation *Reservation) Save() {
	defer Recover()

	date := reservation.Date
	addDay, hour := convertMinuteToDayHour(itoa(reservation.SpendTime))
	shour := getHour(date)
	ehour := shour + hour
	if ehour > 24 {
		ehour -= 24
		addDay++
	}

	day := getDay(date) + addDay
	month := getMonth(date)

	if isLeapYear := getYear(date)%4 == 0; isLeapYear {
		MONTH_END_DAY[1] = 29
	}

	if day > MONTH_END_DAY[month-1] {
		day -= MONTH_END_DAY[month-1]
		month++
	}
	year := getYear(date)

	if month > 12 {
		month -= 12
		year++
	}

	externalReservationId := itoa(reservation.Id)
	reservation.Memo = reservation.Platform.(*Platform).Code + " " + externalReservationId
	sDate := itoa(getYear(date)) + addDatePadding(itoa(getMonth(date))) + addDatePadding(itoa(getDay(date)))
	eDate := itoa(year) + addDatePadding(itoa(month)) + addDatePadding(itoa(day))
	requestJson := fmt.Sprintf(`{"%s":"%s","%s":"%s","%s":"%s","%s":"%s","%s":"%s","%s":"%s","%s":"%s","%s":"-1","%s":"%s"}`,
		JSON_START_DATE, sDate,
		JSON_END_DATE, eDate,
		JSON_START_HOUR, itoa(shour),
		JSON_END_HOUR, itoa(ehour-1),
		JSON_NAME, reservation.Customer.(*Customer).Name,
		JSON_TEL, reservation.Customer.(*Customer).Phone,
		JSON_MEMO, reservation.Memo,
		JSON_REPEAT,
		JSON_REPEAT_END_DATE, itoa(getYear(date))+itoa(getMonth(date))+"01")

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.Header.SetMethod(HEADER_METHOD_POST)
	req.Header.Set(HEADER_CONTENT_TYPE, "application/json; charset=utf-8")
	req.Header.Set(HEADER_AUTHORIZATION, reservation.Platform.(*Platform).Session[PLATFORM_COLUMN_ACCESS_TOKEN])
	req.Header.Set(HEADER_CONTENT_LENGTH, itoa(len(requestJson)))

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	req.SetRequestURI(URI_SAVE_RESERVATION_PREFIX + itoa(reservation.Room.(*Room).Id) + URI_SAVE_RESERVATION_SUFFIX)
	req.SetBodyString(requestJson)

	err := fasthttp.Do(req, resp)
	if err != nil {
		Error(err)
	}

	loadReq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(loadReq)

	loadReq.Header.SetMethod(HEADER_METHOD_GET)
	loadReq.Header.Set(HEADER_AUTHORIZATION, reservation.Platform.(*Platform).Session[PLATFORM_COLUMN_ACCESS_TOKEN])

	loadResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(loadResp)
	loadReq.SetRequestURI(URI_RETRIEVE_CALENDAR + URI_QUERY_MONTH + addDatePadding(itoa(month)) + URI_QUERY_AND + URI_QUERY_PRODUCT_ID + itoa(reservation.Room.(*Room).Id) + URI_QUERY_AND + URI_QUERY_YEAR + itoa(year))

	err = fasthttp.Do(loadReq, loadResp)
	if err != nil {
		Error(err)
	}

	for _, day := range decodeJsonArray(string(loadResp.Body())) {
		if day[PLATFORM_COLUMN_DATE] == sDate {
			for _, schedule := range day[PLATFORM_COLUMN_EXTERNAL_SCHEUDLE].([]interface{}) {
				if schedule.(map[string]interface{})[PLATFORM_COLUMN_PHONE] == reservation.Customer.(*Customer).Phone {
					reservation.Id = int(schedule.(map[string]interface{})[COLUMN_ID].(float64))
					break
				}
			}
			break
		}
	}

	if resp.StatusCode() == fasthttp.StatusOK {
		reservationId := reservation.Id
		externalPlatformCode := reservation.Platform.(*Platform).Code
		reservation.Get()

		if reservation.Id == 0 {
			dml := dbcore.NewDml()
			dml.Insert()
			dml.Into(SCHEMA_PAYMENT)
			dml.Value(COLUMN_ADMIN_ID, itoa(reservation.Admin.(*Admin).Id))
			dml.Value(COLUMN_PLATFORM_CODE, externalPlatformCode)
			dml.Value(COLUMN_CUSTOMER_ID, itoa(reservation.Customer.(*Customer).Id))
			dml.Value(COLUMN_RESERVATION_ID, itoa(reservationId))
			dml.Value(COLUMN_AMOUNT, ftoa(reservation.Payment.Amount))
			dml.Value(COLUMN_PAID_AMOUNT, ftoa(reservation.Payment.PaidAmount))
			dml.Value(COLUMN_PAID_POINT, ftoa(reservation.Payment.PaidPoint))
			dml.Value(COLUMN_STATUS, "0") // TODO: Set status
			dml.Value(COLUMN_CREATED_AT, reservation.Payment.CreatedAt)
			dml.Value(COLUMN_UPDATED_AT, reservation.Payment.UpdatedAt)
			dml.Execute(db.GetDb())

			dml.Clear()
			dml.SelectColumn(COLUMN_ID)
			dml.From(SCHEMA_PAYMENT)
			dml.Where("", COLUMN_ADMIN_ID, dbcore.EQUAL, itoa(reservation.Admin.(*Admin).Id))
			dml.Where(dbcore.AND, COLUMN_PLATFORM_CODE, dbcore.EQUAL, externalPlatformCode)
			dml.Where(dbcore.AND, COLUMN_CUSTOMER_ID, dbcore.EQUAL, itoa(reservation.Customer.(*Customer).Id))
			dml.Where(dbcore.AND, COLUMN_RESERVATION_ID, dbcore.EQUAL, itoa(reservationId))
			queryResult := dml.Execute(db.GetDb())
			reservation.Payment.Id = atoi(queryResult[0][COLUMN_ID])

			dml.Clear()
			dml.Insert()
			dml.Into(SCHEMA_RESERVATION)
			dml.Value(COLUMN_ID, itoa(reservationId))
			dml.Value(COLUMN_ADMIN_ID, itoa(reservation.Admin.(*Admin).Id))
			dml.Value(COLUMN_CUSTOMER_ID, itoa(reservation.Customer.(*Customer).Id))
			dml.Value(COLUMN_ROOM_ID, itoa(reservation.Room.(*Room).Id))
			dml.Value(COLUMN_PAYMENT_ID, itoa(reservation.Payment.Id))
			dml.Value(COLUMN_STATUS, "0") // TODO: Set status
			dml.Value(COLUMN_DATE, reservation.Date)
			dml.Value(COLUMN_SPEND_TIME, itoa(reservation.SpendTime))
			dml.Value(COLUMN_PERSON_COUNT, itoa(reservation.PersonCount))
			dml.Value(COLUMN_MEMO, reservation.Memo)
			dml.Value(COLUMN_URL, reservation.Url)
			dml.Value(COLUMN_CREATED_AT, reservation.CreatedAt)
			dml.Value(COLUMN_UPDATED_AT, reservation.UpdatedAt)
			dml.Execute(db.GetDb())
			reservation.Id = reservationId
		} else {
			reservation.Update()
		}
	}
}

func (reservation *Reservation) Delete() {
	defer Recover()

	reservation.Get()

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.Header.SetMethod(HEADER_METHOD_DELETE)
	req.Header.Set(HEADER_AUTHORIZATION, reservation.Platform.(*Platform).Session[PLATFORM_COLUMN_ACCESS_TOKEN])

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	req.SetRequestURI(URI_SAVE_RESERVATION_PREFIX + itoa(reservation.Room.(*Room).Id) + URI_SAVE_RESERVATION_SUFFIX + itoa(reservation.Id))

	err := fasthttp.Do(req, resp)
	if err != nil {
		Error(err)
	} else {
		dml := dbcore.NewDml()
		dml.Delete()
		dml.From(SCHEMA_RESERVATION)
		dml.Where("", COLUMN_ID, dbcore.EQUAL, itoa(reservation.Id))
		dml.Execute(db.GetDb())

		dml.Clear()
		dml.Delete()
		dml.From(SCHEMA_PAYMENT)
		dml.Where("", COLUMN_ID, dbcore.EQUAL, itoa(reservation.Payment.Id))
		dml.Execute(db.GetDb())

		reservation = NewReservation(Reservation{}).(*Reservation)
		_ = reservation
	}
}

func (reservation *Reservation) Update() {
	defer Recover()

	dml := dbcore.NewDml()
	dml.Update(SCHEMA_RESERVATION)
	dml.Set(COLUMN_DATE, reservation.Date)
	dml.Set(COLUMN_SPEND_TIME, itoa(reservation.SpendTime))
	dml.Set(COLUMN_PERSON_COUNT, itoa(reservation.PersonCount))
	dml.Set(COLUMN_MEMO, reservation.Memo)
	dml.Set(COLUMN_UPDATED_AT, getNow())
	dml.Where("", COLUMN_ID, dbcore.EQUAL, itoa(reservation.Id))
	dml.Execute(db.GetDb())
}

func (reservation *Reservation) Scrape() {}

func (reservation *Reservation) Retrieve() {}
