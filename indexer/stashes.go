package indexer

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/0x66656c6978/poe-go/api"
)

// OfferStash is a wrapper around the api.Stash from the poe-go/api
type OfferStash struct {
	AccountName       string `json:"accountName"`
	LastCharacterName string `json:"lastCharacterName"`
	ID                string `json:"id"`
	Label             string `json:"stash"`
	Type              string `json:"stashType"`
	IsPublic          bool   `json:"public"`
}

// Offer is an offer
type Offer struct {
	PoeNinjaItem PoeNinjaItem
	OfferedItem  api.Item
	Stash        OfferStash
}

func getNumAbyssalSockets(item api.Item) int {
	numAbyssalSockets := 0
	for _, socket := range item.Sockets {
		if socket.Attribute == "A" {
			numAbyssalSockets++
		}
	}
	return numAbyssalSockets
}

func getVariant(item api.Item) string {
	variant := ""
	if item.IsShaper {
		variant = "Shaper"
	}
	if item.IsElder {
		variant = "Elder"
	}
	return variant
}

func getNumLinks(item api.Item) int {
	numLinks := 0
	countByGroupID := make(map[int]int)
	for i := 0; i < len(item.Sockets); i++ {
		socket := item.Sockets[i]
		countByGroupID[socket.GroupId]++
		if countByGroupID[socket.GroupId] > numLinks {
			numLinks = countByGroupID[socket.GroupId]
		}
	}
	if item.FrameType == api.UniqueItemFrameType && numLinks >= 5 {
		return numLinks
	}
	return 0
}

func getName(item api.Item) string {
	name := item.Name
	if item.Name == "" {
		name = item.Type
	} else {
		name += " " + item.Type
	}
	return name
}

func getPropertyValue(item api.Item, propertyName string) (string, error) {
	for _, property := range item.Properties {
		if property.Name == propertyName {
			return property.Values[0].([]interface{})[0].(string), nil
		}
	}
	return "", errors.New("Not found")
}

func getPropertyValueAsFloat32(item api.Item, propertyName string) (float32, error) {
	strVal, err := getPropertyValue(item, propertyName)
	if err != nil {
		return 0, err
	}
	floatVal, convErr := strconv.ParseFloat(strVal, 32)
	if convErr != nil {
		return 0, convErr
	}
	return float32(floatVal), nil
}

func getPropertyValueAsInt(item api.Item, propertyName string) (int, error) {
	strVal, err := getPropertyValue(item, propertyName)
	if err != nil {
		return 0, err
	}
	intVal, convErr := strconv.Atoi(strVal)
	if convErr != nil {
		return 0, convErr
	}
	return intVal, nil
}

func isInCategory(item api.Item, category string, subCategory string) bool {
	if subCategory == "" {
		return item.Extended.Category == category
	}
	if item.Extended.Category == category {
		for _, otherSubCategory := range item.Extended.SubCategories {
			if otherSubCategory == subCategory {
				return true
			}
		}
	}
	return false
}

func getAndNormalizeGemQuality(item api.Item) (int, error) {
	comparisonGemQuality := 0
	comparisonGemQualityStr, err := getPropertyValue(item, "Quality")
	if err != nil {
		return 0, err
	}
	// remove leading + and trailing % sign
	comparisonGemQualityStr2 := comparisonGemQualityStr[1 : len(comparisonGemQualityStr)-1]
	comparisonGemQuality, err = strconv.Atoi(comparisonGemQualityStr2)
	if err != nil {
		return 0, err
	}
	if comparisonGemQuality < 20 {
		comparisonGemQuality = 0
	}
	return comparisonGemQuality, nil
}

func getAndNormalizeGemLevel(item api.Item) (int, error) {
	comparisonGemLevel, err := getPropertyValueAsInt(item, "Level")
	if err != nil {
		return 0, err
	}
	if comparisonGemLevel < 20 {
		comparisonGemLevel = 1
	}
	return comparisonGemLevel, nil
}

func getIndexKeyByStashItem(item api.Item) string {
	comparisonItemLevel := 0
	comparisonGemLevel, _ := getAndNormalizeGemLevel(item)
	comparisonGemQuality, _ := getAndNormalizeGemQuality(item)

	// this edge-case should be fixed inside of the functions
	if item.Extended.Category == "gems" {
		if comparisonGemLevel == 0 && comparisonGemQuality == 20 {
			comparisonGemLevel = 1
		}
	} else {
		comparisonGemLevel = 0
		comparisonGemQuality = 0
	}

	mapTier, _ := getPropertyValueAsInt(item, "Map Tier")
	variant := getVariant(item)
	numLinks := getNumLinks(item)
	numAbyssalSockets := getNumAbyssalSockets(item)

	if numLinks < 5 {
		numLinks = 0
	}
	if numAbyssalSockets == 2 {
		variant = "2 Jewels"
	}
	if item.Name == "Mark of the Elder" || item.Name == "Mark of the Shaper" {
		variant = ""
	}

	// item level only applies to equippable items
	switch item.Extended.Category {
	case "armour":
		fallthrough
	case "weapons":
		fallthrough
	case "jewels":
		fallthrough
	case "accessories":
		if item.FrameType != api.UniqueItemFrameType {
			// this excludes standard jewels since their item level is irrelevant
			comparisonItemLevel = item.ItemLevel
			if comparisonItemLevel > 86 {
				comparisonItemLevel = 86
			}
		}
		break
	}

	var name string
	if item.FrameType == api.UniqueItemFrameType {
		name = item.Name
	} else {
		name = item.Type
		switch item.Extended.Category {
		case "armour":
			fallthrough
		case "weapons":
			fallthrough
		case "jewels":
			fallthrough
		case "accessories":
			if item.FrameType == api.MagicItemFrameType {
				hasPrefix := false
				hasSuffix := strings.Contains(name, "of")
				if len(item.ExplicitMods) == 2 || (len(item.ExplicitMods) == 1 && !hasSuffix) {
					hasPrefix = true
				}
				if hasPrefix {
					pieces := strings.Split(name, " ")
					name = strings.Join(pieces[1:], " ")
				}
				if hasSuffix {
					pieces := strings.Split(name, " of ")
					name = pieces[0]
				}
			}
		}
	}

	name = strings.Replace(name, "Synthesised ", "", 1)
	name = strings.Replace(name, "Superior ", "", 1)
	return makeItemIndexKey(
		name,
		variant,
		comparisonItemLevel,
		numLinks,
		mapTier,
		comparisonGemLevel,
		comparisonGemQuality,
		item.IsCorrupted,
	)
}

func getDefaultItemIndexName(item api.Item) string {
	category := item.Extended.Category
	switch category {

	case "maps":
		if isInCategory(item, "maps", "scarab") {
			return "Scarab"
		}
		return "Map"

	case "incubator":
		return "Incubator"

	case "jewels":
		fallthrough
	case "armour":
		fallthrough
	case "accessories":
		fallthrough
	case "weapons":
		return "BaseType"

	case "monsters":
		return "Beast"

	}
	return ""
}

func getUniqueItemIndexName(item api.Item) string {
	switch item.Extended.Category {
	case "maps":
		return "UniqueMap"
	case "armour":
		return "UniqueArmour"
	case "weapons":
		return "UniqueWeapon"
	case "flasks":
		return "UniqueFlask"
	case "jewels":
		return "UniqueJewel"
	case "accessories":
		return "UniqueAccessory"
	}
	return ""
}

func getCurrencyItemIndexName(item api.Item) string {
	if isInCategory(item, "currency", "fossil") {
		return "Fossil"
	}
	if isInCategory(item, "currency", "resonator") {
		return "Resonator"
	}
	if isInCategory(item, "currency", "") {
		if strings.Contains(item.Type, "Oil") {
			return "Oil"
		}
		if strings.Contains(item.Type, "Essence") {
			return "Essence"
		}
	}
	return ""
}

func getGemItemIndexName(item api.Item) string {
	return "SkillGem"
}

func getDivCardItemIndexName(item api.Item) string {
	return "DivinationCard"
}

func getProphecyItemIndexName(item api.Item) string {
	return "Prophecy"
}

func getItemIndexNameByItem(item api.Item) string {
	switch item.FrameType {

	case api.GemFrameType:
		return getGemItemIndexName(item)

	case api.CurrencyFrameType:
		return getCurrencyItemIndexName(item)

	case api.DivinationCardFrameType:
		return getDivCardItemIndexName(item)

	case api.ProphecyFrameType:
		return getProphecyItemIndexName(item)

	case api.NormalItemFrameType:
		fallthrough
	case api.MagicItemFrameType:
		fallthrough
	case api.RareItemFrameType:
		return getDefaultItemIndexName(item)

	case api.UniqueItemFrameType:
		return getUniqueItemIndexName(item)

	}
	return ""
}

func makeItemIndexKey(name, variant string, itemLevel, numLinks, mapTier, gemLevel, gemQuality int, isCorrupted bool) string {
	return fmt.Sprintf("%v.%v.%v.%v.%v.%v.%v.%v", name, variant, itemLevel, numLinks, mapTier, gemLevel, gemQuality, isCorrupted)
}

func findItemInIndex(item api.Item) (*PoeNinjaItem, error) {
	indexKey := getIndexKeyByStashItem(item)
	indexName := getItemIndexNameByItem(item)

	index := itemIndex[indexName]
	if len(index) == 0 {
		return nil, nil
	}

	poeNinjaItem := index[indexKey]
	if poeNinjaItem.ID == 0 {
		if indexName != "SkillGem" &&
			(indexName == "BaseType" && item.ItemLevel >= 82) &&
			item.IsCorrupted == false {
			return nil, nil
		}
	}
	if poeNinjaItem.ID == 0 {
		return nil, errors.New("Item \"" + indexKey + "\" not found in index \"" + indexName + "\"")
	}
	return &poeNinjaItem, nil
}

func findSimilarKey(searchVal string, index PoeNinjaItemIndex) string {
	for key := range index {
		if strings.Contains(key, searchVal) {
			return key
		}
	}
	return ""
}

func processStash(activeLeague string, stash *api.Stash, broadcastChannel chan Offer) {

	if len(stash.Items) == 0 {
		return
	}
	for _, indexType := range indexTypes {
		if itemIndex[indexType] == nil {
			return
		}
	}

	league := stash.Items[0].League
	if league != activeLeague {
		return
	}

	for _, item := range stash.Items {
		poeNinjaItem, err := findItemInIndex(item)
		if err != nil {
			continue // index not found for item
		}
		if poeNinjaItem == nil {
			continue // item variant not found in index
		}

		broadcastChannel <- Offer{
			PoeNinjaItem: *poeNinjaItem,
			OfferedItem:  item,
			Stash: OfferStash{
				AccountName:       stash.AccountName,
				LastCharacterName: stash.LastCharacterName,
				ID:                stash.Id,
				Label:             stash.Label,
				Type:              stash.Type,
				IsPublic:          stash.IsPublic,
			},
		}
	}
}

func closeSubscriptionOnInterrupt(subscription *api.PublicStashTabSubscription) {
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)
	<-ch
	subscription.Close()
	appLogger.Printf("Item subscription closed")
}

func subscribeItems(recentChangeID string, offersChannel chan Offer) {
	subscription := api.OpenPublicStashTabSubscription(recentChangeID)
	go closeSubscriptionOnInterrupt(subscription)
	for result := range subscription.Channel {
		if result.Error != nil {
			appLogger.Fatal(result.Error)
			continue
		}
		for _, stash := range result.PublicStashTabs.Stashes {
			processStash(league, &stash, offersChannel)
		}
	}
}
