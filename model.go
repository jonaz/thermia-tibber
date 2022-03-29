package main

import (
	"sort"
	"sync"
	"time"
)

type Response struct {
	Data struct {
		Viewer struct {
			Homes []struct {
				CurrentSubscription CurrentSubscription `json:"currentSubscription"`
			} `json:"homes"`
		} `json:"viewer"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type CurrentSubscription struct {
	PriceRating *PriceRating `json:"priceRating"`
}

type PriceRating struct {
	Hourly Hourly `json:"hourly"`
}

type Hourly struct {
	Entries []Price `json:"entries"`
}

type Price struct {
	Level    string    `json:"level"`
	Total    float64   `json:"total"`
	Energy   float64   `json:"energy"`
	Tax      float64   `json:"tax"`
	Currency string    `json:"currency"`
	Time     time.Time `json:"time"`
}

type Prices struct {
	prices              map[time.Time]Price
	mutex               sync.RWMutex
	cheapestChargeStart time.Time
	cheapestHour        time.Time
	lastCalculated      time.Time
}

func NewPrices() *Prices {
	return &Prices{
		prices: make(map[time.Time]Price),
	}
}

func (p *Prices) Add(price Price) {
	p.mutex.Lock()
	p.prices[price.Time] = price
	p.mutex.Unlock()
}

func (p *Prices) SetCheapestChargeStart(t time.Time) {
	p.mutex.Lock()
	p.cheapestChargeStart = t
	p.mutex.Unlock()
}

func (p *Prices) CheapestChargeStart() time.Time {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.cheapestChargeStart
}
func (p *Prices) SetCheapestHour(t time.Time) {
	p.mutex.Lock()
	p.cheapestHour = t
	p.mutex.Unlock()
}

func (p *Prices) CheapestHour() time.Time {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.cheapestHour
}

func (p *Prices) Last() time.Time {
	ss := make([]time.Time, len(p.prices))
	i := 0
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for t := range p.prices {
		ss[i] = t
		i++
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].After(ss[j])
	})

	return ss[0]
}

func (p *Prices) calculateCheapestHour(from, to time.Time) time.Time {
	ss := []Price{}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for _, t := range p.prices {
		if t.Time.Before(from) || t.Time.After(to) {
			continue
		}
		ss = append(ss, t)
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Total < ss[j].Total
	})

	return ss[0].Time
}

func (p *Prices) Current() *Price {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for t, price := range p.prices {
		if inTimeSpan(t, t.Add(60*time.Minute), time.Now()) {
			return &price
		}
	}

	return nil
}

func (p *Prices) HasTomorrowPricesYet() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for t := range p.prices {
		if time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour).Equal(t.Truncate(24 * time.Hour)) {
			return true
		}
	}

	return false
}

func (p *Prices) SetLastCalculated(t time.Time) {
	p.mutex.Lock()
	p.lastCalculated = t
	p.mutex.Unlock()
}
func (p *Prices) LastCalculated() time.Time {
	p.mutex.Lock()
	t := p.lastCalculated
	p.mutex.Unlock()
	return t
}

func (p *Prices) ClearOld() {
	p.mutex.Lock()
	for t := range p.prices {
		if time.Now().Add(time.Hour * -24).After(t) {
			delete(p.prices, t)
		}
	}
	p.mutex.Unlock()
}
func (p *Prices) SortedByTime() []Price {
	p.mutex.Lock()
	prices := make([]Price, len(p.prices))
	i := 0
	for _, v := range p.prices {
		prices[i] = v
		i++
	}
	p.mutex.Unlock()

	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Time.Before(prices[j].Time)
	})

	return prices
}

func isSameHourAndDay(t1, t2 time.Time) bool {
	return t1.Truncate(1 * time.Hour).Equal(t2.Truncate(1 * time.Hour))
}

func inTimeSpan(start, end, check time.Time) bool {
	if start.Before(end) {
		return !check.Before(start) && !check.After(end)
	}
	if start.Equal(end) {
		return check.Equal(start)
	}
	return !start.After(check) || !end.Before(check)
}

func (p *Prices) calculateLevel(t time.Time, total float64) (diff float64, lvl int) {
	tot := 0.0
	totCnt := 0.0
	p.mutex.RLock()
	for _, v := range p.prices {
		if t.Add(time.Hour*-24).After(v.Time) || v.Time.After(t) {
			continue
		}
		tot += v.Total
		totCnt += 1.0
	}
	p.mutex.RUnlock()
	average := tot / totCnt

	switch {
	case total >= average*1.20:
		lvl = 3 // HIGH
	case total <= average*0.90:
		lvl = 1 // LOW
	default:
		lvl = 2 // NORMAL
	}

	diff = total / average

	// fmt.Println("avg: ", average)
	// fmt.Println("price: ", total)
	// fmt.Println("diff: ", diff)
	// fmt.Println("lvl: ", lvl)
	return
}
