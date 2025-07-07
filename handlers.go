package main

import (
	"database/sql"
	"log"
     "github.com/gorilla/csrf"
	"net/http"
	"strconv"
	"strings"
	"time"

	
	"golang.org/x/crypto/bcrypt"

	"gohotel/internal/auth"
)

// ---------- НОМЕРА ----------------------------------------------------

func roomsListHandler(w http.ResponseWriter, r *http.Request) {
    rooms, err := GetRooms()
    if err != nil {
        http.Error(w, "Ошибка получения номеров", 500); return
    }

    data := map[string]interface{}{
        "Rooms":   rooms,
        "IsAdmin": auth.Is(r, "admin"),  
    }

    renderTemplate(w, r, data,
        "templates/base.html",
        "templates/rooms.html")
}

func roomAddHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderTemplate(w, r, map[string]interface{}{"Room": Room{}},
			"templates/base.html", "templates/room_edit.html")
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ошибка формы", 400)
		return
	}
	roomType := r.FormValue("RoomType")
	status   := r.FormValue("Status")
	price, _ := strconv.ParseFloat(r.FormValue("Price"), 64)
	category := r.FormValue("Category")

	_, err := db.Exec(`INSERT INTO rooms (room_type, status, price, category)
	                   VALUES ($1, $2, $3, $4)`,
		roomType, status, price, category)
	if err != nil {
		http.Error(w, "Ошибка при добавлении номера", 500)
		return
	}
	http.Redirect(w, r, "/rooms", http.StatusSeeOther)
}

func roomEditHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/rooms/edit/"))

	if r.Method == http.MethodGet {
		room, err := GetRoomByID(id)
		if err != nil {
			http.Error(w, "Номер не найден", 404)
			return
		}
		renderTemplate(w, r, map[string]interface{}{"Room": room},
			"templates/base.html", "templates/room_edit.html")
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ошибка формы", 400)
		return
	}
	room := Room{
		RoomID:   id,
		RoomType: r.FormValue("RoomType"),
		Status:   r.FormValue("Status"),
		Price:    func() float64 { p, _ := strconv.ParseFloat(r.FormValue("Price"), 64); return p }(),
		Category: r.FormValue("Category"),
	}
	if err := UpdateRoom(room); err != nil {
		http.Error(w, "Ошибка обновления", 500)
		return
	}
	http.Redirect(w, r, "/rooms", http.StatusSeeOther)
}

func roomDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/rooms/delete/"))

	
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE room_id=$1`, id).Scan(&cnt); err != nil {
		http.Error(w, "Ошибка проверки бронирований", 500)
		return
	}
	if cnt > 0 {
		http.Error(w, "Нельзя удалить номер: есть активные бронирования", 400)
		return
	}
	if err := DeleteRoom(id); err != nil {
		http.Error(w, "Ошибка удаления", 500)
		return
	}
	http.Redirect(w, r, "/rooms", http.StatusSeeOther)
}

// ---------- ГОСТИ ----------------------------------------------------

func guestsListHandler(w http.ResponseWriter, r *http.Request) {
    guests, err := GetGuests()
    if err != nil {
        http.Error(w, "Ошибка получения гостей", 500); return
    }

    data := map[string]interface{}{
        "Guests":  guests,
        "IsAdmin": auth.Is(r, "admin"),
    }

    renderTemplate(w, r, data,
        "templates/base.html",
        "templates/guests.html")
}

func guestAddHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderTemplate(w, r, map[string]interface{}{"Guest": Guest{}},
			"templates/base.html", "templates/guest_edit.html")
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ошибка формы", 400)
		return
	}
	g := Guest{
		Name:  r.FormValue("Name"),
		Email: r.FormValue("Email"),
		Phone: r.FormValue("Phone"),
	}
	if _, err := AddGuest(g); err != nil {
		http.Error(w, "Ошибка добавления", 500)
		return
	}
	http.Redirect(w, r, "/guests", http.StatusSeeOther)
}

func guestEditHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/guests/edit/"))

	if r.Method == http.MethodGet {
		guest, err := GetGuestByID(id)
		if err != nil {
			http.Error(w, "Гость не найден", 404)
			return
		}
		renderTemplate(w, r, map[string]interface{}{"Guest": guest},
			"templates/base.html", "templates/guest_edit.html")
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ошибка формы", 400)
		return
	}
	guest := Guest{
		GuestID: id,
		Name:    r.FormValue("Name"),
		Email:   r.FormValue("Email"),
		Phone:   r.FormValue("Phone"),
	}
	if err := UpdateGuest(guest); err != nil {
		http.Error(w, "Ошибка обновления", 500)
		return
	}
	http.Redirect(w, r, "/guests", http.StatusSeeOther)
}

func guestDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/guests/delete/"))
	if err := DeleteGuest(id); err != nil {
		http.Error(w, "Ошибка удаления: "+err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/guests", http.StatusSeeOther)
}

// ---------- БРОНИРОВАНИЯ ---------------------------------------------

func bookingAddHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodGet {
        guests, _ := GetGuests()
        rooms, _  := GetRooms()

        // Инициализируем «сегодня» и «завтра»
        now := time.Now()
        newBooking := Booking{
            CheckIn:  now,
            CheckOut: now.Add(24 * time.Hour),
        }

        renderTemplate(w, r, map[string]interface{}{
            "Booking":   newBooking,
            "Guests":    guests,
            "Rooms":     rooms,
            "csrfField": csrf.TemplateField(r), 
        }, "templates/base.html", "templates/booking_edit.html")
        return
    }

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ошибка формы", 400)
		return
	}
	guestID, _ := strconv.Atoi(r.FormValue("GuestID"))
	roomID,  _ := strconv.Atoi(r.FormValue("RoomID"))
	in, _     := time.Parse("2006-01-02", r.FormValue("CheckIn"))
	out,_     := time.Parse("2006-01-02", r.FormValue("CheckOut"))

	b := Booking{
		RoomID: roomID, GuestID: guestID,
		CheckIn: in, CheckOut: out,
		Paid: false, Status: "created",
	}
	if err := AddBooking(b); err != nil {
		http.Error(w, "Ошибка добавления бронирования", 500)
		return
	}
	http.Redirect(w, r, "/bookings", http.StatusSeeOther)
}

func bookingCancelHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/bookings/cancel/"))
	if err := CancelBooking(id); err != nil {
		http.Error(w, "Ошибка отмены", 500)
		return
	}
	http.Redirect(w, r, "/bookings", http.StatusSeeOther)
}

func bookingMarkPaidHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/bookings/pay/"))
	if err := MarkBookingPaid(id); err != nil {
		http.Error(w, "Ошибка оплаты", 500)
		return
	}
	http.Redirect(w, r, "/bookings", http.StatusSeeOther)
}

func bookingsListHandler(w http.ResponseWriter, r *http.Request) {

    // 1. создаём объект-фильтр
    filter := BookingFilter{}

    if v := r.URL.Query().Get("CheckIn");  v != "" { filter.CheckIn,  _ = time.Parse("2006-01-02", v) }
    if v := r.URL.Query().Get("CheckOut"); v != "" { filter.CheckOut, _ = time.Parse("2006-01-02", v) }
    if v := r.URL.Query().Get("RoomID");   v != "" { filter.RoomID,   _ = strconv.Atoi(v) }
    if v := r.URL.Query().Get("status");   v != "" { filter.Status     = v }

    // 2. получаем данные
    bookings, err := GetAllBookingsFiltered(filter)
    if err != nil {
        http.Error(w, "Ошибка БД", 500)
        return
    }

    // 3. передаём их в шаблон
    data := map[string]interface{}{
        "Bookings": bookings,
        "Filter":   filter,                 
        "IsAdmin":  auth.Is(r, "admin"),
    }

    renderTemplate(w, r, data,
        "templates/base.html",
        "templates/bookings.html")
}

// ---------- ЛОГИН / ЛОГАУТ --------------------------------------------

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { // GET → форма
		loginForm(w, r, "")
		return
	}

	email    := r.FormValue("email")
	password := r.FormValue("password")

	user, err := GetUserByEmail(email)
	if err == sql.ErrNoRows {
		loginForm(w, r, "Пользователь не найден")
		return
	} else if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		loginForm(w, r, "Неверный пароль")
		return
	}

	if err = auth.Set(w, auth.Session{UserID: user.UserID, Role: user.Role}); err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if user.Role == "admin" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/bookings", http.StatusSeeOther)
}

// GET /login — показать форму
func loginPageHandler(w http.ResponseWriter, r *http.Request) {
    data := map[string]interface{}{
        "Message":   "",                 // здесь будет текст ошибки
        "csrfField": csrf.TemplateField(r),
    }
    renderTemplate(w, r, data,
        "templates/base.html",
        "templates/login.html")
}


func logoutHandler(w http.ResponseWriter, r *http.Request) {
    // 1. стираем подпись + данные
    _ = auth.Set(w, auth.Session{})

    // 2. необязательный аудит
    if u, ok := auth.Get(r); ok {
        log.Printf("Logout: userID=%d role=%s", u.UserID, u.Role)
    }

    // 3. редирект
    http.Redirect(w, r, "/login", http.StatusSeeOther)
}


// ---------- ПАНЕЛЬ АДМИНА --------------------------------------------

func adminPanelHandler(w http.ResponseWriter, r *http.Request) {
    renderTemplate(w, r,
        map[string]interface{}{"IsAdmin": true},   // всегда админ
        "templates/base.html",
        "templates/admin_panel.html")
}


// ---------- СПИСОК ПОЛЬЗОВАТЕЛЕЙ -------------------------------------

func usersHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query(`SELECT user_id, email, role FROM users ORDER BY user_id`)
    if err != nil { http.Error(w, "Ошибка БД", 500); return }
    defer rows.Close()

    var list []User
    for rows.Next() {
        var u User
        _ = rows.Scan(&u.UserID, &u.Email, &u.Role)
        list = append(list, u)
    }

    renderTemplate(w, r,
        map[string]interface{}{
            "Users":   list,
            "IsAdmin": true,           
        },
        "templates/base.html",
        "templates/users.html")
}


