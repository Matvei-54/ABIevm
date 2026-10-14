Локальный запуск ABIevm со StakeDAO

Установите Go 1.27.2 (версия, на которой проверялся этот код).
Распакуйте архив и откройте терминал в папке ABIevm.

Windows PowerShell:
$env:STAKEDAO_ETHEREUM_RPC_URL = "https://ethereum-rpc.publicnode.com"
go run -mod=readonly ./cmd

Linux / macOS:
export STAKEDAO_ETHEREUM_RPC_URL="https://ethereum-rpc.publicnode.com"
go run -mod=readonly ./cmd

Сервер слушает порт 8081.
Проверка Ethereum из другого терминала:
curl --max-time 330 http://localhost:8081/reservesStakedao/ethereum

Настройка переменной нужна в том же терминале, где запускается приложение.
Автоматического чтения .env нет. При первом запуске Go скачает зависимости.
Первый запрос Ethereum сканирует исторические регистрации: он может быть долгим
и делать много RPC-запросов. Если хотите сначала проверить только запуск без RPC:
curl http://localhost:8081/reservesStakedao/base
Ожидается HTTP 400: поддерживается только ethereum.

Офлайн по отношению к блокчейну проверки:
go test -mod=readonly -p 3 ./...
Тесты используют локальные фикстуры, не бесплатные RPC.

Статус шага 5:
Расчет reward APR для Ethereum Curve V2 и extra rewards реализован.
Mainnet-значения еще не подтверждены: исторические запросы из облака получали 403.
Локальный доступ PublicNode может отличаться; если исторические методы тоже
недоступны, нужен другой RPC с поддержкой исторического состояния и eth_getLogs.
APY остается null; Balancer primary rewards пока не поддерживаются.
Описание модели: internal/service/stakedao/README.md.

Архив содержит текущие исходники, ABI, документацию и тесты, включая новые файлы.
История Git, инструменты облака, бинарники, кэши и заметки/логи в архив не включены.
Изменения опубликованы в отдельной ветке GitHub; main не изменяется.
