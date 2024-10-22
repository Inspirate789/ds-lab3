package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Inspirate789/ds-lab2/internal/car/delivery"
	"github.com/Inspirate789/ds-lab2/internal/models"
	"github.com/sony/gobreaker/v2"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type cars struct {
	items      []models.Car
	totalCount uint64
}

type car struct {
	item  models.Car
	found bool
}

type CarsAPI struct {
	baseURL string
	client  *http.Client
	carsCB  *gobreaker.CircuitBreaker[cars]
	carCB   *gobreaker.CircuitBreaker[car]
	logger  *slog.Logger
}

func New(baseURL string, client *http.Client, maxFails uint, logger *slog.Logger) *CarsAPI {
	logCB := func(name string, from gobreaker.State, to gobreaker.State) {
		logger.Debug("change circuit breaker state",
			slog.String("name", name),
			slog.String("from", from.String()),
			slog.String("to", to.String()),
		)
	}

	carsCB := gobreaker.NewCircuitBreaker[cars](gobreaker.Settings{
		Name:          "get_cars",
		MaxRequests:   uint32(maxFails),
		Timeout:       time.Second,
		OnStateChange: logCB,
	})

	carCB := gobreaker.NewCircuitBreaker[car](gobreaker.Settings{
		Name:          "get_car",
		MaxRequests:   uint32(maxFails),
		Timeout:       time.Second,
		OnStateChange: logCB,
	})

	return &CarsAPI{
		baseURL: baseURL,
		client:  client,
		carsCB:  carsCB,
		carCB:   carCB,
		logger:  logger,
	}
}

func (api *CarsAPI) HealthCheck(ctx context.Context) error {
	endpoint := api.baseURL + "/manage/health"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}

	return nil

}

func (api *CarsAPI) getCars(ctx context.Context, offset, limit uint64, showAll bool) (res []models.Car, totalCount uint64, err error) {
	endpoint := api.baseURL + fmt.Sprintf("/api/v1/cars?offset=%d&limit=%d&showAll=%v", offset, limit, showAll)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, 0, errors.New(string(body))
	}

	var cars delivery.CarsDTO

	err = json.Unmarshal(body, &cars)
	if err != nil {
		return nil, 0, err
	}

	return cars.ToModel(), cars.Count, nil
}

func (api *CarsAPI) GetCars(ctx context.Context, offset, limit uint64, showAll bool) ([]models.Car, uint64, error) {
	res, err := api.carsCB.Execute(func() (cars, error) {
		items, totalCount, err := api.getCars(ctx, offset, limit, showAll)
		return cars{
			items:      items,
			totalCount: totalCount,
		}, err
	})
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			return make([]models.Car, 0), 0, nil
		}

		return nil, 0, err
	}

	return res.items, res.totalCount, nil
}

func (api *CarsAPI) getCar(ctx context.Context, carUID string) (res models.Car, found bool, err error) {
	endpoint := api.baseURL + "/api/v1/cars/" + carUID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return models.Car{}, false, err
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return models.Car{}, false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Car{}, false, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return models.Car{}, false, nil
	} else if resp.StatusCode != http.StatusOK {
		return models.Car{}, false, errors.New(string(body))
	}

	var car delivery.CarDTO

	err = json.Unmarshal(body, &car)
	if err != nil {
		return models.Car{}, false, err
	}

	return car.ToModel(), true, nil
}

func (api *CarsAPI) GetCar(ctx context.Context, carUID string) (models.Car, bool, error) {
	res, err := api.carCB.Execute(func() (car, error) {
		item, found, err := api.getCar(ctx, carUID)
		return car{
			item:  item,
			found: found,
		}, err
	})
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			return models.Car{CarUID: carUID}, true, nil
		}

		return models.Car{}, false, err
	}

	return res.item, res.found, nil
}

func (api *CarsAPI) LockCar(ctx context.Context, carUID string) (res models.Car, found, success bool, err error) {
	endpoint := api.baseURL + "/api/v1/cars/" + carUID + "/lock"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return models.Car{}, false, false, err
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return models.Car{}, false, false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Car{}, false, false, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return models.Car{}, false, false, nil
	} else if resp.StatusCode == http.StatusLocked {
		return models.Car{}, true, false, nil
	} else if resp.StatusCode != http.StatusOK {
		return models.Car{}, false, false, errors.New(string(body))
	}

	var car delivery.CarDTO

	err = json.Unmarshal(body, &car)
	if err != nil {
		return models.Car{}, false, false, err
	}

	return car.ToModel(), true, true, nil
}

func (api *CarsAPI) UnlockCar(ctx context.Context, carUID string) (err error) {
	endpoint := api.baseURL + "/api/v1/cars/" + carUID + "/lock"

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}

	return nil
}
