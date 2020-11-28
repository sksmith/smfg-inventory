package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sksmith/smfg-inventory/db"
	"github.com/sksmith/smfg-inventory/inventory"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var testProducts = []inventory.Product{
	{
		Sku:       "TestOneSKU",
		Upc:       "1111111111",
		Name:      "Test One Name",
		Available: 50,
		Reserved:  0,
	},
	{
		Sku:       "TestTwoSKU",
		Upc:       "2222222222",
		Name:      "Test Two Name",
		Available: 0,
		Reserved:  5,
	},
	{
		Sku:       "TestThreeSKU",
		Upc:       "3333333333",
		Name:      "Test Three Name",
		Available: 60,
		Reserved:  0,
	},
}

var testProductionEvents = []inventory.ProductionEvent{
	{
		ID:       10,
		Sku:      "TestOneSKU",
		Quantity: 5,
		Created:  time.Date(2020, 8, 6, 10, 55, 0, 0, time.UTC),
	},
	{
		ID:       11,
		Sku:      "TestOneSKU",
		Quantity: 5,
		Created:  time.Date(2020, 11, 22, 10, 55, 0, 0, time.UTC),
	},
}

var testReservations = []inventory.Reservation{
	{
		ID:                20,
		Requester:         "ResOneRequester",
		Sku:               "TestOneSKU",
		State:             inventory.Open,
		ReservedQuantity:  0,
		RequestedQuantity: 30,
		Created:           time.Time{},
	},
}

func TestList(t *testing.T) {
	mockRepo := inventory.NewMockRepo()
	mockQueue := inventory.NewMockQueue()

	mockRepo.GetAllProductsFunc = func(ctx context.Context, limit int, offset int, tx ...db.Transaction) ([]inventory.Product, error) {
		products := make([]inventory.Product, 2)
		products[0] = testProducts[0]
		products[1] = testProducts[2]
		return products, nil
	}

	ts := httptest.NewServer(configureRouter(mockQueue, mockRepo))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/inventory/v1")
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%s", body)
}

func TestListError(t *testing.T) {
	mockRepo := inventory.NewMockRepo()
	mockQueue := inventory.NewMockQueue()

	mockRepo.GetAllProductsFunc = func(ctx context.Context, limit int, offset int, tx ...db.Transaction) ([]inventory.Product, error) {
		return nil, errors.New("some terrible error has occurred in the repo")
	}

	ts := httptest.NewServer(configureRouter(mockQueue, mockRepo))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/inventory/v1")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 500 {
		t.Errorf("Status Code got=%d want=%d", res.StatusCode, 500)
	}
}

func TestPagination(t *testing.T) {
	mockRepo := inventory.NewMockRepo()
	mockQueue := inventory.NewMockQueue()

	wantLimit := 10
	wantOffset := 50

	mockRepo.GetAllProductsFunc = func(ctx context.Context, limit int, offset int, tx ...db.Transaction) ([]inventory.Product, error) {
		if limit != wantLimit {
			t.Errorf("limit got=%d want=%d", limit, wantLimit)
		}
		if offset != wantOffset {
			t.Errorf("limit got=%d want=%d", offset, wantOffset)
		}

		return nil, nil
	}

	ts := httptest.NewServer(configureRouter(mockQueue, mockRepo))
	defer ts.Close()

	_, err := http.Get(ts.URL + fmt.Sprintf("/inventory/v1?limit=%d&offset=%d", wantLimit, wantOffset))
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreate(t *testing.T) {
	mockRepo := inventory.NewMockRepo()
	mockQueue := inventory.NewMockQueue()

	tp := testProducts[0]

	mockRepo.SaveProductFunc = func(ctx context.Context, product inventory.Product, tx ...db.Transaction) error {
		if product.Name != tp.Name {
			t.Errorf("name got=%s want=%s", product.Name, tp.Name)
		}
		if product.Sku != tp.Sku {
			t.Errorf("sku got=%s want=%s", product.Sku, tp.Sku)
		}
		if product.Upc != tp.Upc {
			t.Errorf("upc got=%s want=%s", product.Upc, tp.Upc)
		}
		if product.Reserved != 0 {
			t.Errorf("reserved quantity should be ignored on creation got=%d want=%d", product.Reserved, 0)
		}
		if product.Available != 0 {
			t.Errorf("available quantity should be ignored on creation got=%d want=%d", product.Available, 0)
		}
		return nil
	}

	ts := httptest.NewServer(configureRouter(mockQueue, mockRepo))
	defer ts.Close()

	data, err := json.Marshal(tp)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(ts.URL + "/inventory/v1", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%s", body)
}

func TestCreateProductionEvent(t *testing.T) {
	mockRepo := inventory.NewMockRepo()
	mockQueue := inventory.NewMockQueue()

	tpe := testProductionEvents[0]

	mockRepo.GetProductFunc = func(ctx context.Context, sku string, tx ...db.Transaction) (inventory.Product, error) {
		return testProducts[0], nil
	}

	mockRepo.GetProductionEventByRequestIDFunc = func(ctx context.Context, requestID string, tx ...db.Transaction) (pe inventory.ProductionEvent, err error) {
		return pe, sql.ErrNoRows
	}

	mockRepo.SaveProductionEventFunc = func(ctx context.Context, event *inventory.ProductionEvent, tx ...db.Transaction) error {
		if event.ID != 0 {
			t.Errorf("id should be ignored on creation got=%d want=%d", event.ID, 0)
		}
		if event.Sku != tpe.Sku {
			t.Errorf("sku got=%s want=%s", event.Sku, tpe.Sku)
		}
		if event.Created == tpe.Created {
			t.Errorf("event created should be set upon creation")
		}
		if event.Quantity != tpe.Quantity {
			t.Errorf("quantity got=%d want=%d", event.Quantity, tpe.Quantity)
		}
		return nil
	}

	mockRepo.SaveProductFunc = func(ctx context.Context, product inventory.Product, tx ...db.Transaction) error {
		want := tpe.Quantity + testProducts[0].Available
		if product.Available != want {
			t.Errorf("available got=%d want=%d", product.Available, want)
		}
		return nil
	}

	ts := httptest.NewServer(configureRouter(mockQueue, mockRepo))
	defer ts.Close()

	data, err := json.Marshal(tpe)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(ts.URL + fmt.Sprintf("/inventory/v1/%s/productionEvent", tpe.Sku),
		"application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 201 {
		t.Errorf("unexpected status code got=%d want=%d", res.StatusCode, 201)
	}

	resp := &inventory.ProductionEventResponse{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateProductNotFound(t *testing.T) {
	mockRepo := inventory.NewMockRepo()
	mockQueue := inventory.NewMockQueue()

	tpe := testProductionEvents[0]

	mockRepo.GetProductFunc = func(ctx context.Context, sku string, tx ...db.Transaction) (inventory.Product, error) {
		if sku != tpe.Sku {
			t.Errorf("sku got=%s want=%s", sku, tpe.Sku)
		}
		return inventory.Product{}, sql.ErrNoRows
	}

	ts := httptest.NewServer(configureRouter(mockQueue, mockRepo))
	defer ts.Close()

	data, err := json.Marshal(tpe)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(ts.URL + fmt.Sprintf("/inventory/v1/%s/productionEvent", tpe.Sku),
		"application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 404 {
		t.Errorf("unexpected StatusCode got=%d want=%d", res.StatusCode, 404)
	}
}

func TestCreateReservation(t *testing.T) {
	mockRepo := inventory.NewMockRepo()
	mockQueue := inventory.NewMockQueue()

	tr := testReservations[0]
	tp := testProducts[0]

	mockRepo.GetProductFunc = func(ctx context.Context, sku string, tx ...db.Transaction) (inventory.Product, error) {
		return tp, nil
	}

	mockRepo.SaveReservationFunc = func(ctx context.Context, r *inventory.Reservation, tx ...db.Transaction) error {
		if r.ID != 0 {
			t.Errorf("id should be ignored on creation got=%d want=%d", r.ID, 0)
		}
		if r.Requester != tr.Requester {
			t.Errorf("requester got=%s want=%s", r.Requester, tr.Requester)
		}
		if r.Sku != tr.Sku {
			t.Errorf("sku got=%s want=%s", r.Sku, tr.Sku)
		}
		if r.State != inventory.Open {
			t.Errorf("state got=%s want=%s", r.State, inventory.Open)
		}
		if r.ReservedQuantity != 0 {
			t.Errorf("reserved quantity should be ignored on creation got=%d want=%d", r.ReservedQuantity, tr.ReservedQuantity)
		}
		if r.RequestedQuantity != tr.RequestedQuantity {
			t.Errorf("requestedQuantity got=%d want=%d", r.RequestedQuantity, tr.RequestedQuantity)
		}
		if r.Created == tr.Created {
			t.Errorf("event created should be set upon creation")
		}
		return nil
	}

	mockRepo.GetSkuReservesByStateFunc =
		func(ctx context.Context, sku string, state inventory.ReserveState, limit, offset int,
			tx ...db.Transaction) ([]inventory.Reservation, error) {

			return []inventory.Reservation{tr}, nil
		}

	mockRepo.SaveProductFunc = func(ctx context.Context, product inventory.Product, tx ...db.Transaction) error {
		if product.Reserved != 0 {
			t.Errorf("reserved got=%d want=%d", product.Reserved, 0)
		}
		if product.Available != 20 {
			t.Errorf("available got=%d want=%d", product.Available, 20)
		}
		return nil
	}

	mockRepo.UpdateReservationFunc =
		func(ctx context.Context, ID uint64, state inventory.ReserveState, qty int64, txs ...db.Transaction) error {
			if ID != tr.ID {
				t.Errorf("id got=%d want=%d", ID, tr.ID)
			}
			if state != inventory.Closed {
				t.Errorf("state got=%s want=%s", state, inventory.Closed)
			}
			if qty != tr.RequestedQuantity {
				t.Errorf("reservedQuantity got=%d want=%d", qty, tr.RequestedQuantity)
			}
			return nil
		}

	sentToQueue := false
	mockQueue.SendFunc = func(body interface{}, options ...inventory.MessageOption) error {
		sentToQueue = true
		return nil
	}

	ts := httptest.NewServer(configureRouter(mockQueue, mockRepo))
	defer ts.Close()

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(ts.URL + fmt.Sprintf("/inventory/v1/%s/reservation", tr.Sku),
		"application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	if !sentToQueue {
		t.Errorf("sentToQueue got=%t want=%t", sentToQueue, true)
	}

	if res.StatusCode != 201 {
		t.Errorf("status code got=%d want=%d", res.StatusCode, 201)
	}

	body, err := ioutil.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	resp := &inventory.ReservationResponse{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		t.Fatal(err)
	}
}
