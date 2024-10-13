package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Inspirate789/ds-lab2/internal/models"
	"github.com/Inspirate789/ds-lab2/internal/rental/delivery"
	"io"
	"log/slog"
	"net/http"
)

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type RentalsAPI struct {
	baseURL string
	client  *http.Client
	logger  *slog.Logger
}

func New(baseURL string, client *http.Client, logger *slog.Logger) *RentalsAPI {
	return &RentalsAPI{
		baseURL: baseURL,
		client:  client,
		logger:  logger,
	}
}

func (api *RentalsAPI) HealthCheck(ctx context.Context) error {
	endpoint := api.baseURL + "/manage/health"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
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

func (api *RentalsAPI) GetUserRentals(ctx context.Context, username string, offset, limit uint64) ([]models.Rental, uint64, error) {
	endpoint := api.baseURL + fmt.Sprintf("/api/v1/rentals?offset=%d&limit=%d", offset, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("X-User-Name", username)

	resp, err := http.DefaultClient.Do(req)
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

	var rentals delivery.RentalsDTO

	err = json.Unmarshal(body, &rentals)
	if err != nil {
		return nil, 0, err
	}

	model, err := rentals.ToModel()
	if err != nil {
		return nil, 0, err
	}

	return model, rentals.Count, nil
}

func (api *RentalsAPI) GetUserRental(ctx context.Context, rentalUID, username string) (res models.Rental, found, permitted bool, err error) {
	endpoint := api.baseURL + "/api/v1/rentals/" + rentalUID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return models.Rental{}, false, false, err
	}

	req.Header.Set("X-User-Name", username)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return models.Rental{}, false, false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Rental{}, false, false, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return models.Rental{}, false, false, nil
	} else if resp.StatusCode == http.StatusForbidden {
		return models.Rental{}, true, false, nil
	} else if resp.StatusCode != http.StatusOK {
		return models.Rental{}, false, false, errors.New(string(body))
	}

	var rental delivery.RentalDTO

	err = json.Unmarshal(body, &rental)
	if err != nil {
		return models.Rental{}, true, true, err
	}

	model, err := rental.ToModel()
	if err != nil {
		return models.Rental{}, true, true, err
	}

	return model, true, true, nil
}

func (api *RentalsAPI) CreateRental(ctx context.Context, properties models.RentalProperties) (models.Rental, error) {
	endpoint := api.baseURL + "/api/v1/rentals"
	dto := delivery.NewRentalPropertiesDTO(properties)

	body, err := json.Marshal(dto)
	if err != nil {
		return models.Rental{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return models.Rental{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return models.Rental{}, err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return models.Rental{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return models.Rental{}, errors.New(string(body))
	}

	var rental delivery.RentalDTO

	err = json.Unmarshal(body, &rental)
	if err != nil {
		return models.Rental{}, err
	}

	model, err := rental.ToModel()
	if err != nil {
		return models.Rental{}, err
	}

	return model, nil
}

func (api *RentalsAPI) SetRentalStatus(ctx context.Context, rentalUID string, status models.RentalStatus) (found bool, err error) {
	endpoint := api.baseURL + "/api/v1/rentals/" + rentalUID + "/status"

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewBufferString(fmt.Sprint(status)))
	if err != nil {
		return false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	} else if resp.StatusCode != http.StatusOK {
		return false, errors.New(string(body))
	}

	return true, nil
}
