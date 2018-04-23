package clustering

import (
	"fmt"

	"github.com/amadeusitgroup/redis-operator/pkg/redis"
)

// SelectMastersToReplace used to replace currentMaster with new Nodes
func SelectMastersToReplace(currentOldMaster, currentNewMasters, newNoneNodes redis.Nodes, nbMaster, nbMasterToReplace int32) (selectedMasters redis.Nodes, newSelectedMasters redis.Nodes, err error) {
	newSelectedMasters = redis.Nodes{}
	if len(currentNewMasters) == int(nbMaster) {
		return currentNewMasters, newSelectedMasters, nil
	}

	selectedMasters = append(selectedMasters, currentNewMasters...)
	nbMasterReplaced := int32(0)
	for _, newNode := range newNoneNodes {
		if nbMasterReplaced == nbMasterToReplace {
			break
		}
		nbMasterReplaced++
		selectedMasters = append(selectedMasters, newNode)
		newSelectedMasters = append(newSelectedMasters, newNode)
	}
	nbRemainingOldMaster := int(nbMaster) - len(selectedMasters)
	if nbRemainingOldMaster > len(currentOldMaster) {
		nbRemainingOldMaster = len(currentOldMaster)
	}
	currentOldMasterSorted := currentOldMaster.SortNodes()
	selectedMasters = append(selectedMasters, currentOldMasterSorted[:nbRemainingOldMaster]...)
	if nbMasterReplaced != nbMasterToReplace {
		return selectedMasters, newSelectedMasters, fmt.Errorf("unable to select enough new node for master replacement, wanted:%d, current:%d", nbMasterToReplace, nbMasterReplaced)
	}

	if len(selectedMasters) != int(nbMaster) {
		return selectedMasters, newSelectedMasters, fmt.Errorf("unable to select enough Masters wanted:%d, current:%d", len(selectedMasters), nbMaster)
	}

	return selectedMasters, newSelectedMasters, err
}
