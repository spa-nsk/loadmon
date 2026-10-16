package dorado

import (
	"context"
	"fmt"
	"strconv"
)

type realtimePower struct {
	PowerConsumptionValue string `json:"powerConsumptionValue"`
}

// GetRealtimePowerConsumption возвращает текущую потребляемую мощность
// системы, Вт (GET /get_realtime_power_consumption, справочник REST
// Interface Reference: Power Information in Real Time).
func (d *Device) GetRealtimePowerConsumption(ctx context.Context) (float64, error) {
	req, err := d.newRequest(ctx, "GET", "/get_realtime_power_consumption", nil)
	if err != nil {
		return 0, fmt.Errorf(ErrCreateRequest+": %w", err)
	}
	var out realtimePower
	if err = d.requestWithRetry(req, &out, DefaultHTTPRetryCount); err != nil {
		return 0, fmt.Errorf(ErrRequestWithRetry+": %w", err)
	}
	return strconv.ParseFloat(out.PowerConsumptionValue, 64)
}
