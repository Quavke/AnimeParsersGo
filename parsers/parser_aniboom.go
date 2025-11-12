package parsers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	perrors "github.com/Quavke/AnimeParsersGo/errors"
)

type Client struct {
	httpClient *http.Client
}

type AniboomParser struct {
	dmn     string
	Client  *Client
	Context context.Context
}

func NewAniboomParser(mirror string) *AniboomParser {
	var dmn string
	if mirror != "" {
		dmn = mirror
	} else {
		dmn = "animego.me"
	}
	client := &Client{
		&http.Client{Timeout: 10 * time.Second},
	}
	return &AniboomParser{
		dmn:     dmn,
		Client:  client,
		Context: context.Background(),
	}
}

type FastSearchResult struct {
	Title      string `json:"title"`
	Year       string `json:"year"`
	OtherTitle string `json:"other_title"`
	Type       string `json:"type"`
	Link       string `json:"link"`
	AnimegoID  string `json:"animego_id"`
}

type JsonResponse struct {
	Status  string `json:"status"`
	Content string `json:"content"`
	Message string `json:"message,omitempty"`
}

type EpisodeInfo struct {
	Num    string `json:"num"`
	Title  string `json:"title"`
	Date   string `json:"date"`
	Status string `json:"status"`
}

type Translation struct {
	Name          string `json:"name"`
	TranslationID string `json:"translation_id"`
}

type OtherAnimeInfo struct {
	AgeRests       string   `json:"age_restrictions"`
	ReleaseDate    string   `json:"release_date"`
	MainCharacters []string `json:"main_characters"`
	Duration       string   `json:"duration"`
	OriginalSource string   `json:"original_source"`
	MPAARating     string   `json:"mpaa_rating"`
	Season         string   `json:"season"`
	OriginalRanobe string   `json:"ranobe"`
	OriginalManga  string   `json:"manga"`
	Studio         string   `json:"studio"`
}
type SearchResult struct {
	Title        string            `json:"title"`
	OtherTitle   []string          `json:"other_title"`
	Status       string            `json:"status"`
	Type         string            `json:"type"`
	Genres       []string          `json:"genres"`
	Description  string            `json:"description"`
	Episodes     string            `json:"episodes"`
	EpisodesInfo []*EpisodeInfo    `json:"episodes_info"`
	Translations []*Translation    `json:"translations"`
	PosterURL    string            `json:"poster_url"`
	Trailer      string            `json:"trailer"`
	Screenshots  []string          `json:"screenshots"`
	OtherInfo    *OtherAnimeInfo   `json:"other_info"`
	Link         string            `json:"link"`
	AnimegoID    string            `json:"animego_id"`
	Unparsed     map[string]string `json:"unparsed"`
}

/*
Быстрый поиск через animego.me

:title: Название аниме

Возвращает срез ссылок на FastSearchResult
*/
func (ab *AniboomParser) FastSearch(title string) ([]*FastSearchResult, error) {
	params := url.Values{}

	params.Set("type", "small")
	params.Set("q", title)
	domain := fmt.Sprintf("https://%s/", ab.dmn)
	URL := fmt.Sprintf("%ssearch/all?%s", domain, params.Encode())
	request, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : FastSearch : не смог создать request. Ошибка: %v", err)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	}

	request.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	request.Header.Set("Referer", domain)

	var resp *http.Response
	for attempt := 1; attempt <= 50; attempt++ {
		resp, err = ab.Client.httpClient.Do(request)
		if err != nil {
			error_message := fmt.Sprintf("Aniboom parser error : FastSearch : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			continue
		} else if resp.StatusCode != http.StatusOK {
			error_message := fmt.Sprintf("Aniboom parser error : FastSearch : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			resp.Body.Close()
			continue
		} else {
			break
		}
	}

	if err != nil || resp.StatusCode != http.StatusOK {
		error_message := fmt.Sprintf("Aniboom parser error : FastSearch : http клиент не смог выполнить запрос. Ошибка: %v\n", err)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	}
	defer resp.Body.Close()

	var json_response JsonResponse

	if err := json.NewDecoder(resp.Body).Decode(&json_response); err != nil {
		return nil, perrors.NewJsonDecodeFailureError(fmt.Sprintf("Aniboom parser error : FastSearch : ошибка декодирования json: %v", err))
	}

	if json_response.Status != "success" {
		return nil, perrors.NewServiceError(fmt.Sprintf(
			"Aniboom parser error : FastSearch : сервер вернул статус отличный от success: %q, сообщение: %q для названия: %q",
			json_response.Status, json_response.Message, title,
		))
	}

	htmlContent := html.UnescapeString(json_response.Content)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : FastSearch : goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	}
	res := make([]*FastSearchResult, 0)
	var items *goquery.Selection
	items = doc.Find("div.result-search-anime").Find("div.result-search-item")
	if items.Length() == 0 {
		warn_message := "Aniboom parser error : FastSearch : в контейнере result-search-anime не найдено ни одного элемента div.result-search-item"
		log.Println(warn_message)
		items = doc.Find("div.result-search-item")
		if items.Length() == 0 {
			error_message := "Aniboom parser error : FastSearch : в html ответа не найдено ни одного элемента div.result-search-item"
			log.Println(error_message)
			return nil, perrors.NewNoResultsError(error_message)
		}
	}

	items.Each(func(i int, s *goquery.Selection) {
		c_data := FastSearchResult{}
		c_data.Title = strings.TrimSpace(s.Find("h5").First().Text())
		c_data.Year = strings.TrimSpace(s.Find("span.anime-year").First().Text())
		c_data.OtherTitle = s.Find("div.text-truncate").First().Text()
		c_data.Type = s.Find("a[href*=\"anime/type\"]").First().Text()
		link := s.Find("h5 a").First()
		var rawLink string
		if link.Length() > 0 {
			href, exists := link.Attr("href")
			if exists {
				rawLink = href
			}
		}
		c_data.Link = fmt.Sprintf("https://%s", ab.dmn) + rawLink
		fullLink := c_data.Link
		lastDashIndex := strings.LastIndex(fullLink, "-")
		if lastDashIndex != -1 && lastDashIndex < len(fullLink)-1 {
			c_data.AnimegoID = fullLink[lastDashIndex+1:]
		} else {
			c_data.AnimegoID = ""
		}
		res = append(res, &c_data)
	})
	return res, nil
}

// Возвращает данные по эпизодам.
//
// :link: ссылка на страницу с данными (прим: https://animego.me/anime/volchica-i-pryanosti-torgovec-vstrechaet-mudruyu-volchicu-2546)
//
// Возвращает отсортированный по номеру серии срез ссылок на EpisodeInfo
func (ab *AniboomParser) EpisodesInfo(link string) ([]*EpisodeInfo, error) {
	episodes_info := make([]*EpisodeInfo, 0)

	params := url.Values{}
	referer := fmt.Sprintf("https://%s/search/all?q=anime", ab.dmn)
	params.Set("type", "episodeSchedule")
	params.Set("episodeNumber", "99999")

	URL := link + "?" + params.Encode()
	request, err := http.NewRequestWithContext(ab.Context, "GET", URL, nil)
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : EpisodesInfo : не смог создать request. Ошибка: %v", err)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	}
	request.Header.Set("Referer", referer)
	request.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")

	var resp *http.Response
	for attempt := 1; attempt <= 50; attempt++ {
		resp, err = ab.Client.httpClient.Do(request)
		if err != nil {
			error_message := fmt.Sprintf("Aniboom parser error : EpisodesInfo : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			continue
		} else if resp.StatusCode != http.StatusOK {
			error_message := fmt.Sprintf("Aniboom parser error : EpisodesInfo : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			resp.Body.Close()
			continue
		} else {
			break
		}
	}
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : EpisodesInfo : http клиент не смог выполнить запрос. Ошибка: %v", err)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		error_message := fmt.Sprintf("Aniboom parser error : EpisodesInfo : Сервер не вернул ожидаемый код 200. Код: %d\n", resp.StatusCode)
		log.Println(error_message)
		return nil, perrors.NewNoResultsError(error_message)
	}

	var json_response JsonResponse

	if err := json.NewDecoder(resp.Body).Decode(&json_response); err != nil {
		return nil, perrors.NewJsonDecodeFailureError(fmt.Sprintf("Aniboom parser error : EpisodesInfo : ошибка декодирования json: %v", err))
	}

	if json_response.Status != "success" {
		return nil, perrors.NewServiceError(fmt.Sprintf(
			"Aniboom parser error : EpisodesInfo : сервер вернул статус отличный от success: %q, сообщение: %q для ссылки: %q",
			json_response.Status, json_response.Message, link,
		))
	}
	htmlContent := html.UnescapeString(json_response.Content)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : EpisodesInfo : goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	}

	doc.Find("div.row.m-0").Each(func(i int, s *goquery.Selection) {
		episode_info := EpisodeInfo{}
		items := s.Find("div")
		selections := make([]*goquery.Selection, 0)
		items.Each(func(i int, s *goquery.Selection) {
			selections = append(selections, s)
		})
		num, exists := selections[0].Find("meta").Attr("content")
		if exists {
			episode_info.Num = num
		}
		episode_info.Status = "анонс"
		if selections[1].Length() > 0 {
			episode_info.Title = strings.TrimSpace(selections[1].Text())
		}
		if selections[2].Find("span").Length() > 0 {
			date, exists := selections[2].Find("span").Attr("data-label")
			if exists {
				episode_info.Date = date
			}
		}
		if selections[3].Find("span").Length() > 0 {
			episode_info.Status = "вышел"
		}
		episodes_info = append(episodes_info, &episode_info)
	})
	sort.Slice(episodes_info, func(i, j int) bool {
		numI := episodes_info[i].Num
		numJ := episodes_info[j].Num

		isDigitStr := func(s string) bool {
			if s == "" {
				return false
			}
			for _, r := range s {
				if r < '0' || r > '9' {
					return false
				}
			}
			return true
		}

		if isDigitStr(numI) && isDigitStr(numJ) {
			intI, _ := strconv.Atoi(numI)
			intJ, _ := strconv.Atoi(numJ)
			return intI < intJ
		}

		if isDigitStr(numI) && !isDigitStr(numJ) {
			return true
		}
		if !isDigitStr(numI) && isDigitStr(numJ) {
			return false
		}

		return numI < numJ
	})
	return episodes_info, nil
}

// Расширенный поиск через animego.me. Собирает дополнительные данные об аниме.
//
// :title: Название
//
// Возвращает срез ссылок на SearchResult
func (ab *AniboomParser) Search(title string) ([]*SearchResult, error) {
	elements, err := ab.FastSearch(title)
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : search : FastSearch не смог найти данные для title %s. Ошибка: %v", title, err)
		log.Println(error_message)
		return nil, err
	}
	res := make([]*SearchResult, 0)
	for _, anime := range elements {
		anime_indirect := *anime
		c_data, err := ab.AnimeInfo(anime_indirect.Link)
		if err != nil {
			error_message := fmt.Sprintf("Aniboom parser error : search : AnimeInfo не смог найти данные для %s по ссылке %s. Ошибка: %v", anime.Title, anime.Link, err)
			log.Println(error_message)
			continue
		}
		res = append(res, c_data)
	}

	return res, nil
}

/*
Получение данных об аниме с его страницы на animego.me.

:link: Ссылка на страницу (прим: https:animego.me/anime/volchica-i-pryanosti-torgovec-vstrechaet-mudruyu-volchicu-2546)

Возвращает модель SearchResult
*/
func (ab *AniboomParser) AnimeInfo(link string) (*SearchResult, error) {
	var c_data SearchResult

	request, err := http.NewRequestWithContext(ab.Context, "GET", link, nil)
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : AnimeInfo : не смог создать request. Ошибка: %v", err)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	}

	URL := fmt.Sprintf("https://%s/search/all?q=anime", ab.dmn)

	request.Header.Set("Referer", URL)
	var resp *http.Response
	for attempt := 1; attempt <= 50; attempt++ {
		resp, err = ab.Client.httpClient.Do(request)
		if err != nil {
			error_message := fmt.Sprintf("Aniboom parser error : AnimeInfo : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			continue
		} else if resp.StatusCode != http.StatusOK {
			error_message := fmt.Sprintf("Aniboom parser error : AnimeInfo : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			resp.Body.Close()
			continue
		} else {
			break
		}
	}
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : AnimeInfo : http клиент не смог выполнить запрос. Ошибка: %v", err)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	} else if resp.StatusCode == http.StatusTooManyRequests {
		error_message := "Aniboom parser error : AnimeInfo : Сервер вернул код ошибки 429. Слишком частые запросы"
		log.Println(error_message)
		return nil, perrors.NewTooManyRequestsError(error_message)
	} else if resp.StatusCode != http.StatusOK {
		log.Printf("Aniboom parser error : AnimeInfo : Сервер не вернул ожидаемый код 200. Код: %d\n", resp.StatusCode)
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : AnimeInfo : goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	}
	c_data.Link = link
	fullLink := c_data.Link
	lastDashIndex := strings.LastIndex(fullLink, "-")
	if lastDashIndex != -1 && lastDashIndex < len(fullLink)-1 {
		c_data.AnimegoID = fullLink[lastDashIndex+1:]
	} else {
		c_data.AnimegoID = ""
	}
	dmn := fmt.Sprintf("https://%s", ab.dmn)
	c_data.Title = strings.TrimSpace(doc.Find("div.anime-title h1").Text())

	other_titles := make([]string, 0)

	doc.Find("div.anime-synonyms").Find("li").Each(func(i int, s *goquery.Selection) {
		other_titles = append(other_titles, strings.TrimSpace(s.Text()))
	})

	poster_url_doc := doc.Find("img").First()
	if poster_url_doc.Length() > 0 {
		src, exists := poster_url_doc.Attr("src")
		if exists {
			c_data.PosterURL = src

			SlashIndex := strings.Index(c_data.PosterURL, "/upload")
			if SlashIndex != -1 && SlashIndex < len(c_data.PosterURL)-1 {
				c_data.PosterURL = c_data.PosterURL[SlashIndex+1:]
			} else {
				c_data.PosterURL = ""
			}
			if len(c_data.PosterURL) > 0 {
				c_data.PosterURL = dmn + c_data.PosterURL
			}
		}
	}
	anime_info := doc.Find("div.anime-info dl").First()
	if anime_info.Length() == 0 {
		error_message := "Aniboom parser error : AnimeInfo : doc.Find(\"div.anime-info dl\") не смог найти тег dl"
		log.Println(error_message)
		return nil, perrors.NewNoResultsError(error_message)
	}
	var allDTs []*goquery.Selection
	var allDDs []*goquery.Selection

	anime_info.Find("dt").Each(func(i int, s *goquery.Selection) {
		allDTs = append(allDTs, s)
	})

	anime_info.Find("dd").Each(func(i int, s *goquery.Selection) {
		skip := false

		if s.HasClass("mt-2") && s.HasClass("col-12") {
			skip = true
		}

		if s.Find("hr").First().Length() > 0 {
			skip = true
		}

		if !skip {
			allDDs = append(allDDs, s)
		}
	})
	other_anime_info := make(map[string]string, 0)
	OtherAniInfo := OtherAnimeInfo{}
	minLen := len(allDTs)
	if minLen > len(allDDs) {
		minLen = len(allDDs)
	}

	for i := 0; i < minLen; i++ {
		key := strings.TrimSpace(allDTs[i].Text())
		value := strings.TrimSpace(allDDs[i].Text())

		if key != "" && value != "" {
			other_anime_info[key] = value
		}
	}
	c_data.Unparsed = make(map[string]string)
	for i := 0; i < len(allDTs); i++ {
		key := strings.TrimSpace(allDTs[i].Text())
		value := strings.TrimSpace(allDDs[i].Text())
		if key == "Озвучка" {
			continue
		} else if key == "Жанр" {
			genres := make([]string, 0)
			allDDs[i].Find("a").Each(func(i int, s *goquery.Selection) {
				genres = append(genres, strings.TrimSpace(s.Text()))
			})
			c_data.Genres = genres
		} else if key == "Главные герои" {
			main_characters := make([]string, 0)
			allDDs[i].Find("a").Each(func(i int, s *goquery.Selection) {
				main_characters = append(main_characters, strings.TrimSpace(s.Text()))
			})
			OtherAniInfo.MainCharacters = main_characters
		} else if key == "Эпизоды" {
			c_data.Episodes = value
		} else if key == "Статус" {
			c_data.Status = value
		} else if key == "Тип" {
			c_data.Type = value
		} else {
			switch key {
			case "Возрастные ограничения":
				OtherAniInfo.AgeRests = value
			case "Выпуск":
				OtherAniInfo.ReleaseDate = value
			case "Длительность":
				OtherAniInfo.Duration = value
			case "Первоисточник":
				OtherAniInfo.OriginalSource = value
			case "Рейтинг MPAA":
				OtherAniInfo.MPAARating = value
			case "Сезон":
				OtherAniInfo.Season = value
			case "Снят по ранобэ":
				OtherAniInfo.OriginalRanobe = value
			case "Снят по манге":
				OtherAniInfo.OriginalManga = value
			case "Студия":
				OtherAniInfo.Studio = value
			default:
				c_data.Unparsed[key] = value
			}
		}
	}

	c_data.Description = strings.TrimSpace(doc.Find("div.description").Text())
	screenshots_urls := make([]string, 0)
	doc.Find("a.screenshots-item").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			URL := dmn + strings.TrimSpace(href)
			screenshots_urls = append(screenshots_urls, URL)
		}
	})
	c_data.Screenshots = screenshots_urls

	trailer_cont := doc.Find("a.video-block")
	if trailer_cont.Length() > 0 {
		href, exists := trailer_cont.Find("a.video-item").Attr("href")
		if exists {
			c_data.Trailer = strings.TrimSpace(href)
		}
	}
	result, err := ab.EpisodesInfo(link)
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : AnimeInfo : EpisodesInfo вернул неожиданную ошибку: %v", err)
		log.Println(error_message)
		return nil, err
	}

	c_data.EpisodesInfo = result

	translations_info, err := ab.GetTranslationsInfo(c_data.AnimegoID)
	var contentBlocked *perrors.ContentBlocked
	if errors.As(err, &contentBlocked) {
		log.Println("Aniboom parser warning : AnimeInfo : GetTranslationsInfo вернул ошибку ContentBlocked")
		c_data.Translations = []*Translation{}
	} else if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : AnimeInfo : GetTranslationsInfo вернул неожиданную ошибку: %v", err)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	} else {
		c_data.Translations = translations_info
	}

	c_data.OtherInfo = &OtherAniInfo
	return &c_data, nil
}

// Получает информацию о переводах и их id для плеера aniboom
//
// :animego_id: id аниме на animego.me
//
// Возвращает срез ссылок на Translation:
func (ab *AniboomParser) GetTranslationsInfo(animego_id string) ([]*Translation, error) {

	params := url.Values{}

	params.Set("_allow", "true")

	URL := fmt.Sprintf("https://%s/anime/%s/player?", ab.dmn, animego_id) + params.Encode()

	request, err := http.NewRequestWithContext(ab.Context, "GET", URL, nil)

	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : GetTranslationsInfo : не смог создать request. Ошибка: %v", err)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	}

	referer := fmt.Sprintf("https://%s/search/all?q=anime", ab.dmn)
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	request.Header.Set("Referer", referer)

	var resp *http.Response
	for attempt := 1; attempt <= 50; attempt++ {
		resp, err = ab.Client.httpClient.Do(request)
		if err != nil {
			error_message := fmt.Sprintf("Aniboom parser error : GetTranslationsInfo : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			continue
		} else if resp.StatusCode != http.StatusOK {
			error_message := fmt.Sprintf("Aniboom parser error : GetTranslationsInfo : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			resp.Body.Close()
			continue
		} else {
			break
		}
	}

	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : GetTranslationsInfo :  http клиент не смог выполнить запрос. Ошибка: %v", err)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, perrors.NewTooManyRequestsError("Aniboom parser error : GetTranslationsInfo : Сервер вернул код ошибки 429. Слишком частые запросы")
	}
	if resp.StatusCode != http.StatusOK {
		error_message := fmt.Sprintf("Aniboom parser error : GetTranslationsInfo : Сервер не вернул ожидаемый код 200. Код: %d", resp.StatusCode)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	}

	var json_response JsonResponse

	if err := json.NewDecoder(resp.Body).Decode(&json_response); err != nil {
		return nil, perrors.NewJsonDecodeFailureError(fmt.Sprintf("Aniboom parser error : GetTranslationsInfo : ошибка декодирования json: %v", err))
	}

	if json_response.Status != "success" {
		return nil, perrors.NewServiceError(fmt.Sprintf(
			"Aniboom parser error : FastSearch : сервер вернул статус отличный от success: %q, сообщение: %q для animegoID: %q",
			json_response.Status, json_response.Message, animego_id,
		))
	}
	htmlContent := html.UnescapeString(json_response.Content)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : GetTranslationsInfo : goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
		log.Println(error_message)
		return nil, perrors.NewServiceError(error_message)
	}

	if doc.Find("div.player-blocked").Length() > 0 {
		reason_elem := doc.Find("div.h5")
		var reason string
		if reason_elem.Length() > 0 {
			reason = strings.TrimSpace(reason_elem.Text())
		}
		error_message := fmt.Sprintf("Aniboom parser error : GetTranslationsInfo : Контент по id %s заблокирован. Причина блокировки: \"%s\"", animego_id, reason)
		log.Println(error_message)
		return nil, perrors.NewContentBlockedError(error_message)
	}
	translations_container := doc.Find("#video-dubbing").Find("span.video-player-toggle-item")
	players_container := doc.Find("#video-players").Find("span.video-player-toggle-item")
	translation := make(map[string]*Translation)

	if translations_container.Length() == 0 {
		log.Printf("Aniboom parser warning : GetTranslationsInfo : ни одного translations контейнера не было найдено для animego_id %s", animego_id)
	}
	if players_container.Length() == 0 {
		log.Printf("Aniboom parser warning : GetTranslationsInfo : ни одного players контейнера не было найдено для animego_id %s", animego_id)
	}

	translations_container.Each(func(i int, s *goquery.Selection) {
		dubbing, exists := s.Attr("data-dubbing")
		if !exists || dubbing == "" {
			return
		}

		name := strings.TrimSpace(s.Text())
		if name == "" {
			return
		}

		if _, exists := translation[dubbing]; !exists {
			translation[dubbing] = &Translation{}
		}

		translation[dubbing].Name = name

	})
	translations := make([]*Translation, 0)

	players_container.Each(func(i int, s *goquery.Selection) {
		provider, exists := s.Attr("data-provider")
		if !exists || provider != "24" {
			return
		}

		dubbing, exists := s.Attr("data-provide-dubbing")
		if !exists {
			return
		}

		if _, exists := translation[dubbing]; !exists {
			translation[dubbing] = &Translation{}
		}

		translationID, exists := s.Attr("data-player")
		if !exists {
			return
		}

		lastIndex := strings.LastIndex(translationID, "=")
		if lastIndex != -1 && lastIndex < len(translationID)-1 {
			translationID = translationID[lastIndex+1:]
		}

		translation[dubbing].TranslationID = translationID
	})
	result := make([]*Translation, 0)
	for _, translation_info := range translation {
		if len(translation_info.Name) > 0 && len(translation_info.TranslationID) > 0 {
			result = append(translations, translation_info)
		}
	}

	return result, nil
}

// Возвращает ссылку на embed от aniboom. Сама по себе ссылка не может быть использована, однако требуется для дальнейшего парсинга.
//
// :animego_id: id аниме на animego.me
//
// Возвращает ссылку в виде: https://aniboom.one/embed/yxVdenrqNar
// Если ссылка не найдена, выкидывает NoResults exception
func (ab *AniboomParser) get_embed_link(animego_id string) (string, error) {
	params := url.Values{}

	params.Set("_allow", "true")

	URL := fmt.Sprintf("https://%s/anime/%s/player?%s", ab.dmn, animego_id, params.Encode())

	request, err := http.NewRequestWithContext(ab.Context, "GET", URL, nil)
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_embed_link : не смог создать request. Ошибка: %v", err)
		log.Println(error_message)
		return "", perrors.NewServiceError(error_message)
	}
	request.Header.Set("X-Requested-With", "XMLHttpRequest")

	var resp *http.Response
	for attempt := 1; attempt <= 50; attempt++ {
		resp, err = ab.Client.httpClient.Do(request)
		if err != nil {
			error_message := fmt.Sprintf("Aniboom parser error : get_embed_link : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			continue
		} else if resp.StatusCode != http.StatusOK {
			error_message := fmt.Sprintf("Aniboom parser error : get_embed_link : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			resp.Body.Close()
			continue
		} else if resp.StatusCode == http.StatusTooManyRequests {
			return "", perrors.NewTooManyRequestsError("Aniboom parser error : get_embed_link : Сервер вернул код ошибки 429. Слишком частые запросы")
		} else {
			break
		}
	}
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_embed_link :  http клиент не смог выполнить запрос. Ошибка: %v", err)
		log.Println(error_message)
		return "", perrors.NewServiceError(error_message)
	}
	if resp == nil {
		error_message := "Aniboom parser error : get_embed_link :  http клиент не смог выполнить запрос. Ответ равен nil"
		log.Println(error_message)
		return "", perrors.NewServiceError(error_message)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		error_message := fmt.Sprintf("Aniboom parser error : get_embed_link : Сервер не вернул ожидаемый код 200. Код: %d", resp.StatusCode)
		log.Println(error_message)
		return "", perrors.NewServiceError(error_message)
	}

	var json_response JsonResponse

	if err := json.NewDecoder(resp.Body).Decode(&json_response); err != nil {
		return "", perrors.NewJsonDecodeFailureError(fmt.Sprintf("Aniboom parser error : GetTranslationsInfo : ошибка декодирования json: %v", err))
	}

	if json_response.Status != "success" {
		return "", perrors.NewServiceError(fmt.Sprintf(
			"Aniboom parser error : FastSearch : сервер вернул статус отличный от success: %q, сообщение: %q для animegoID: %q",
			json_response.Status, json_response.Message, animego_id,
		))
	}

	htmlContent := html.UnescapeString(json_response.Content)
	// file, err := os.Create("index.html")
	// if err != nil {
	// 	log.Println("Aniboom parser error : GetAsFile : GetMPDPlaylist не смог создать файл")
	// 	return "", err
	// }
	// defer file.Close()

	// if _, err := file.WriteString(htmlContent); err != nil {
	// 	log.Println("Aniboom parser error : GetAsFile : GetMPDPlaylist не смог записать данные в файл")
	// 	return "", err
	//}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_embed_link : goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
		log.Println(error_message)
		return "", perrors.NewServiceError(error_message)
	}

	items := doc.Find("div.player-blocked").First()

	if items.Length() > 0 {
		reason_elem := doc.Find("div.h5").First()
		reason := ""
		if reason_elem.Length() > 0 {
			reason = strings.TrimSpace(reason_elem.Text())
		}
		return "", perrors.NewNoResultsError(fmt.Sprintf("Aniboom parser error : get_embed_link : контент по id %s заблокирован. Причина: %v", animego_id, reason))
	}
	link := doc.Find("div#video-players")

	span := link.Find("span.video-player-toggle-item[data-provider=\"24\"]").First()

	var player_link string
	if span.Length() > 0 {
		if attrValue, exists := span.Attr("data-player"); exists && len(attrValue) > 0 {
			player_link = attrValue
		} else {
			return "", perrors.NewAttributeError(fmt.Sprintf("Aniboom parser error : get_embed_link : для указанного id %s не удалось найти aniboom embed_link", animego_id))
		}
	} else {
		return "", perrors.NewServiceError("Aniboom parser error : get_embed_link : span с video-player-toggle-item не найден или отсутствует атрибут data-provider=24")
	}

	lastQuestionIndex := strings.LastIndex(player_link, "?")
	if lastQuestionIndex != -1 && lastQuestionIndex < len(player_link)-1 {
		return "https:" + player_link[:lastQuestionIndex], nil
	} else {
		return "", perrors.NewServiceError(fmt.Sprintf("Не удалось найти \"?\" для ссылки: %s", player_link))
	}
}

// Возвращает html от embed(iframe) плеера aniboom
//
// :embed_link: ссылка на embed (можно получить из get_embed_link)
//
// :episode: Номер эпизода (вышедшего) (Если фильм - 0)
//
// :translation: id перевода (который именно для aniboom плеера) (можно получить из GetTranslationsInfo)
func (ab *AniboomParser) get_embed(embed_link, translation string, episode int) (string, error) {
	params := url.Values{}

	referer := fmt.Sprintf("https://%s/", ab.dmn)
	params.Set("translation", translation)

	if episode != 0 {
		params.Set("episode", fmt.Sprintf("%d", episode))
	}
	link := fmt.Sprintf("%s?", embed_link) + params.Encode()

	request, err := http.NewRequestWithContext(ab.Context, "GET", link, nil)
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_embed_link : не смог создать request. Ошибка: %v", err)
		log.Println(error_message)
		return "", perrors.NewServiceError(error_message)
	}
	request.Header.Set("Referer", referer)

	var resp *http.Response
	for attempt := 1; attempt <= 50; attempt++ {
		resp, err = ab.Client.httpClient.Do(request)
		if err != nil {
			error_message := fmt.Sprintf("Aniboom parser error : get_embed : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			continue
		} else if resp.StatusCode != http.StatusOK {
			error_message := fmt.Sprintf("Aniboom parser error : get_embed : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			resp.Body.Close()
			continue
		} else {
			break
		}
	}
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_embed :  http клиент не смог выполнить запрос. Ошибка: %v", err)
		log.Println(error_message)
		return "", perrors.NewServiceError(error_message)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", perrors.NewTooManyRequestsError("Aniboom parser error : get_embed : Cервер вернул код ошибки 429. Слишком частые запросы")
	}
	if resp.StatusCode != http.StatusOK {
		error_message := fmt.Sprintf("Aniboom parser error : get_embed_link : Сервер не вернул ожидаемый код 200. Код: %d", resp.StatusCode)
		return "", perrors.NewServiceError(error_message)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Aniboom parser error : get_embed_link : Ошибка при чтении тела ответа:", err)
		return "", perrors.NewServiceError(fmt.Sprintf("Aniboom parser error : get_embed_link : Ошибка при чтении тела ответа: %v", err))
	}

	bodyText := string(bodyBytes)
	return bodyText, nil
}

// :embed_link: ссылка на embed (можно получить из _get_embed_link)
//
// :episode: Номер эпизода (вышедшего) (Если фильм - 0)
//
// :translation: id перевода (который именно для aniboom плеера) (можно получить из GetTranslationsInfo)
//
// Пример возвращаемого: https://sophia.yagami-light.com/7p/7P9qkv26dQ8/v26utto64xx66.mpd
func (ab *AniboomParser) get_media_src(embed_link, translation string, episode int) (string, error) {
	embed, err := ab.get_embed(embed_link, translation, episode)
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_media_src : get_embed вернула ошибку: %v", err)
		return "", perrors.NewServiceError(error_message)
	}
	htmlContent := html.UnescapeString(embed)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_media_src : goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
		return "", perrors.NewServiceError(error_message)
	}
	var jsonData string
	doc.Find("div#video").First().Each(func(i int, s *goquery.Selection) {
		data_params, exists := s.Attr("data-parameters")
		if exists {
			jsonData = data_params
		} else {
			jsonData = ""
		}
	})
	if len(jsonData) == 0 {
		return "", perrors.NewServiceError("Aniboom parser error : get_media_src : для указанного embed_link \"%s\" в div#video не найден атрибут data-parameters")
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_media_src : не удалось преобразовать jsonData. Ошибка: %v\njsonData: %s", err, jsonData)
		return "", perrors.NewServiceError(error_message)
	}

	var dash_data map[string]interface{}
	str_data := fmt.Sprintf("%v", data["dash"])
	if err := json.Unmarshal([]byte(str_data), &dash_data); err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_media_src : не удалось преобразовать first_data. Ошибка: %v", err)
		return "", perrors.NewServiceError(error_message)
	}
	src := dash_data["src"]
	media_src, ok := src.(string)
	if !ok {
		error_message := fmt.Sprintf("Aniboom parser error : get_media_src : src не является строкой. Src: %v", src)
		return "", perrors.NewServiceError(error_message)
	}
	return media_src, nil
}

// Возвращает путь до mpd файла (без самого файла)
//
// :embed_link: ссылка на embed (можно получить из _get_embed_link)
// :episode: Номер эпизода (вышедшего) (Если фильм - 0)
// :translation: id перевода (который именно для aniboom плеера) (можно получить из GetTranslationsInfo)
//
// Пример возвращаемого: https://sophia.yagami-light.com/7p/7P9qkv26dQ8/
func (ab *AniboomParser) get_media_server(embed_link, translation string, episode int) (string, error) {
	src, err := ab.get_media_src(embed_link, translation, episode)
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_media_server : get_media_src вернул ошибку. Ошибка: %v", err)
		return "", perrors.NewServiceError(error_message)
	}
	lastSlashIndex := strings.LastIndex(src, "/")
	if lastSlashIndex != -1 && lastSlashIndex < len(src)-1 {
		src = src[:lastSlashIndex+1]
	} else {
		error_message := fmt.Sprintf("Aniboom parser error : get_media_server : Не удалось найти \"/\" для ссылки: %s", src)
		return "", perrors.NewServiceError(error_message)
	}
	return src, nil
}

// Просто отрезает от ссылки на mpd файл сам файл.
//
// :media_src: ссылка на mpd файл (прим: https://sophia.yagami-light.com/7p/7P9qkv26dQ8/v26utto64xx66.mpd)
//
// Пример возвращаемого: https://sophia.yagami-light.com/7p/7P9qkv26dQ8/
func (ab *AniboomParser) get_media_server_from_src(media_str string) (string, error) {
	lastSlashIndex := strings.LastIndex(media_str, "/")
	if lastSlashIndex != -1 && lastSlashIndex < len(media_str)-1 {
		media_str = media_str[:lastSlashIndex+1]
	} else {
		error_message := fmt.Sprintf("Aniboom parser error : get_media_server_from_src : Не удалось найти \"/\" для ссылки: %s", media_str)
		return "", perrors.NewServiceError(error_message)
	}
	return media_str, nil
}

// Получение файла mpd через embed_link
//
// :embed_link: ссылка на embed (можно получить из _get_embed_link)
// :episode: Номер эпизода (вышедшего) (Если фильм - 0)
// :translation: id перевода (который именно для aniboom плеера) (можно получить из GetTranslationsInfo)
//
// Возвращает mpd файл в виде текста. (Можно сохранить результат как res.mpd и при запуске через поддерживающий mpd файлы плеер должна начаться серия)
// Обратите внимание, что файл содержит именно ссылки на части изначального файла, поэтому не сможет запуститься без интернета.
// Также в файле содержится сразу несколько "качеств" видео (от 480 до 1080 в большинстве случаев).
// Если вам нужен mp4 файл воспользуйтесь ffmpeg или другими конвертерами
func (ab *AniboomParser) get_mpd_playlist(embed_link, translation string, episode int) (string, error) {
	embed, err := ab.get_embed(embed_link, translation, episode)
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_mpd_playlist : get_embed вернул ошибку. Ошибка: %v", err)
		return "", perrors.NewServiceError(error_message)
	}
	// htmlContent := html.UnescapeString(embed)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(embed))
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_mpd_playlist : goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
		return "", perrors.NewServiceError(error_message)
	}
	var jsonData string
	doc.Find("div#video").First().Each(func(i int, s *goquery.Selection) {
		data_params, exists := s.Attr("data-parameters")
		if exists {
			jsonData = data_params
		} else {
			jsonData = ""
		}
	})
	if len(jsonData) == 0 {
		return "", perrors.NewServiceError("Aniboom parser error : get_mpd_playlist : для указанного embed_link \"%s\" в div#video не найден атрибут data-parameters")
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_mpd_playlist : не удалось преобразовать jsonData. Ошибка: %v\njsonData:%s", err, jsonData)
		return "", perrors.NewServiceError(error_message)
	}

	var dash_data map[string]interface{}
	str_data := fmt.Sprintf("%v", data["dash"])
	if err := json.Unmarshal([]byte(str_data), &dash_data); err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_mpd_playlist : не удалось преобразовать first_data. Ошибка: %v", err)
		return "", perrors.NewServiceError(error_message)
	}
	src := dash_data["src"]
	media_src, ok := src.(string)
	if !ok {
		error_message := fmt.Sprintf("Aniboom parser error : get_mpd_playlist : src не является строкой. Src: %v", src)
		return "", perrors.NewServiceError(error_message)
	}

	origin := "https://aniboom.one"
	referer := "https://aniboom.one/"

	request, err := http.NewRequestWithContext(ab.Context, "GET", media_src, nil)
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_mpd_playlist : не смог создать request. Ошибка: %v", err)
		log.Println(error_message)
		return "", perrors.NewServiceError(error_message)
	}
	request.Header.Set("Origin", origin)
	request.Header.Set("Referer", referer)

	var playlist *http.Response
	for attempt := 1; attempt <= 50; attempt++ {
		playlist, err = ab.Client.httpClient.Do(request)
		if err != nil {
			error_message := fmt.Sprintf("Aniboom parser error : get_mpd_playlist : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			playlist.Body.Close()
			continue
		} else if playlist.StatusCode != http.StatusOK {
			error_message := fmt.Sprintf("Aniboom parser error : get_mpd_playlist : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			playlist.Body.Close()
			continue
		} else if playlist == nil {
			error_message := fmt.Sprintf("Aniboom parser error : get_mpd_playlist : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			continue
		} else {
			break
		}
	}
	if err != nil {
		error_message := fmt.Sprintf("Aniboom parser error : get_mpd_playlist :  http клиент не смог выполнить запрос. Ошибка: %v", err)
		log.Println(error_message)
		return "", perrors.NewServiceError(error_message)
	}
	defer playlist.Body.Close()
	if playlist.StatusCode != http.StatusOK {
		error_message := fmt.Sprintf("Aniboom parser error : get_mpd_playlist : Сервер не вернул ожидаемый код 200. Код: %d", playlist.StatusCode)
		return "", perrors.NewServiceError(error_message)
	}

	body, err := io.ReadAll(playlist.Body)
	str_playlist := string(body)
	if strings.Contains(str_playlist, "<MPD") {
		lastSlashIndex := strings.LastIndex(media_src, "/")
		lastDotIndex := strings.LastIndex(media_src, ".")
		if lastSlashIndex == -1 || lastDotIndex == -1 || lastSlashIndex >= lastDotIndex {
			return "", perrors.NewServiceError("Aniboom parser error: invalid media_src format")
		}
		filename := media_src[lastSlashIndex+1 : lastDotIndex]

		server_path := media_src[:lastDotIndex]
		str_playlist = strings.Replace(str_playlist, filename, server_path, -1)
	} else {
		lastSubstrIndex := strings.LastIndex(media_src, "master_device.m3u8")
		if lastSubstrIndex == -1 {
			return "", errors.New("master_device.m3u8 не найден в media_src")
		}
		server_path := media_src[:lastSubstrIndex]
		str_playlist = strings.Replace(str_playlist, "media_", server_path+"media_", -1)
	}
	return str_playlist, nil
}

// Возвращает mpd файл строкой (содержимое файла)
//
// :animego_id: id аниме на animego.me (может быть найдена из FastSearch по в поле AnimegoID для нужного аниме или из Search по тому же полю для нужного аниме) (из ссылки на страницу аниме https://animego.me/anime/volchica-i-pryanosti-torgovec-vstrechaet-mudruyu-volchicu-2546 > 2546)
//
// :episode: Номер эпизода (вышедшего) (Если фильм - 0)
//
// :translation_id: id перевода (который именно для aniboom плеера) (можно получить из GetTranslationsInfo)
//
// Возвращает mpd файл в виде текста. (Можно сохранить результат как res.mpd и при запуске через поддерживающий mpd файлы плеер должна начаться серия)
// Обратите внимание, что файл содержит именно ссылки на части изначального файла, поэтому не сможет запуститься без интернета.
// Также в файле содержится сразу несколько "качеств" видео (от 480 до 1080 в большинстве случаев).
// Если вам нужен mp4 файл воспользуйтесь ffmpeg или другими конвертерами
func (ab *AniboomParser) GetMPDPlaylist(animego_id, translation_id string, episode int) (string, error) {
	embed_link, err := ab.get_embed_link(animego_id)
	if err != nil {
		log.Printf("Aniboom parser error : GetMPDPlaylist : get_embed_link вернул ошибку. Ошибка: %v", err)
		return "", err
	}

	mpd_playlist, err := ab.get_mpd_playlist(embed_link, translation_id, episode)
	if err != nil {
		log.Printf("Aniboom parser error : GetMPDPlaylist : get_mpd_playlist вернул ошибку. Ошибка: %v", err)
		return "", err
	}
	return mpd_playlist, nil
}

// Сохраняет mpd файл как указанный filename
//
// :animego_id: id аниме на animego.me (может быть найдена из FastSearch в поле AnimegoID для нужного аниме или из Search по тому же полю для нужного аниме) (из ссылки на страницу аниме https://animego.me/anime/volchica-i-pryanosti-torgovec-vstrechaet-mudruyu-volchicu-2546 > 2546)
//
// :episode: Номер эпизода (вышедшего) (Если фильм - 0)
//
// :translation_id: id перевода (который именно для aniboom плеера) (можно получить из GetTranslationsInfo)
//
// :filename: Имя/путь для сохраняемого файла обязательно чтобы было .mpd расширение (прим: result.mpd или content/result.mpd)
//
// Обратите внимание, что файл содержит именно ссылки на части изначального файла, поэтому не сможет запуститься без интернета.
// Также в файле содержится сразу несколько "качеств" видео (от 480 до 1080 в большинстве случаев).
// Если вам нужен mp4 файл воспользуйтесь ffmpeg или другими конвертерами
func (ab *AniboomParser) GetAsFile(animego_id, translation_id, filename string, episode int) error {
	mpd_playlist, err := ab.GetMPDPlaylist(animego_id, translation_id, episode)
	if err != nil {
		log.Println("Aniboom parser error : GetAsFile : GetMPDPlaylist вернул ошибку")
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		log.Println("Aniboom parser error : GetAsFile : GetMPDPlaylist не смог создать файл")
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(mpd_playlist); err != nil {
		log.Println("Aniboom parser error : GetAsFile : GetMPDPlaylist не смог записать данные в файл")
		return err
	}

	return nil
}
