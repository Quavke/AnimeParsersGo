package main

import (
	"fmt"

	"github.com/Quavke/AnimeParsersGo/parsers"
)

func main() {
	title := "Врата Штейна"
	AniboomParser := parsers.NewAniboomParser("")
	result, err := AniboomParser.FastSearch(title)
	if err != nil {
		fmt.Printf("FastSearch вернул ошибку: %v", err)
		return
	}
	for _, v := range *result {
		fmt.Println("------------------------------------")
		fmt.Printf("AnimegoID: %s\n", v.AnimegoID)
		fmt.Printf("Link: %s\n", v.Link)
		fmt.Printf("OtherTitle: %s\n", v.OtherTitle)
		fmt.Printf("Title: %s\n", v.Title)
		fmt.Printf("Type: %s\n", v.Type)
		fmt.Printf("Year: %s\n", v.Year)
		fmt.Println("------------------------------------")
	}
}
