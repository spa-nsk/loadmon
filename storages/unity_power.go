package storages

import "fmt"

// GetUnityPowerWatts возвращает текущую потребляемую мощность Unity, Вт:
// сумма currentPower по DPE и DAE (одна REST-сессия).
func GetUnityPowerWatts(address string) (float64, error) {
	session, err := openUnitySession(address)
	if err != nil {
		return 0, err
	}
	defer session.CloseSession()

	var sum float64
	var n int
	dpe, err := session.GetDPE()
	if err != nil {
		return 0, fmt.Errorf("dpe: %w", err)
	}
	for _, e := range dpe.Entries {
		sum += float64(e.Content.CurrentPower)
		n++
	}
	dae, err := session.GetDAE()
	if err != nil {
		return 0, fmt.Errorf("dae: %w", err)
	}
	for _, e := range dae.Entries {
		sum += float64(e.Content.CurrentPower)
		n++
	}
	if n == 0 {
		return 0, fmt.Errorf("DPE/DAE не найдены")
	}
	return sum, nil
}
