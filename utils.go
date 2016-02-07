// Coping

package main

func Contains(needle string, haystack *[]string) bool {
	for _, v := range *haystack {
		if v == needle {
			return true
		}
	}

	return false
}
