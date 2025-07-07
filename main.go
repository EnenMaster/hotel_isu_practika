package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"gohotel/internal/auth"
	"gohotel/internal/logging"

	_ "github.com/joho/godotenv/autoload" 
	"github.com/gorilla/csrf"
	_ "github.com/lib/pq"

	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

var db *sql.DB

func main() {
	//--------------------------------------------------------------------
	// 0. Логгер
	//--------------------------------------------------------------------
	logging.Init()
	logging.L.Info("app_start")

	//--------------------------------------------------------------------
	// 1. Подключение к Postgres
	//--------------------------------------------------------------------
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=postgres password=postgres dbname=hotel_db sslmode=disable"
	}
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Не удалось подключиться к БД:", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal("База недоступна:", err)
	}

	//--------------------------------------------------------------------
	// 2.  Rate-limit: 5 POST-попыток / минуту на /login_post
	//--------------------------------------------------------------------
	rate, _ := limiter.NewRateFromFormatted("5-M")
	loginLimiter := stdlib.NewMiddleware(
		limiter.New(memory.NewStore(), rate),
		stdlib.WithErrorHandler(func(w http.ResponseWriter, _ *http.Request, _ error) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Слишком много попыток, попробуйте позже"))
		}),
	)

	//--------------------------------------------------------------------
	// 3.  CSRF-middleware
	//--------------------------------------------------------------------
	csrfKey := os.Getenv("CSRF_AUTH_KEY")          
if len(csrfKey) == 0 {
    log.Fatal("CSRF_AUTH_KEY не задан в .env") 
}

csrfProt := csrf.Protect(
    []byte(csrfKey),
    csrf.Secure(false),
    csrf.Path("/"),
    csrf.TrustedOrigins([]string{
        "localhost:8080",
        "127.0.0.1:8080",
        // если заходите с другого ПК в сети:
        "192.168.0.50:8080",
        // если фронтенд/SPA на 3000:
        "localhost:3000",
    }),
)



	//--------------------------------------------------------------------
	// 4.  Роутер
	//--------------------------------------------------------------------
	mux := http.NewServeMux()
	

	// ------- публичные -------
	mux.Handle("/login", loginLimiter.Handler(http.HandlerFunc(loginHandler)))
	mux.Handle("/login_post", loginLimiter.Handler(http.HandlerFunc(loginHandler))) // POST

	mux.HandleFunc("/logout", logoutHandler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// ------- авторизованные -------
	authOnly := auth.RequireAuth
	mux.Handle("/rooms",           authOnly(http.HandlerFunc(roomsListHandler)))
	mux.Handle("/rooms/add",       authOnly(http.HandlerFunc(roomAddHandler)))
	mux.Handle("/rooms/edit/",     authOnly(http.HandlerFunc(roomEditHandler)))
	mux.Handle("/rooms/delete/",   authOnly(http.HandlerFunc(roomDeleteHandler)))

	mux.Handle("/guests",          authOnly(http.HandlerFunc(guestsListHandler)))
	mux.HandleFunc("/guests/add",  guestAddHandler)  
	mux.Handle("/guests/edit/",    authOnly(http.HandlerFunc(guestEditHandler)))
	mux.Handle("/guests/delete/",  authOnly(http.HandlerFunc(guestDeleteHandler)))

	mux.Handle("/bookings",         authOnly(http.HandlerFunc(bookingsListHandler)))
	mux.Handle("/bookings/add",     authOnly(http.HandlerFunc(bookingAddHandler)))
	mux.Handle("/bookings/cancel/", authOnly(http.HandlerFunc(bookingCancelHandler)))

	// ------- только админ -------
	adminOnly := auth.RequireRole("admin")
	mux.Handle("/admin", adminOnly(http.HandlerFunc(adminPanelHandler)))
	mux.Handle("/users", adminOnly(http.HandlerFunc(usersHandler)))

	//--------------------------------------------------------------------
	// 5.  Оборачиваем лог-middleware и запускаем сервер
	//--------------------------------------------------------------------
	root := logging.Access(mux)

	log.Println("server started at :8080")
	log.Fatal(http.ListenAndServe(
    ":8080",
    csrfProt(root),   // <- оборачиваем root через csrfProt
))
}
