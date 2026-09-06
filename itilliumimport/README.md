# itilliumimport

Конвертация выгрузок из Итиллиум в модели loadmon:

- **rack-серверы** (пример `Серверы_templ.xlsx`, лист `export`) →
  `models.ServerSprRecords`/`models.ServerHardware` — `ImportItiliumServer`.
- **блейд-корзины (шасси)** → `models.BladeSystems` — `ImportItiliumBlade`
  (2026-09-06, аналог импорта серверов).
- **блейд-серверы** (пример `Блейды_templ.xlsx`, лист `export`) →
  `models.ServerSprRecords`/`models.ServerHardware` — `ImportItiliumBladeServer`
  (2026-09-06, использует ТЕ ЖЕ целевые структуры, что и rack-серверы, плюс
  дополнительную привязку к `models.BladeSystems`).

Все три связки наполняют `models.DatacenterSite`/`models.RackCabinet` (ЦОД/
шкаф) по одному и тому же принципу "найти по имени, создать при
отсутствии" — см. `kontext/03-data-model.md`, раздел 5a. **Порядок
импорта важен**: блейд-корзины (`/admin/importblades`) нужно импортировать
ДО блейд-серверов (`/admin/importbladeservers`) — иначе привязка
`ServerHardware.BladeSystemId` не найдёт корзину и будет пропущена (с
предупреждением в логе).

## Файлы

- `mapping.yaml` — образец конфигурации связки "колонка Итиллиум → поле
  модели" для **rack-серверов**. **Боевой файл, который реально читает
  `ImportItiliumServer`, лежит в каталоге, заданном ключом `yaml` в
  `app.conf`** (см. `server_web_new/conf/app.conf`, по умолчанию
  `./conf/yaml`), под именем `rack_server.yaml` (см.
  `mappingConfigFileName`/`mappingConfigDir()` в `importrackservers.go`).
  Этот файл в пакете — просто копия-образец для истории/сравнения;
  **правьте `<yaml-каталог>/rack_server.yaml`**, если поменялся
  состав/название колонок в новой выгрузке или нужно другое
  сопоставление/словарь статусов. Подробные комментарии — прямо в файле.
- `mappingblade.yaml` — то же самое, но образец для **блейд-корзин**; боевой
  файл — `<yaml-каталог>/rack_blade.yaml` (см.
  `mappingConfigFileNameBlade` в `importblades.go`). Сверен с реальным
  примером `Корзины блейдов_templ.xlsx` (14 колонок, лист `export`) —
  состав заметно уже, чем у серверов (нет модели/памяти/CPU/IPMI и т.п.).
  ⚠️ Важная особенность: колонка `"Стар.инв.№"` в этой выгрузке содержит НЕ
  старый инвентарный номер, а короткое имя шасси (`C7000-01`, `ENC-21`,
  `M1-01` и т.п.) — именно оно замаплено на `BladeSystems.Name` и служит
  ключом поиска при повторном импорте (см. предупреждение прямо в файле).
- `mappingbladeservers.yaml` — образец для **блейд-серверов**; боевой файл —
  `<yaml-каталог>/rack_blade_server.yaml` (см.
  `mappingConfigFileNameBladeServer` в `importbladeservers.go`). Сверен с
  реальным примером `Блейды_templ.xlsx` (18 колонок, лист `export`) —
  состав уже, чем у rack-серверов (нет памяти/CPU/HDD/RAID/IPMI/сети/ОС), но
  есть две колонки, которых нет у rack-серверов: `"Корзина"` (ссылка на
  блейд-шасси вида `"(IT-0000003911) C7000-15"` — тех.номер в скобках + имя
  шасси) и `"Bay Number"` (номер слота, → `ServerHardware.BladeNum`
  напрямую). Целевые структуры — `ServerSprRecords`/`ServerHardware`, те же,
  что и у rack-серверов, поэтому читается тем же `LoadMapping`/`ConvertRow`
  — отдельный composite потребовался только для `"Корзина"`
  (`blade_ref_split`, см. `convert.go`), который кладёт разобранное имя
  шасси в `extra.blade_ref_name` (потребляется в `importbladeservers.go`,
  сам composite ничего в БД не ищет — пакет по-прежнему не пишет в БД, см.
  ниже).
- `mapping.go` — загрузка `mapping.yaml`/`mappingblade.yaml`/
  `mappingbladeservers.yaml` (`LoadMapping`, общая для всех связок — формат
  YAML одинаковый, различается только префикс в `target:` —
  `ServerSprRecords.`/`ServerHardware.` для серверов (rack и блейд),
  `BladeSystems.` для блейд-корзин).
- `convert.go` — логика конвертации одной строки rack-сервера (`ConvertRow`)
  по правилам из `mapping.yaml`: простые поля (`fields`) и составные
  преобразования (`composites`: разбор местоположения, разбор юнита в
  стойке, суммарный объём памяти, словарь статусов, склейка описания).
- `convertblade.go` — аналог `convert.go` для блейд-корзин (`ConvertRowBlade`)
  по правилам из `mappingblade.yaml`/`rack_blade.yaml`. Отдельный файл, а не
  расширение `convert.go` — `setTarget`/`applyComposite` там жёстко
  завязаны на `ServerSprRecords`/`ServerHardware`, а для блейд-корзин нужна
  ровно одна целевая структура (`BladeSystems`); композитов меньше
  (`memory_total`/`enum_multi` не нужны — у `BladeSystems` нет полей
  `MemoryMb`/`Active`/`Stat`).
- `reader.go` — чтение исходного `.xlsx` (`ReadRows` — общая для обеих
  связок; `ConvertFile`/`ConvertFileBlade` — по одной на каждую), через
  `excelize` (уже используется в проекте).
- `importrackservers.go` — запись в БД для rack-серверов
  (`ImportItiliumServer`, вызывается из
  `controllers/adminimportrackservers.go` → `/admin/importrackservers`).
- `importblades.go` — запись в БД для блейд-корзин (`ImportItiliumBlade`,
  вызывается из `controllers/adminimportblades.go` →
  `/admin/importblades`). Отличие от импорта серверов: поиск существующей
  записи `BladeSystems` ведётся **по имени** (`Name`, короткое
  мнемоническое имя вида `C7000-01`, `C7000-02`), а не по серийному номеру
  — серийники шасси в Итиллиум заполняются не так надёжно, как у серверов,
  а имя — устойчивый заранее согласованный идентификатор.
- `importbladeservers.go` — запись в БД для блейд-серверов
  (`ImportItiliumBladeServer`, вызывается из
  `controllers/adminimportbladeservers.go` → `/admin/importbladeservers`).
  Использует ТОТ ЖЕ код find-or-create `DatacenterSite`/`RackCabinet`, что и
  `ImportItiliumServer` ("аналогично обычным серверам"), плюс:
  `ServerHardware.FormFactor` всегда = 2 (блейд); ДОПОЛНИТЕЛЬНО ищет
  `BladeSystems` по имени из `extra.blade_ref_name` (см.
  `mappingbladeservers.yaml`) и при находке проставляет
  `ServerHardware.BladeSystemId` и переопределяет `RackCabinetId` значением
  из найденной корзины (её размещение авторитетнее, чем повторный разбор
  "Местоположение" самого сервера — тем более что колонки "Размещение в
  шкафу/стойке" в этой выгрузке нет вовсе). Если корзина не найдена -
  предупреждение в лог, `RackCabinetId` остаётся тем, что дал стандартный
  разбор места установки (может быть 0, если "Местоположение" тоже не
  разобралось в стойку).

## Использование

```go
cfg, err := itilliumimport.LoadMapping("itilliumimport/mapping.yaml")
if err != nil { /* ... */ }

results, err := itilliumimport.ConvertFile(cfg, "Серверы_templ.xlsx")
if err != nil { /* ... */ }

for _, r := range results {
    // r.Spr  - models.ServerSprRecords (частично заполнена)
    // r.Hw   - models.ServerHardware (частично заполнена)
    // r.Extra    - map[string]string: поля без прямого соответствия в
    //              схеме (см. ниже) + диагностические "сырые" значения
    // r.Warnings - []string: на что стоит посмотреть глазами перед сохранением

    for _, w := range r.Warnings {
        log.Println("itilliumimport:", w)
    }
    // ... здесь: найти/создать HWSprId<->ServerId связку, o.Insert/o.Update ...
}
```

Пакет **не пишет в БД сам** — только строит и возвращает `Result` со
структурами `ServerSprRecords`/`ServerHardware` и картой `Extra`. Запись в
БД (создание/обновление записей, разрешение конфликтов по `S/N`/hostname,
проставление `ServerHardware.ServerId`↔`ServerSprRecords.HWSprId`,
`SourceUpdateType = models.SourceUpdateManual` или отдельный код для
"импорт из Итиллиум", если такой заведёте) — отдельный шаг, который нужно
написать под конкретный сценарий использования (разовый импорт CLI-тулом,
периодическая сверка, страница в админке и т.п.) — за пределами этого
пакета, т.к. это уже бизнес-решения о поведении при конфликтах, а не о
разборе формата файла.

## Что осталось в `Extra` и почему

Часть колонок Итиллиум не имеет однозначного поля в текущей схеме
`ServerSprRecords`/`ServerHardware` — они не отбрасываются, а кладутся в
`Result.Extra` (карта `ключ → текст`) на ручной разбор/донастройку:

| Ключ | Источник | Почему не смаплено напрямую |
|---|---|---|
| `normative_model` | "Марка/модель(Норматив)" | нет отдельного поля под нормативное наименование конфигурации (частично дублирует `ServerHardware.Model`) |
| `legal_entity` | "Юр.Лицо" | в схеме это FK `OwnerFromFGId`/`CounterpartyId` (числовые ссылки на справочники `MembersFG`/`Counterparty`), а не текст — нужен отдельный поиск по названию |
| `repair_date` | "Дата ремонта" | нет поля под историю ремонтов |
| `hdd_1`/`hdd_2` | "HDD"/"HDD 2" | компонентный уровень (`models.ServerHardwareComponents`, `Type` = диск), не отдельное поле `ServerHardware` |
| `raid_1`/`raid_2` | "RAID"/"RAID(2)" | то же — либо `ServerHardwareComponents`, либо новое поле |
| `ipmi_2` | "IPMI(2)" | `ServerHardware.MpIPAddress` — одно поле, вторичный IPMI/BMC девать некуда |
| `motherboard`, `network_adapter`, `videocard` | соотв. колонки | нет отдельных полей — либо `ServerHardwareComponents`, либо новые поля |
| `author_name` | "Автор" | это ФИО, а `ServerSprRecords.Id_User`/`ServerHardware.Id_User` — числовой FK на `models.AuthUser`; нужен поиск пользователя по имени (неоднозначная операция — ФИО не гарантированно уникальны) |
| `equipment_config` | "Комплектация" | нет отдельного поля |
| `status_raw` | "Статус" | заполняется, только если встретится значение, отсутствующее в словаре `enums.status` — сигнал дополнить словарь в `mapping.yaml` |
| `location_zone_code`, `location_room_code` | "Местоположение" (2-я и 3-я часть через `\|`) | внутренние коды корпуса/зала Итиллиум — для сопоставления со справочником `models.RackCabinet`/`models.DatacenterSite` (см. `kontext/03-data-model.md`, раздел 5a) пока нет автоматического матчинга по имени |

## Известные допущения (проверьте перед боевым запуском)

- "Стар.инв.№" сопоставлен с основным `ServerHardware.InventoryNum` (в
  примере там осмысленные коды вида `XXX_S_111`, а не "старый номер" в
  буквальном смысле) — если в вашем Итиллиум есть отдельная колонка
  актуального инвентарного номера, перенаправьте её в `mapping.yaml`.
- `ServerSprRecords.Name` и `NameFQDN` заполняются ОДНИМ и тем же значением
  колонки "Имя (hostname)" — в примере отдельного FQDN нет.
- "Поставщик" → `ServerHardware.Counterparty` (свободнотекстовое поле
  "контрагент") — проверьте по смыслу, в оригинальном Итиллиум "Поставщик"
  может означать не то же самое, что "контрагент" в loadmon.
- "Дата документа закупки" → `ServerHardware.DateContractDoc` — в схеме это
  "дата документа" для `ContractDoc` (номер договора), а не обязательно
  дата накладной/счёта.
- Словарь статусов (`enums.status` в `mapping.yaml`) заполнен по видимым в
  примере значениям ("В работе (эксплуатации)", "Выведен", "Списан",
  "Свободно", "На складе", пустое) — числа `Active`/`Stat` подобраны по
  смыслу по комментариям в `models/models_add.go`, но не проверялись живыми
  данными — пересмотрите при первом реальном запуске (см. `Extra.status_raw`
  для необработанных значений).
