package indexer

import (
	"log"
	"os"
	"os/signal"
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
func Run(theLeague string, offersChannel chan Offer) {
	setLeague(theLeague)
	appLogger.SetPrefix("indexer: ")
	appLogger.SetOutput(log.Writer())
	recentChangeID, errRecentChangeID := getRecentChangeID()
	if errRecentChangeID != nil {
		appLogger.Fatal(errRecentChangeID)
		return
	}
	appLogger.Printf("Initial change id is \"%v\"", recentChangeID)
	go func() {
		interruptChannel := make(chan os.Signal)
		signal.Notify(interruptChannel, os.Interrupt)
		for {
			for _, indexType := range indexTypes {
				index, errRequestIndex := requestItemIndex(theLeague, indexType)
				if errRequestIndex != nil {
					appLogger.Fatal(errRequestIndex)
					break
				}
				itemIndex[indexType] = index
				select {
				case <-interruptChannel:
					return
				case <-time.After(300 * time.Millisecond):
					continue
				}
			}
			appLogger.Printf("Updated item indices")

			select {
			case <-interruptChannel:
				return
			case <-time.After(5 * time.Minute):
				continue
			}

		}
	}()
	subscribeItems(recentChangeID, offersChannel)
}
