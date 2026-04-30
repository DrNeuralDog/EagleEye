<p align="left"><img src="resources/logo/Logo_Optimal_Gradient.png" alt="EagleEye Logo" width="360" /></p>

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://go.dev) [![Fyne](https://img.shields.io/badge/Fyne-2.7+-00ACD7.svg)](https://fyne.io) [![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-brightgreen.svg)]() [![Release](https://img.shields.io/github/v/release/DrNeuralDog/EagleEye?include_prereleases&sort=semver)](https://github.com/DrNeuralDog/EagleEye/releases)

## Проблема и решение

Если постоянно залипать в экран на много часов подряд, глаза быстро начинают уставать: появляется сухость, тяжесть, расфокус, а нормальные перерывы легко забываются. Именно из этой боли и появился **EagleEye** - небольшая кроссплатформенная утилита, которая живет в системном трее и мягко возвращает внимание к здоровью глаз. А существующие на рынке альтернативы уже морально устарели и не обладают всеми требуемыми характеристиками: либо это тяжелые корпоративные решения, либо заброшенные pet-проекты без нормальной кроссплатформенности, idle tracking и свежего UI.

Приложение напоминает о коротких паузах, показывает оверлей с анимированным соколом и предлагает простые упражнения: посмотреть влево-вправо, вверх-вниз, поморгать или перевести взгляд вдаль. По личному опыту автора, такой режим помог работать примерно на **20% дольше без дискомфорта в глазах** и чаще делать мини-зарядку днем.

| До EagleEye | После EagleEye |
| --- | --- |
| Работа идет часами без пауз | Перерывы появляются по расписанию |
| Глаза устают, но отвлечься забываешь | Оверлей явно напоминает, что пора дать глазам отдых |
| Зарядка для глаз остается "на потом" | Сокол показывает конкретное упражнение |
| Нет единого управления перерывами | Все управляется из системного трея |

### Сравнение до и после

![Сравнение комфорта работы до и после EagleEye](resources/Designs/EyeHealthComparison.gif)

> Важно: EagleEye не является медицинским продуктом и не заменяет рекомендации врача. Это практичная утилита для регулярных пауз и снижения бытовой нагрузки от долгой работы за экраном.

## Почему EagleEye полезен

- **Короткие перерывы:** по умолчанию каждые 15 минут на 15 секунд.
- **Длинные перерывы:** по умолчанию каждые 50 минут на 5 минут.
- **Анимированный оверлей:** сокол показывает упражнение и таймер оставшегося отдыха.
- **Strict mode:** режим без быстрого пропуска, если нужно дисциплинированно соблюдать отдых.
- **Idle tracking:** удобная автоматизация - не нужно вручную ставить таймер на паузу, когда отходишь от ПК. Иначе получается неприятность: только сел за комп - и сразу прилетает разминка. EagleEye сам видит, что пользователя не было 5+ минут, считает это отдыхом и запускает отсчет заново.
- **Системный трей:** статус, пауза, принудительный следующий перерыв, длинный перерыв, временное отключение напоминаний и выход.
- **Автозапуск:** поддержка Windows Registry Run Key, Linux autostart desktop entry и macOS LaunchAgent.
- **Локальные настройки:** YAML-файл в стандартной пользовательской config-директории ОС.
- **RU/EN локализация:** язык можно переключить в окне настроек.

## Основные сценарии

**Обычный рабочий день:** запустил приложение, нажал Start, свернул настройки и работаешь дальше. EagleEye остается в трее и показывает, сколько осталось до следующего перерыва.

**Короткая разминка:** когда наступает короткий перерыв, появляется компактный или полноэкранный оверлей с упражнением для глаз и обратным отсчетом.

**Длинный отдых:** после более длинного рабочего отрезка приложение предлагает расслабить взгляд и посмотреть вдаль.

**Контроль из трея:** можно поставить таймер на паузу, отключить напоминания на 5/15/30/60 минут, начать следующий перерыв сразу или открыть настройки.

## Принципы проекта

- **Set and forget:** приложение должно помогать в фоне, а не становиться еще одним источником шума.
- **Локальность:** без серверов, баз данных и внешних аккаунтов.
- **Тестируемое ядро:** расписание перерывов отделено от GUI.
- **Кроссплатформенность:** platform-specific код изолирован в отдельных файлах с build tags.
- **Безопасные настройки:** конфигурация и служебные файлы хранятся в пользовательской config-директории с ограниченными правами (Windows: `%AppData%\EagleEye\settings.yaml`, Linux: `~/.config/EagleEye/settings.yaml` или `$XDG_CONFIG_HOME/EagleEye/settings.yaml`, macOS: `~/Library/Application Support/EagleEye/settings.yaml`).

## Технические детали

EagleEye написан на Go и Fyne. Внутри используется чистая state machine для расписания перерывов, а UI и платформенные интеграции вынесены в отдельные слои.

- **`cmd/main.go`** - тонкая точка входа, которая вызывает `internal/app.Run`.
- **`internal/app`** - runtime orchestration: связывает настройки, таймер, tray, overlay, анимации и платформенные сервисы.
- **`internal/core/timekeeper`** - состояние рабочего времени, коротких/длинных перерывов, паузы и progress-событий.
- **`internal/ui/preferences`** - окно настроек Fyne.
- **`internal/ui/tray`** - системный tray-менеджер и команды управления.
- **`internal/ui/overlay`** - окно перерыва с таймером, прозрачностью, fullscreen-режимом и topmost-поведением.
- **`internal/ui/animation`** - логика смены sprites для упражнений.
- **`internal/storage`** - загрузка и сохранение `settings.yaml`.
- **`internal/platform`** - single instance, autostart и idle detection для разных ОС.
- **`resources`** - встроенные логотипы и sprites через Go `embed`.

## Архитектура приложения

```mermaid
flowchart TD
    Main["cmd/main.go"] --> Run["internal/app.Run"]

    subgraph App["internal/app"]
        Controller["AppController<br/>runtime orchestration"]
        State["appState<br/>service started / paused / next break"]
        Logger["JSON logger"]
    end

    subgraph Core["internal/core"]
        Keeper["TimeKeeper<br/>state machine"]
        Events["Events<br/>state_change / progress / idle_reset / idle_error"]
        Model["TimeKeeperConfig"]
    end

    subgraph UI["internal/ui"]
        Prefs["PreferencesWindow<br/>settings + service status"]
        Tray["TrayManager<br/>system tray menu"]
        Overlay["OverlayWindow<br/>break screen + timer"]
        Animation["Animation Engine<br/>falcon exercises"]
        I18N["Localizer<br/>RU / EN"]
    end

    subgraph Infra["Infrastructure"]
        Storage["storage<br/>settings.yaml"]
        Platform["platform<br/>autostart / idle / single instance"]
        Resources["resources<br/>embedded logo + sprites"]
    end

    Run --> Controller
    Controller --> State
    Controller --> Logger
    Controller --> Keeper
    Controller --> Prefs
    Controller --> Tray
    Controller --> Overlay
    Controller --> Storage
    Controller --> Platform
    Controller --> I18N

    Prefs -->|"update config / OnSave"| Controller
    Controller -->|"save settings"| Storage
    Tray -->|"pause / force break / quit"| Controller
    Keeper -->|"emit"| Events
    Events -->|"consume"| Controller
    Controller -->|"show / hide / progress"| Overlay
    Controller -->|"status + menu state"| Tray
    Overlay --> Animation
    Animation -->|"sprite updates"| Overlay
    Resources --> Animation
    Resources --> Controller
    Platform -->|"idle duration"| Keeper
    Platform -->|"single instance / autostart"| Controller
    Model --> Keeper
```

## Пользовательский сценарий

```mermaid
flowchart TD
    Start(["Пользователь запускает EagleEye"]) --> Single{"Уже запущен другой экземпляр?"}
    Single -->|"Да"| Activate["Активировать существующее окно настроек"]
    Single -->|"Нет"| Load["Загрузить settings.yaml<br/>или взять значения по умолчанию"]

    Load --> First{"Таймер должен стартовать сразу?"}
    First -->|"Первый запуск / ручной старт"| Prefs["Окно настроек"]
    First -->|"Автозапуск и таймер был включен"| Background["Работа в фоне через трей"]

    Prefs --> Save["Сохранить настройки"]
    Save --> StartTimer["Старт TimeKeeper"]
    StartTimer --> Background

    Background --> Work["Пользователь работает за экраном"]
    Work --> Idle{"Пользователь отошел на 5+ минут?"}
    Idle -->|"Да"| Reset["Сбросить отсчет до перерыва"]
    Idle -->|"Нет"| NextBreak{"Какой перерыв наступил?"}
    Reset --> Work

    NextBreak -->|"Короткий"| Short["Оверлей с упражнением<br/>влево-вправо / вверх-вниз / мигание"]
    NextBreak -->|"Длинный"| Long["Оверлей с отдыхом<br/>посмотреть вдаль"]

    Short --> Strict{"Strict mode включен?"}
    Long --> Strict
    Strict -->|"Да"| NoSkip["Кнопка Skip скрыта<br/>перерыв нужно пройти"]
    Strict -->|"Нет"| CanSkip["Можно пропустить через Skip"]

    NoSkip --> Done["Перерыв завершен"]
    CanSkip --> Done
    Done --> Work

    Background --> TrayActions{"Действие в трее"}
    TrayActions -->|"Pause / Resume"| Pause["Пауза или продолжение таймера"]
    TrayActions -->|"Start next break now"| Short
    TrayActions -->|"Take long break now"| Long
    TrayActions -->|"Disable breaks for..."| Delay["Пауза на 5 / 15 / 30 / 60 минут"]
    TrayActions -->|"Preferences"| Prefs
    TrayActions -->|"Quit"| End(["Выход"])

    Pause --> Work
    Delay --> Work
```

## Установка

Самый простой способ — скачать готовый бинарник со страницы релизов. Никаких Go, компиляторов и зависимостей ставить не нужно.

**➡️ [Скачать последний релиз с GitHub Releases](https://github.com/DrNeuralDog/EagleEye/releases/latest)**

| ОС | Файл | Что делать после скачивания |
| --- | --- | --- |
| **Windows x64** | `EagleEye_windows_amd64.zip` | Распаковать, запустить `EagleEye.exe`. При первом запуске Windows SmartScreen может показать предупреждение «Windows protected your PC» — нажать **More info → Run anyway** (приложение не подписано code-signing сертификатом, это нормально для open source). |
| **Linux x64** | `EagleEye_linux_amd64.tar.gz` | `tar -xzf EagleEye_linux_amd64.tar.gz && ./eagleeye`. Нужны системные библиотеки OpenGL: `sudo apt install libgl1 libxxf86vm1` (Debian/Ubuntu). |
| **macOS Intel** | `EagleEye_darwin_amd64.tar.gz` | `tar -xzf EagleEye_darwin_amd64.tar.gz && ./EagleEye`. Если Gatekeeper ругается — ПКМ по бинарнику → **Открыть** → подтвердить. |
| **macOS Apple Silicon (M1/M2/M3)** | `EagleEye_darwin_arm64.tar.gz` | Аналогично Intel-версии. |

Файл `checksums.txt` в том же релизе содержит SHA-256 каждого архива — можно проверить целостность через `sha256sum -c checksums.txt` (Linux/macOS) или `Get-FileHash` (Windows PowerShell).

Если нужно собрать из исходников вручную — см. раздел «Сборка» ниже.

## Сборка

### Требования

- Go 1.21+
- Fyne v2.7+
- Для Linux: системные зависимости Fyne/OpenGL, например `libgl1-mesa-dev` и `xorg-dev`
- Для Windows-сборки с иконкой: PowerShell и `rsrc.exe` (скрипт может установить его при запуске с `-AllowGoNetwork`)

### Windows

```powershell
# Обычная сборка
go mod tidy
go build -o bin/EagleEye.exe ./cmd

# Сборка Windows GUI exe с иконкой
powershell -ExecutionPolicy Bypass -File .\build_with_icon.ps1

# Если rsrc.exe еще не установлен
powershell -ExecutionPolicy Bypass -File .\build_with_icon.ps1 -AllowGoNetwork

# Запуск
.\bin\EagleEye.exe
```

### Linux

```bash
# Пример для Debian/Ubuntu
sudo apt install libgl1-mesa-dev xorg-dev

go mod tidy
go build -o bin/eagleeye ./cmd
./bin/eagleeye
```

### macOS

```bash
go mod tidy
go build -o bin/EagleEye ./cmd
./bin/EagleEye
```

### Кроссплатформенная сборка

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o bin/EagleEye.exe ./cmd

# Linux
GOOS=linux GOARCH=amd64 go build -o bin/eagleeye-linux ./cmd

# macOS
GOOS=darwin GOARCH=amd64 go build -o bin/eagleeye-macos ./cmd
```

### Релизный пайплайн

CI-пайплайн `.github/workflows/release.yml` собирает бинарники под Windows / Linux / macOS (Intel + Apple Silicon) на нативных GitHub-раннерах, генерирует `checksums.txt` и публикует в GitHub Releases автоматически при пуше git-тега вида `v*` (например, `v0.1.0`). Локально для быстрой snapshot-проверки можно использовать GoReleaser:

```bash
# Snapshot-сборка текущей платформы (без публикации)
goreleaser build --snapshot --clean --single-target
```

## Проверка

```bash
# Все тесты
go test ./...

# Статическая проверка стандартным Go-инструментом
go vet ./...

# Проверка сборки
go build ./cmd/...

# Если установлен golangci-lint
golangci-lint run ./...
```

## Планы дальнейшего развития

EagleEye задумывался как простая утилита для глаз, но направление движения шире - превратить его в удобную персональную платформу для цифрового здоровья и продуктивности.

- **Мобильные версии:** порт на Android и iOS, чтобы напоминания о зарядке для глаз и перерывах работали и вне десктопа.
- **Платформа для мониторинга здоровья:** расширение от "таймера перерывов" до полноценного трекера цифрового благополучия - осанка, количество активного экранного времени, баланс работы и отдыха.
- **Тесты усталости глаз:** короткие встроенные проверки зрения и фокусировки (контрастность, острота, аккомодация), чтобы пользователь мог видеть динамику.
- **Тесты умственной усталости:** простые когнитивные мини-задания (reaction time, рабочая память, внимание), по которым можно отследить, когда пора закругляться.
- **Трекеры поведения:** сколько разминок за день было выполнено, а сколько скипнуто, статистика по дням и неделям, тренды.
- **Социальные интеграции:** возможность делиться прогрессом с друзьями и коллегами - меньше усталости глаз, больше успешных разминок, более ровный рабочий график.
- **Экспорт данных:** выгрузка локальной статистики в CSV/JSON для тех, кто хочет анализировать себя вручную или интегрировать с другими health-трекерами.

## Связаться 📫

Email: neural_dog@proton.me

---

*EagleEye built with Go and Fyne. Маленькая утилита, которая вовремя напоминает: глаза тоже часть рабочего процесса.*
