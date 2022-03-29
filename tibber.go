package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func fetchAndCalculate(config *Config) {
	prices, err := fetchPrices(config.Token)
	if err != nil {
		logrus.Error(err)
		curr := pricesStore.Current()
		if curr != nil { // update state if we have cached hour and API call fails.
			logrus.Errorf("using cached value for Price: %#v", curr)
			// node.UpdateState("1", pricesStore.State())
			// TODO DO THE UPDATE HERE
		}
		return
	}

	now := time.Now()
	for _, p := range prices {
		if now.Add(time.Hour * -24).After(p.Time) {
			continue
		}
		pricesStore.Add(p)
	}

	pricesStore.ClearOld()

	if pricesStore.HasTomorrowPricesYet() && !pricesStore.LastCalculated().Truncate(24*time.Hour).Equal(now.Truncate(24*time.Hour)) {
		cheapestHour := pricesStore.calculateCheapestHour(
			now.Truncate(24*time.Hour).Add(time.Hour*18),                  // today plus 18 is 19:00 CET
			now.Add(24*time.Hour).Truncate(24*time.Hour).Add(time.Hour*8), // tomorrow 00 plus 8 hours is 09:00 CET
		)
		pricesStore.SetCheapestHour(cheapestHour)
		logrus.Infof("cheapestHour is: %s", cheapestHour)

		pricesStore.SetLastCalculated(time.Now())
	}

	// node.UpdateState("1", pricesStore.State())
	// TODO DO THE UPDATE HERE
}

func fetchPrices(token string) ([]Price, error) {
	query := `
{
  viewer {
    homes {
      currentSubscription{
        priceRating{
          hourly {
            entries {
              total
              time
              level
              energy
              difference
              tax
            }
          }
        }
      }
    }
  }
}`

	out, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	reqBody := fmt.Sprintf(`{"query":%s}`, out)

	req, err := http.NewRequest("POST", "https://api.tibber.com/v1-beta/gql", strings.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error fetching from tibber api status: %d", resp.StatusCode)
	}

	response := &Response{}
	err = json.NewDecoder(resp.Body).Decode(response)
	if err != nil {
		return nil, err
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("tibber error: %s", response.Errors[0].Message)
	}
	if len(response.Data.Viewer.Homes) == 0 {
		return nil, fmt.Errorf("no homes found in response")
	}
	return response.Data.Viewer.Homes[0].CurrentSubscription.PriceRating.Hourly.Entries, nil
}
