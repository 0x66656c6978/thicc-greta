package indexer

import (
	"log"
	"time"
)

var (
	// maps index types to indices
	itemIndex = make(map[string]PoeNinjaItemIndex)
	// list of index types
	indexTypes = [18]string{"BaseType", "Oil", "Incubator", "Scarab", "Fossil", "Resonator", "Essence", "DivinationCard", "Prophecy", "SkillGem", "Map", "UniqueMap", "UniqueJewel", "UniqueFlask", "UniqueWeapon", "UniqueArmour", "UniqueAccessory", "Beast"}
	// the logger for everything indexer related
	appLogger = log.Logger{}
	// callers to setLeague should set this
	league = ""
)

func setLeague(newLeague string) {
	league = newLeague
}

// Run the indexer
func Run(league string, offersChannel chan Offer) {
	appLogger.SetPrefix("indexer: ")
	appLogger.SetOutput(log.Writer())
	recentChangeID, errRecentChangeID := getRecentChangeID()
	if errRecentChangeID != nil {
		appLogger.Fatal(errRecentChangeID)
		return
	}
	appLogger.Printf("Initial change id is \"%v\"", recentChangeID)
	go func() {
		for {
			for _, indexType := range indexTypes {
				index, errRequestIndex := requestItemIndex(league, indexType)
				if errRequestIndex != nil {
					appLogger.Fatal(errRequestIndex)
					break
				}
				itemIndex[indexType] = index
				time.Sleep(300 * time.Millisecond)
			}
			appLogger.Printf("Updated item indices")
			time.Sleep(5 * time.Minute)
		}
	}()
	subscribeItems(recentChangeID, offersChannel)
}
