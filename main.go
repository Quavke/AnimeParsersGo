package main

import (
	"fmt"

	"github.com/Quavke/AnimeParsersGo/parsers"
)

func main() {
	title := "Связанные"
	AniboomParser := parsers.NewAniboomParser("")
	result, err := AniboomParser.Search(title)
	if err != nil {
		fmt.Printf("FastSearch вернул ошибку: %v", err)
		return
	}
	for _, v := range result {
		fmt.Printf("%+v", *v)
	}
}
