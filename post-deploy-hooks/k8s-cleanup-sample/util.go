package main

// diffSlices returns the elements in slice1 that are not in slice2.
func diffSlices(slice1, slice2 []string) []string {
	var diff []string
	slice2Map := make(map[string]bool)

	for _, val := range slice2 {
		slice2Map[val] = true
	}

	for _, val := range slice1 {
		// If they're not in slice2 add them to the diff slice.
		if !slice2Map[val] {
			diff = append(diff, val)
		}
	}

	return diff
}
