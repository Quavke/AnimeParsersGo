package parsers_errors

// Ошибка для обозначения неверного токена.
type TokenError struct {
	message string
}

func NewTokenError(message string) error {
	return &TokenError{message: message}
}

func (e *TokenError) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Неверный токен"
}

// Ошибка для обозначения ошибки на стороне сервера
type ServiceError struct {
	message string
}

func NewServiceError(message string) error {
	return &ServiceError{message: message}
}

func (e *ServiceError) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Ошибка на стороне сервера"
}

// Ошибка для обозначения неверно переданных аргументов серверу
type PostArgumentsError struct {
	message string
}

func NewPostArgumentsError(message string) error {
	return &PostArgumentsError{message: message}
}

func (e *PostArgumentsError) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Серверу поданы неверные аргументы"
}

// Ошибка для обозначения отсутствия результатов
type NoResults struct {
	message string
}

func NewNoResultsError(message string) error {
	return &NoResults{message: message}
}

func (e *NoResults) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Результаты отсутствуют"
}

// Ошибка для обозначения неожиданного или необработанного поведения
type UnexpectedBehavior struct {
	message string
}

func NewUnexpectedBehaviorError(message string) error {
	return &UnexpectedBehavior{message: message}
}

func (e *UnexpectedBehavior) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Программа повела себя неожиданно или поведение не было обработано"
}

// Ошибка для обозначения не найденного запрашиваемого качества видео
type QualityNotFound struct {
	message string
}

func NewQualityNotFoundError(message string) error {
	return &QualityNotFound{message: message}
}

func (e *QualityNotFound) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Запрашиваемое качество видео не найдено"
}

// Ошибка для обозначения что контент заблокирован из-за возрастного рейтинга
type AgeRestricted struct {
	message string
}

func NewAgeRestrictedError(message string) error {
	return &AgeRestricted{message: message}
}

func (e *AgeRestricted) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Контент заблокирован из-за возрастного рейтинга"
}

// Ошибка для обозначения ошибки 429 из-за слишком частых запросов. В основном для шикимори
type TooManyRequests struct {
	message string
}

func NewTooManyRequestsError(message string) error {
	return &TooManyRequests{message: message}
}

func (e *TooManyRequests) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Слишком частые запросы"
}

// Ошибка для обозначения заблокированного контента/плеера
type ContentBlocked struct {
	message string
}

func NewContentBlockedError(message string) error {
	return &ContentBlocked{message: message}
}

func (e *ContentBlocked) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Контент или плеер был заблокирован"
}

// Ошибка для обозначения http кода 520. Используется в парсере shikimori
type ServiceIsOverloaded struct {
	message string
}

func NewServiceIsOverloadedError(message string) error {
	return &ServiceIsOverloaded{message: message}
}

func (e *ServiceIsOverloaded) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Сервер перегружен"
}

// При попытке дешифровать ссылку от Kodik возникла ошибка
type DecryptionFailure struct {
	message string
}

func NewDecryptionFailureError(message string) error {
	return &DecryptionFailure{message: message}
}

func (e *DecryptionFailure) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Не удалось дешифровать ссылку от kodik"
}

// Ошибка для обозначения неудачного декодирования ответа сервера в json
type JsonDecodeFailure struct {
	message string
}

func NewJsonDecodeFailureError(message string) error {
	return &JsonDecodeFailure{message: message}
}

func (e *JsonDecodeFailure) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Не удалось преобразовать ответ сервера в json"
}

// Ошибка для обозначения ошибки парсинга html
type HTMLParse struct {
	message string
}

func NewHTMLParseError(message string) error {
	return &HTMLParse{message: message}
}

func (e *HTMLParse) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Не удалось найти тег, атрибут, класс, id или другое"
}

type AttributeError struct {
	message string
}

func NewAttributeError(message string) error {
	return &HTMLParse{message: message}
}

func (e *AttributeError) Error() string {
	if e.message != "" {
		return e.message
	}
	return "Не удалось найти атрибут"
}
