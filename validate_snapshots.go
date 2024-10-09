package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type State int

const (
	noMX State = iota
	wantMX
	inMX
)

type Snapshot struct {
	ID            int
	State         State
	ReqTs         int
	NbrResps      int
	Waiting       []bool
	ChannelStates map[int][]string
}

func main() {
	snapshotDir := "./snapshots"
	snapshots, err := readSnapshots(snapshotDir)
	if err != nil {
		fmt.Println("Error reading snapshots:", err)
		return
	}

	// Group snapshots by snapshot ID
	snapshotGroups := groupSnapshotsByID(snapshots)

	// Validate each group of snapshots
	for snapshotID, group := range snapshotGroups {
		// fmt.Printf("Validating Snapshot ID: %d\n", snapshotID)
		if validateSnapshotGroup(group) {
			// fmt.Printf("Snapshot ID %d is consistent.\n", snapshotID)
		} else {
			fmt.Printf("Snapshot ID %d is inconsistent.\n", snapshotID)
		}
	}
}

func readSnapshots(dir string) ([]Snapshot, error) {
	var snapshots []Snapshot

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(dir, file.Name())
			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				return nil, err
			}

			var snapshot Snapshot
			err = json.Unmarshal(data, &snapshot)
			if err != nil {
				return nil, err
			}

			snapshots = append(snapshots, snapshot)
		}
	}

	return snapshots, nil
}

func groupSnapshotsByID(snapshots []Snapshot) map[int][]Snapshot {
	groups := make(map[int][]Snapshot)
	for _, snapshot := range snapshots {
		groups[snapshot.ID] = append(groups[snapshot.ID], snapshot)
	}
	return groups
}

func validateSnapshotGroup(group []Snapshot) bool {
	if !validateInvariant1(group) {
		return false
	}
	if !validateInvariant2(group) {
		return false
	}
	if !validateInvariant3(group) {
		return false
	}
	return true
}

// Invariant 1: At most one process in the SC
func validateInvariant1(group []Snapshot) bool {
	countInMX := 0
	for _, snapshot := range group {
		if snapshot.State == inMX {
			countInMX++
		}
	}
	return countInMX <= 1
}

// Invariant 2: If all processes are in "noMX" state, then all `waiting` flags must be false and there should be no messages
func validateInvariant2(group []Snapshot) bool {
	allNoMX := true
	for _, snapshot := range group {
		if snapshot.State != noMX {
			allNoMX = false
			break
		}
	}

	if allNoMX {
		for _, snapshot := range group {
			for _, waiting := range snapshot.Waiting {
				if waiting {
					return false
				}
			}
			for _, messages := range snapshot.ChannelStates {
				if len(messages) > 0 {
					return false
				}
			}
		}
	}

	return true
}

// Invariant 3: If a process is marked as `waiting` in another process, then the other process is in the SC or wants the SC
func validateInvariant3(group []Snapshot) bool {
	for _, snapshot := range group {
		for j, waiting := range snapshot.Waiting {
			if waiting {
				if group[j].State != inMX && group[j].State != wantMX {
					return false
				}
			}
		}
	}
	return true
}