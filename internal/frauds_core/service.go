package fraudscore

func CalculateFraudScore(req FraudScoreRequest, rules Rules) FraudScoreResponse {
	score := 0.0

	var vector [14]float32
	vector[0] = limitar(req.Transaction.Amount, rules.MaxAmount)
	vector[1] = limitar(float32(req.Transaction.Installments), float32(rules.MaxInstallments))
	vector[2] = limitar(req.Transaction.Amount/req.Customer.AvgAmount, rules.AmountVsAvgRatio)
	vector[3] = limitar(float32(req.Transaction.RequestedAt.Hour()), float32(rules.MaxHour))
	vector[4] = limitar(float32(req.Transaction.RequestedAt.Weekday()), 6)

	if req.LastTx != nil {
		vector[5] = limitar(float32(req.LastTx.Timestamp.Minute()), float32(rules.MaxMinutes))
		vector[6] = limitar(req.LastTx.KmFromCurrent, rules.MaxKm)
	} else {
		vector[5] = -1
		vector[6] = -1
	}

	vector[7] = limitar(req.Terminal.KmFromHome, rules.MaxKm)
	vector[8] = limitar(float32(req.Customer.TxCount24h), float32(rules.MaxTxCount24h))

	if req.Terminal.IsOnline {
		vector[9] = 1
	} else {
		vector[9] = 0
	}

	if req.Terminal.CardPresent {
		vector[10] = 1
	} else {
		vector[10] = 0
	}

	known := make(map[string]struct{})

	for _, m := range req.Customer.KnownMerchants {
		known[m] = struct{}{}
	}

	if _, ok := known[req.Merchant.ID]; ok {
		vector[11] = 1
	} else {
		vector[11] = 0
	}

	weight, ok := MCCRisk[req.Merchant.MCC]
	if !ok {
		weight = 0.5
	}
	vector[12] = weight

	vector[13] = limitar(req.Merchant.AvgAmount, rules.MaxMerchantAvgAmount)

	approved := score < 0.6

	return FraudScoreResponse{
		Approved:   approved,
		FraudScore: float32(score),
	}
}

func limitar(value float32, max float32) float32 {
	resultado := value / max

	if resultado < 0 {
		return 0
	}
	if resultado > 1 {
		return 1
	}

	return resultado
}
