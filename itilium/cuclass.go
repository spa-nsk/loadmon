package itilium

// Классификаторы КЕ (cuClass) в HTTP-сервисе `/hs/api/getCU` (BCS / 1С:ITILIUM).
// Числовые коды сверены с перечнем, переданным для публикации ITILIUM.
const (
	CUClassComputers       = 139 // IT_Компьютеры
	CUClassMonitors        = 140 // IT_Мониторы
	CUClassLaptops         = 141 // IT_Ноутбуки
	CUClassBlades          = 142 // IT_Блейды
	CUClassKVM             = 143 // IT_КВМ
	CUClassPrinters        = 144 // IT_Принтеры
	CUClassServers         = 145 // IT_Серверы
	CUClassNetwork         = 146 // IT_Сетевое оборудование
	CUClassScanners        = 147 // IT_Сканеры
	CUClassMobilePhones    = 148 // IT_Мобильные телефоны
	CUClassThinClients     = 149 // IT_Тонкие клиенты
	CUClassUSBOverIP       = 154 // IT_USBOverIP
	CUClassStorage         = 158 // IT_СХД
	CUClassBladeEnclosures = 159 // IT_Корзины блейдов
	CUClassSAN             = 162 // IT_SAN
	CUClassIPPhones        = 168 // IT_IP телефоны
	CUClassIPHeadsets      = 170 // IT_IP гарнитуры
	CUClassUPS             = 171 // IT_ИБП
	CUClassAdapters        = 173 // IT_Адаптеры
	CUClassPDU             = 177 // IT_PDU
	CUClassNettops         = 178 // IT_Неттопы
	CUClassDockStations    = 179 // IT_Док-станции
	CUClassTablets         = 180 // IT_Планшеты
	CUClassKeyboards       = 181 // IT_Клавиатуры
	CUClassMice            = 182 // IT_Мыши
	CUClassVCSStations     = 183 // IT_ВКС станции
	CUClassSpeakerphones   = 185 // IT_Спикерфоны
	CUClassAllInOnes       = 186 // IT_Моноблоки
	CUClassCabinets        = 187 // IT_Шкафы
	CUClassWebcams         = 189 // IT_web-камеры
	CUClassPhones          = 190 // IT_Телефоны
	CUClassOther           = 192 // IT_Прочее
	CUClassTapeLibraries   = 198 // IT_Ленточные библиотеки
	CUClassProjectors      = 201 // IT_Проекторы
	CUClassMediaPlayers    = 203 // IT_Медиаплееры
	CUClassMicrophones     = 205 // IT_Микрофоны
	CUClassTVs             = 226 // IT_Телевизоры
	CUClassUSBKeys         = 228 // IT_USB ключи
	CUClassBarcodeScanners = 231 // IT_Сканеры штрих-кодов
)

// CUClassName — человекочитаемое имя класса по коду (пусто, если неизвестен).
func CUClassName(code int) string {
	if name, ok := cuClassNames[code]; ok {
		return name
	}
	return ""
}

// CUClasses — упорядоченный список известных классов (для UI/справок).
func CUClasses() []CUClassInfo {
	out := make([]CUClassInfo, len(cuClassOrder))
	for i, code := range cuClassOrder {
		out[i] = CUClassInfo{Code: code, Name: cuClassNames[code]}
	}
	return out
}

// CUClassInfo — пара код/имя классификатора КЕ.
type CUClassInfo struct {
	Code int
	Name string
}

var cuClassNames = map[int]string{
	CUClassIPHeadsets:      "IT_IP гарнитуры",
	CUClassIPPhones:        "IT_IP телефоны",
	CUClassPDU:             "IT_PDU",
	CUClassSAN:             "IT_SAN",
	CUClassUSBKeys:         "IT_USB ключи",
	CUClassUSBOverIP:       "IT_USBOverIP",
	CUClassWebcams:         "IT_web-камеры",
	CUClassAdapters:        "IT_Адаптеры",
	CUClassBlades:          "IT_Блейды",
	CUClassVCSStations:     "IT_ВКС станции",
	CUClassDockStations:    "IT_Док-станции",
	CUClassUPS:             "IT_ИБП",
	CUClassKVM:             "IT_КВМ",
	CUClassKeyboards:       "IT_Клавиатуры",
	CUClassComputers:       "IT_Компьютеры",
	CUClassBladeEnclosures: "IT_Корзины блейдов",
	CUClassTapeLibraries:   "IT_Ленточные библиотеки",
	CUClassMediaPlayers:    "IT_Медиаплееры",
	CUClassMicrophones:     "IT_Микрофоны",
	CUClassMobilePhones:    "IT_Мобильные телефоны",
	CUClassMonitors:        "IT_Мониторы",
	CUClassAllInOnes:       "IT_Моноблоки",
	CUClassMice:            "IT_Мыши",
	CUClassNettops:         "IT_Неттопы",
	CUClassLaptops:         "IT_Ноутбуки",
	CUClassTablets:         "IT_Планшеты",
	CUClassPrinters:        "IT_Принтеры",
	CUClassProjectors:      "IT_Проекторы",
	CUClassOther:           "IT_Прочее",
	CUClassServers:         "IT_Серверы",
	CUClassNetwork:         "IT_Сетевое оборудование",
	CUClassScanners:        "IT_Сканеры",
	CUClassBarcodeScanners: "IT_Сканеры штрих-кодов",
	CUClassSpeakerphones:   "IT_Спикерфоны",
	CUClassStorage:         "IT_СХД",
	CUClassTVs:             "IT_Телевизоры",
	CUClassPhones:          "IT_Телефоны",
	CUClassThinClients:     "IT_Тонкие клиенты",
	CUClassCabinets:        "IT_Шкафы",
}

// Порядок как в переданном списке (по смыслу групп, не по коду).
var cuClassOrder = []int{
	CUClassIPHeadsets, CUClassIPPhones, CUClassPDU, CUClassSAN, CUClassUSBKeys,
	CUClassUSBOverIP, CUClassWebcams, CUClassAdapters, CUClassBlades, CUClassVCSStations,
	CUClassDockStations, CUClassUPS, CUClassKVM, CUClassKeyboards, CUClassComputers,
	CUClassBladeEnclosures, CUClassTapeLibraries, CUClassMediaPlayers, CUClassMicrophones,
	CUClassMobilePhones, CUClassMonitors, CUClassAllInOnes, CUClassMice, CUClassNettops,
	CUClassLaptops, CUClassTablets, CUClassPrinters, CUClassProjectors, CUClassOther,
	CUClassServers, CUClassNetwork, CUClassScanners, CUClassBarcodeScanners,
	CUClassSpeakerphones, CUClassStorage, CUClassTVs, CUClassPhones, CUClassThinClients,
	CUClassCabinets,
}
