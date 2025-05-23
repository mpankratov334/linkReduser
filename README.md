# Учебный проект: Сервис сокращения URL

### Содержит в себе пример реализации

## Сетевое приложение на Golang
- **На основе пакета `net/http`**  
  Сделан HTTP-сервер с маршрутизацией и обработкой запросов.  
- **С использованием стандартного тестового фреймворка Go**  
  Написаны юнит-тесты сервера.  
- **С использованием пакетов `flag` и `os`**  
  Настроена конфигурация сервера через аргументы командной строки и переменные окружения.
- **Через пакет `os`**
 Конфигурация поддерживает обработку данных в файлах, вместо базы данных
- **При помощи `logrus`**  
  Реализовано логирование запросов через *Middleware*.  
- **В рамках REST API применяется `encoding/json`**
	Сервис поддерживает JSON запросы
- **Пакет `compress/gzip`**
	Оптимизирует передачу тяжелых данных через *Midleware*

## Хранение в базе данных
- **С использованием пакетов `database/sql` , `context`, `pgx`**  
  Cервис интегрирован с PostgreSQL: batch-транзакция, многопоточное асинхронное удаление и основные функции БД сервиса.
- **С использованием пакета `errors`**  
  Обработаны возможные ошибки при работе приложения, в т.ч. конфликты sql БД.  
- **Через pgAdmin и SQL-запросы**
	Созданы и индексированы таблицы 	

## Безопасность
- **С использованием пакетов `crypto`, `hash` и JWT**  
  Реализована аутентификация пользователей как *Midleware* обработка *cookie*.  
## Многопоточность
- **С использованием `sync.Mutex` и каналов**  
  Оптимизирована работа БД использованием batch-update  

