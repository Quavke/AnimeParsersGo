package main

import (
	"encoding/json"
	"fmt"

	"github.com/Quavke/AnimeParsersGo/parsers"
)

func main() {
	title := "Поднятие уровня в одиночку"

	shikimori_test(title)

	// headers := models.Headers{
	// 	"User-Agent":       "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0",
	// 	"Accept":           "application/json, text/plain, */*",
	// 	"X-Requested-With": "XMLHttpRequest",
	// }

	// params := models.Params{
	// 	"search": title,
	// }

	// t.TestURL("https://shikimori.one/animes/autocomplete/v2", "GET", params, headers)
}

func shikimori_test(title string) {
	ShikimoriParser := parsers.NewShikimoriParser("")
	// result, err := ShikimoriParser.Search(title)
	// if err != nil {
	// 	fmt.Printf("Search вернул ошибку: %v", err)
	// 	return
	// }
	// for _, v := range result {
	// 	fmt.Printf("Search: %+v\n\n", *v)
	// }
	// first_res := *result[0]
	// anime_info, err := ShikimoriParser.AnimeInfo(first_res.Link)
	// if err != nil {
	// 	fmt.Printf("AnimeInfo вернул ошибку: %v", err)
	// 	return
	// }

	// fmt.Printf("AnimeInfo: %+v\n\n", *anime_info)

	info, err := ShikimoriParser.AdditionalAnimeInfo("https://shikimori.one/animes/60303-shinjiteita-nakama-tachi-ni-dungeon-okuchi-de-korosarekaketa-ga-gift-mugen-gacha-de-level-9999-no-nakama-tachi-wo-te-ni-irete-moto-party-member-to-sekai-ni-fukushuu-zamaa-shimasu")
	if err != nil {
		fmt.Printf("AdditionalAnimeInfo вернул ошибку: %v", err)
		return
	}
	bts, err := json.Marshal(info)
	if err != nil {
		fmt.Printf("Marshal вернул ошибку: %v", err)
		return
	}
	fmt.Println(string(bts))
}

func aniboom_test() {
	title := "Поднятие уровня в одиночку"
	AniboomParser := parsers.NewAniboomParser("")
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
}
