package parsers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	parsers_errors "github.com/Quavke/AnimeParsersGo/errors"
)

type Client struct {
	httpClient *http.Client
}

type AniboomParser struct {
	dmn    string
	Client *Client
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
		dmn:    dmn,
		Client: client,
	}
}

type FastSearchResult struct {
	Title      string `json:"title"`
	Year       string `json:"year"`
	OtherTitle string `json:"other_title"`
	AnimeType  string `json:"type"`
	Link       string `json:"link"`
	AnimegoID  string `json:"animego_id"`
}

//Быстрый поиск через animego.me

//:title: Название аниме

// Возвращает список словарей в виде:
// [
//
//	{
//	'title': Название аниме
//	'year': Год выпуска
//	'other_title': Другое название (оригинальное название)
//	'type': Тип аниме (ТВ сериал, фильм, ...)
//	'link': Ссылка на страницу с информацией
//	'animego_id': id на анимего (по сути в ссылке на страницу с информацией последняя цифра и есть id)
//	},
//
// ...
// ]
func (ab AniboomParser) Fast_search(ctx context.Context, title string) ([]*FastSearchResult, error) {
	params := url.Values{}

	params.Set("type", "small")
	params.Set("q", title)
	domain := fmt.Sprintf("https://%s", ab.dmn)
	url := fmt.Sprintf("/%ssearch/all?%s", domain, params.Encode())
	request, err := http.NewRequestWithContext(ctx, "get", url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	request.Header.Set("Referer", domain)

	resp, err := ab.Client.httpClient.Do(request)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Aniboom parser: http клиент не смог выполнить запрос, код %d. Ошибка: %v", resp.StatusCode, err)
		return nil, parsers_errors.ServiceError
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Aniboom parser: goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
		return nil, parsers_errors.ServiceError
	}
	res := make([]*FastSearchResult, 0)
	items := doc.Find("div.result-search-anime").First().Find("div.result-search-item")

	if items.Length() == 0 {
		log.Printf("Aniboom parser: в контейнере result-search-anime не найдено ни одного элемента div.result-search-item")
		return nil, parsers_errors.NoResults
	}

	items.Each(func(i int, s *goquery.Selection) {
		c_data := &FastSearchResult{}
		c_data.Title = strings.TrimSpace(s.Find("h5").First().Text())
		c_data.Year = strings.TrimSpace(s.Find("span.anime-year").First().Text())
		c_data.OtherTitle = s.Find("div.text-truncate").First().Text()
		c_data.AnimeType = s.Find("a[href*=\"anime/type\"]").First().Text()
		link := s.Find("h5 a").First()
		var rawLink string
		if link.Length() > 0 {
			href, exists := link.Attr("href")
			if exists {
				rawLink = href
			}
		}
		c_data.Link = domain + rawLink
		fullLink := c_data.Link
		var animego_id string
		lastDashIndex := strings.LastIndex(fullLink, "-")
		if lastDashIndex != -1 && lastDashIndex < len(fullLink)-1 {
			animego_id = fullLink[lastDashIndex+1:]
		} else {
			animego_id = ""
		}
		c_data.AnimegoID = animego_id
		res = append(res, c_data)
	})

	return res, nil
}

// Расширенный поиск через animego.me. Собирает дополнительные данные об аниме.

// :title: Название

// Возвращает список из словарей:
// [
//
//	{
//	    'title': Название
//	    'other_titles': [Альтернативное название 1, ...]
//	    'status': Статус аниме (онгоинг, анонс, вышел, ...)
//	    'type': Тип аниме (ТВ сериал, фильм, ...)
//	    'genres': [Список жанров]
//	    'description': описание
//	    'episodes': если аниме вышло, то количество серий, если еще идет, то "вышло / всего"
//	    'episodes_info': [
//	        {
//	            'num': Номер эпизода
//	            'title': Название эпизода
//	            'date': Даты выхода (предполагаемые если анонс)
//	            'status': 'вышло' или 'анонс' (Имеется в виду вышло в оригинале, не переведено)
//	        },
//	        ...
//	    ],
//	    'translations': [
//	        {
//	            'name': Название студии,
//	            'translation_id': id перевода в плеере aniboom
//	        },
//	        ...
//	    ],
//	    'poster_url': Ссылка на постер аниме
//	    'trailer': Ссылка на ютуб embed трейлер
//	    'screenshots': [Список ссылок на скриншоты]
//	    'other_info': {
//	        Данная информация может меняться в зависимости от типа или состояния тайтла
//	        'Возрастные ограничения': (прим: 16+)
//	        'Выпуск': (прим: с 2 апреля 2024)
//	        'Главные герои': [Список главных героев]
//	        'Длительность': (прим: 23 мин. ~ серия)
//	        'Первоисточник': (прим: Легкая новела)
//	        'Рейтинг MPAA': (прим: PG-13),
//	        'Сезон': (прим. Весна 2024),
//	        'Снят по ранобэ': название ранобэ (Или так же может быть 'Снят по манге')
//	        'Студия': название студии
//	    }
//	    'link': Ссылка на страницу с информацией
//	    'animego_id': id на анимего (по сути в ссылке на страницу с информацией последняя цифра и есть id)
//	},
//	...
//
// ]
func (ab AniboomParser) search(title string) SearchResult
