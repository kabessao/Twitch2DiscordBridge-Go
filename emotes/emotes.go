package emotes

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

var emotesCache map[string]string

func GetEmotes() map[string]string {
	return emotesCache
}

func UpdateCache(fileName string) (err error) {
	var newCache = map[string]string{}
	data, err := readFromFille(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating %s: %v", fileName, err)
		return
	}

	for _, item := range data {
		if len(item) < 2 {
			continue
		}

		newCache[strings.TrimSpace(item[0])] = strings.TrimSpace(item[1])
	}

	emotesCache = newCache

	println("Emotes loaded\n")

	return
}

func readFromFille(filePath string) (value [][]string, err error) {

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		return
	}

	defer file.Close()

	csvReader := csv.NewReader(file)

	value, err = csvReader.ReadAll()

	if err != nil {
		fmt.Printf("Error: %s\n\n", err)
		return
	}

	return
}
