package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Inspirate789/ds-lab2/internal/models"
	"github.com/Inspirate789/ds-lab2/internal/payment/delivery"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type PaymentsAPI struct {
	baseURL string
	client  *http.Client
	logger  *slog.Logger
}

func New(baseURL string, client *http.Client, logger *slog.Logger) *PaymentsAPI {
	return &PaymentsAPI{
		baseURL: baseURL,
		client:  client,
		logger:  logger,
	}
}

func (api *PaymentsAPI) HealthCheck(ctx context.Context) error {
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

func (api *PaymentsAPI) CreatePayment(ctx context.Context, price uint64) (res models.Payment, err error) {
	endpoint := api.baseURL + "/api/v1/payments?price=" + strconv.FormatUint(price, 10)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return models.Payment{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return models.Payment{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Payment{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return models.Payment{}, errors.New(string(body))
	}

	var payment delivery.PaymentDTO

	err = json.Unmarshal(body, &payment)
	if err != nil {
		return models.Payment{}, err
	}

	return payment.ToModel(), nil
}

func (api *PaymentsAPI) SetPaymentStatus(ctx context.Context, paymentUID string, status models.PaymentStatus) (found bool, err error) {
	endpoint := api.baseURL + "/api/v1/payments/" + paymentUID + "/status"

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

func (api *PaymentsAPI) GetPayment(ctx context.Context, paymentUID string) (res models.Payment, found bool, err error) {
	endpoint := api.baseURL + "/api/v1/payments/" + paymentUID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return models.Payment{}, false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return models.Payment{}, false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Payment{}, false, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return models.Payment{}, false, nil
	} else if resp.StatusCode != http.StatusOK {
		return models.Payment{}, false, errors.New(string(body))
	}

	var payment delivery.PaymentDTO

	err = json.Unmarshal(body, &payment)
	if err != nil {
		return models.Payment{}, false, err
	}

	return payment.ToModel(), true, nil
}
