package main

import (
	"fmt"

	"github.com/Quavke/AnimeParsersGo/parsers"
)

func main() {
	title := "Поднятие уровня в одиночку"
	AniboomParser := parsers.NewAniboomParser("animego.org")
	result, err := AniboomParser.FastSearch(title)
	if err != nil {
		fmt.Printf("FastSearch вернул ошибку: %v", err)
		return
	}
	for _, v := range result {
		fmt.Printf("FastSearch: %+v\n\n", *v)
	}
	search, err := AniboomParser.Search(title)
	if err != nil {
		fmt.Printf("Search вернул ошибку: %v", err)
		return
	}
	for _, v := range search {
		fmt.Printf("Search: %+v\n\n", *v)
	}
	res := *result[0]
	episodes_info_res, err := AniboomParser.EpisodesInfo(res.Link)
	if err != nil {
		fmt.Printf("EpisodesInfo вернул ошибку: %v", err)
		return
	}
	episodes_info_first := *episodes_info_res[0]
	fmt.Printf("EpisodesInfo: %+v\n\n", episodes_info_first)

	anime_info_res, err := AniboomParser.AnimeInfo(res.Link)
	if err != nil {
		fmt.Printf("AnimeInfo вернул ошибку: %v", err)
		return
	}
	anime_info := *anime_info_res
	fmt.Printf("AnimeInfo: %+v\n\n", anime_info)

	fmt.Printf("Translations: %+v\n\n", anime_info.Translations)

	trans_info, err := AniboomParser.GetTranslationsInfo(anime_info.AnimegoID)
	if err != nil {
		fmt.Printf("GetTranslationsInfo вернул ошибку: %v", err)
		return
	}
	for _, v := range trans_info {
		fmt.Printf("GetTranslationsInfo: %+v\n\n", *v)
	}

	err = AniboomParser.GetAsFile(anime_info.AnimegoID, anime_info.Translations[0].TranslationID, "output", 1)
	if err != nil {
		fmt.Printf("GetAsFile вернул ошибку: %v", err)
		return
	}

	str, err := AniboomParser.GetMPDPlaylist(anime_info.AnimegoID, anime_info.Translations[0].TranslationID, 1)
	if err != nil {
		fmt.Printf("GetMPDPlaylist вернул ошибку: %v", err)
		return
	}
	fmt.Printf("GetMPDPlaylist: %s\n\n", str)
}
