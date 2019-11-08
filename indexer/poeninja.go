package indexer

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// PoeNinjaItem represetns an item from the poe.ninja API
type PoeNinjaItem struct {
	ID           int       `json:"id"`
	ItemLevel    int       `json:"levelRequired"`
	Links        int       `json:"links"`
	Corrupted    bool      `json:"corrupted"`
	Name         string    `json:"name"`
	BaseType     string    `json:"baseType"`
	ItemType     string    `json:"itemType"`
	ItemClass    int       `json:"itemClass"`
	ChaosValue   float32   `json:"chaosValue"`
	ExaltedValue float32   `json:"exaltedValue"`
	Count        int       `json:"count"`
	LCSparkline  Sparkline `json:"lowConfidenceSparkline"`
	Sparkline    Sparkline `json:"sparkline"`
	Variant      string    `json:"variant"`
	MapTier      int       `json:"mapTier"`
	GemLevel     int       `json:"gemLevel"`
	GemQuality   int       `json:"gemQuality"`
}

// Sparkline represents the price change of an item over a time interval
type Sparkline struct {
	TotalChange float32   `json:"totalChange"`
	Data        []float32 `json:"data"`
}

// PoeNinjaItemResponse represents a JSON response from the poe.ninja API
type PoeNinjaItemResponse struct {
	Entries []PoeNinjaItem `json:"lines"`
}

// PoeNinjaItemIndex groups items by their name
type PoeNinjaItemIndex map[string]PoeNinjaItem

// To begin receiving newly updated items immediately, we need to get a recent change id. poe.ninja
// can provide us with that.
func getRecentChangeID() (string, error) {
	resp, err := http.Get("https://poe.ninja/api/Data/GetStats")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var stats struct {
		// There are a few more fields in the response, but we only care about this.
		NextChangeID string `json:"next_change_id"`
	}
	if err := json.Unmarshal(body, &stats); err != nil {
		return "", err
	}

	return stats.NextChangeID, nil
}

// Retreive all item base-types and their prices from poe.ninja
// Also filter these based on configuration settings
func requestItemIndex(league string, indexType string) (PoeNinjaItemIndex, error) {
	resp, err := http.Get("https://poe.ninja/api/data/itemoverview?league=" + league + "&type=" + indexType)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	res := PoeNinjaItemResponse{}

	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}

	index := make(PoeNinjaItemIndex)
	for _, item := range res.Entries {
		comparisonItemLevel := 0
		if indexType == "BaseType" {
			comparisonItemLevel = item.ItemLevel
		}
		indexKey := makeItemIndexKey(item.Name, item.Variant, comparisonItemLevel, item.Links, item.MapTier, item.GemLevel, item.GemQuality, item.Corrupted)
		index[indexKey] = item
	}

	return index, nil
}
